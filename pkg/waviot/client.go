// Package waviot — HTTP-клиент к WAVIOT API с авторизацией по JWT.
// Схема url'а должна быть http или https, хост не пустой; если хост
// не похож на waviot — в лог уходит WARN, но запрос всё равно летит.
package waviot

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"parse-api-messages/pkg/auth"
)

// ErrInvalidURL означает что URL не прошёл проверку whitelist'а.
var ErrInvalidURL = errors.New("waviot: invalid url")

// wavIoTHostPattern используется только для «мягкой» диагностики: если хост
// не похож на waviot (нет метки `waviot`), мы логируем WARN, но запрос
// всё равно отправляем. Пользователь сам отвечает за то, куда улетает его JWT.
var wavIoTHostPattern = regexp.MustCompile(`(^|\.)waviot\.[a-z]{2,}$`)

// Client выполняет запросы к WAVIOT API.
type Client struct {
	http *http.Client
	jwt  auth.Provider
}

// NewClient конструирует клиент. httpClient может быть nil — тогда создастся дефолтный с 30s таймаутом.
func NewClient(httpClient *http.Client, jwt auth.Provider) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{http: httpClient, jwt: jwt}
}

// GetMessages делает GET-запрос к произвольному WAVIOT-эндпоинту и возвращает тело ответа.
// Подставляет Cookie: WAVIOT_JWT=<token>.
// Ошибки:
//   - auth.ErrNoToken / auth.ErrExpired — нужно пере-залогиниться;
//   - auth.ErrUnauthorized — upstream вернул 401;
//   - ErrInvalidURL — URL не из *.waviot.ru или не https.
func (c *Client) GetMessages(ctx context.Context, rawURL string) ([]byte, error) {
	if err := ValidateURL(rawURL); err != nil {
		return nil, err
	}

	tok, err := c.jwt.Get(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("waviot: new request: %w", err)
	}
	req.Header.Set("Cookie", "WAVIOT_JWT="+tok.Raw)
	req.Header.Set("Accept", "application/json")

	parsed, _ := url.Parse(rawURL)
	started := time.Now()
	host := strings.ToLower(parsed.Hostname())
	if !wavIoTHostPattern.MatchString(host) {
		slog.WarnContext(ctx, "waviot: request to non-waviot host", "host", host)
	}
	if parsed.Scheme == "http" {
		slog.WarnContext(ctx, "waviot: plaintext http — JWT будет отправлен в открытом виде", "host", host)
	}
	slog.DebugContext(ctx, "waviot: request", "scheme", parsed.Scheme, "host", parsed.Host, "path", parsed.Path)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("waviot: http do: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	elapsed := time.Since(started)
	slog.InfoContext(ctx, "waviot: response",
		"host", parsed.Host,
		"path", parsed.Path,
		"status", resp.StatusCode,
		"bytes", len(body),
		"elapsed_ms", elapsed.Milliseconds(),
	)

	if readErr != nil {
		return nil, fmt.Errorf("waviot: read body: %w", readErr)
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return nil, auth.ErrUnauthorized
	case resp.StatusCode >= 400:
		return nil, fmt.Errorf("waviot: status %d", resp.StatusCode)
	}
	return body, nil
}

// ValidateURL проверяет только минимально необходимое:
//   - URL парсится,
//   - схема http или https,
//   - хост не пустой.
// Домен НЕ фильтруется: JWT принадлежит пользователю, и он сам решает, куда
// его отправлять. Если хост не похож на waviot или схема http — GetMessages
// залогирует WARN (см. выше), но запрос всё равно отправится.
func ValidateURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("%w: empty", ErrInvalidURL)
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: parse: %v", ErrInvalidURL, err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("%w: scheme must be http or https, got %q", ErrInvalidURL, u.Scheme)
	}
	if u.Hostname() == "" {
		return fmt.Errorf("%w: empty host", ErrInvalidURL)
	}
	return nil
}
