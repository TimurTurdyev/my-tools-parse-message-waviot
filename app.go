package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"parse-api-messages/pkg/auth"
	"parse-api-messages/pkg/localauth"
	"parse-api-messages/pkg/waviot"
)

// Login URL — куда открывается внешний браузер для получения JWT.
const wavIoTLoginURL = "https://lk.waviot.ru/"

// App — корневая структура, экспортируемая во фронтенд через Wails bindings.
type App struct {
	ctx   context.Context
	jwt   auth.Provider
	wvc   *waviot.Client
	local *localauth.Server
	clk   func() time.Time
}

// NewApp собирает App с переданными зависимостями.
func NewApp(jwt auth.Provider, wvc *waviot.Client) *App {
	a := &App{
		jwt: jwt,
		wvc: wvc,
		clk: time.Now,
	}
	a.local = localauth.New(a.handleLocalToken, 5*time.Minute)
	return a
}

// handleLocalToken вызывается localauth-сервером при получении JWT.
// Сохраняет его через Provider и эмитит событие во фронт.
func (a *App) handleLocalToken(_ context.Context, raw string) error {
	if err := a.jwt.Set(a.ctx, raw); err != nil {
		return err
	}
	runtime.EventsEmit(a.ctx, "jwt:received")
	slog.Info("app: jwt received via local auth server")
	return nil
}

// startup вызывается Wails при запуске окна.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	slog.Info("app started")
}

// ---------- Методы, экспортируемые в Vue ----------

// AuthStatus возвращает текущее состояние JWT: "valid" | "expired" | "missing"
// и unix-секунды момента истечения (0 если отсутствует).
type AuthStatusResponse struct {
	Status     string `json:"status"`
	ExpiresAt  int64  `json:"expiresAt"`
	LoginURL   string `json:"loginUrl"`
	StoredPath string `json:"storedPath,omitempty"`
}

// AuthStatus отдаёт состояние провайдера авторизации.
func (a *App) AuthStatus() (AuthStatusResponse, error) {
	status, exp, err := auth.DescribeStatus(a.ctx, a.jwt, a.clk())
	if err != nil {
		slog.Warn("app: auth status failed", "err", err)
		return AuthStatusResponse{}, err
	}
	resp := AuthStatusResponse{
		Status:   string(status),
		LoginURL: wavIoTLoginURL,
	}
	if !exp.IsZero() {
		resp.ExpiresAt = exp.Unix()
	}
	if fp, ok := a.jwt.(*auth.FileProvider); ok {
		resp.StoredPath = fp.Path()
	}
	slog.Debug("app: auth status", "status", status, "exp", exp.Format(time.RFC3339))
	return resp, nil
}

// OpenLoginInBrowser открывает страницу логина WAVIOT в системном браузере.
// Пользователь логинится, копирует WAVIOT_JWT cookie и вставляет его обратно в UI
// через SetJWTManually.
func (a *App) OpenLoginInBrowser() error {
	slog.Info("app: opening login url in browser", "url", wavIoTLoginURL)
	runtime.BrowserOpenURL(a.ctx, wavIoTLoginURL)
	return nil
}

// SetJWTManually принимает JWT, введённый пользователем вручную.
// Парсит, валидирует exp, сохраняет через provider.
func (a *App) SetJWTManually(raw string) (AuthStatusResponse, error) {
	if err := a.jwt.Set(a.ctx, raw); err != nil {
		slog.Warn("app: manual jwt rejected", "err", err)
		return AuthStatusResponse{}, err
	}
	slog.Info("app: manual jwt accepted")
	return a.AuthStatus()
}

// ClearJWT удаляет сохранённый токен.
func (a *App) ClearJWT() error {
	if err := a.jwt.Clear(a.ctx); err != nil {
		slog.Warn("app: clear jwt failed", "err", err)
		return err
	}
	return nil
}

// JWTCaptureSession — информация для фронта о запущенном listener'е.
type JWTCaptureSession struct {
	URL     string `json:"url"`     // полный URL, куда снипет должен POST'ить токен
	Nonce   string `json:"nonce"`   // одноразовый токен для защиты
	Snippet string `json:"snippet"` // готовый JS, который можно скопировать в консоль браузера
}

// StartJWTCapture поднимает локальный HTTP-сервер на 127.0.0.1:RANDOM и
// возвращает данные для вставки JS-снипета в консоль браузера WAVIOT.
func (a *App) StartJWTCapture() (JWTCaptureSession, error) {
	session, err := a.local.Start()
	if err != nil {
		slog.Warn("app: start jwt capture failed", "err", err)
		return JWTCaptureSession{}, err
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/jwt?nonce=%s", session.Port, session.Nonce)
	snippet := buildBrowserSnippet(url)
	slog.Info("app: jwt capture started", "port", session.Port)
	return JWTCaptureSession{URL: url, Nonce: session.Nonce, Snippet: snippet}, nil
}

// StopJWTCapture останавливает listener, если он активен.
func (a *App) StopJWTCapture() error {
	a.local.Stop()
	slog.Info("app: jwt capture stopped")
	return nil
}

// buildBrowserSnippet собирает one-liner для консоли браузера.
// Снипет:
//  1) пробует прочитать WAVIOT_JWT из document.cookie;
//  2) если не вышло (HttpOnly) — открывает prompt для ручной вставки;
//  3) POST'ит токен на локальный listener приложения;
//  4) показывает alert о результате.
func buildBrowserSnippet(deliveryURL string) string {
	// Для читабельности в UI пишем многострочно, но это валидный JS,
	// который можно скопировать и вставить в DevTools Console.
	const tmpl = `(async () => {
  const deliveryURL = %q;
  let jwt = (document.cookie.match(/WAVIOT_JWT=([^;]+)/) || [])[1];
  if (!jwt) {
    jwt = prompt("WAVIOT_JWT не виден в document.cookie (скорее всего HttpOnly). Откройте DevTools > Application > Cookies, найдите WAVIOT_JWT и вставьте его значение сюда:");
  }
  if (!jwt) { console.warn("JWT не получен"); return; }
  jwt = jwt.trim();
  try {
    const res = await fetch(deliveryURL, { method: "POST", body: jwt, headers: { "Content-Type": "text/plain" }, mode: "cors" });
    if (res.ok || res.status === 204) {
      console.log("✅ JWT отправлен в приложение");
      alert("JWT отправлен в Parse API Messages");
    } else {
      const t = await res.text();
      console.error("❌ Ошибка доставки:", res.status, t);
      alert("Ошибка доставки JWT: " + res.status + " " + t);
    }
  } catch (e) {
    console.error("❌ Не удалось соединиться с приложением:", e);
    alert("Не удалось соединиться с приложением. Убедитесь, что в Parse API Messages нажата кнопка «Автопередача JWT».");
  }
})();`
	return fmt.Sprintf(tmpl, deliveryURL)
}

// GetMessages — основной метод: делает GET к URL с текущим JWT.
// Возвращает тело ответа как строку. Ошибки прокидываются как rejected Promise во фронт.
func (a *App) GetMessages(rawURL string) (string, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	body, err := a.wvc.GetMessages(ctx, rawURL)
	if err != nil {
		// Ошибки авторизации нормализуем, чтобы фронт мог показать правильный CTA.
		switch {
		case errors.Is(err, auth.ErrNoToken), errors.Is(err, auth.ErrExpired), errors.Is(err, auth.ErrUnauthorized):
			slog.Warn("app: GetMessages unauthorized", "err", err)
		case errors.Is(err, waviot.ErrInvalidURL):
			slog.Warn("app: GetMessages invalid url", "err", err)
		default:
			slog.Error("app: GetMessages failed", "err", err)
		}
		return "", err
	}
	return string(body), nil
}
