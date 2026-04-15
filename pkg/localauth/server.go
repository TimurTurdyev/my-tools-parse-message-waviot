// Package localauth поднимает одноразовый HTTP-сервер на 127.0.0.1:RANDOM
// для автоматической доставки JWT из браузера в приложение.
//
// Защита:
//   - слушаем только 127.0.0.1 (loopback);
//   - случайный порт;
//   - случайный nonce (16 байт crypto/rand, 32 hex) — только клиент,
//     знающий nonce, может доставить токен;
//   - принимаем ровно один валидный POST и сразу останавливаемся;
//   - auto-timeout (по умолчанию 5 минут), чтобы не оставлять открытый listener.
package localauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ErrAlreadyRunning возвращается при попытке стартовать сервер дважды.
var ErrAlreadyRunning = errors.New("localauth: server is already running")

// Session — активная попытка получения JWT.
type Session struct {
	Port  int
	Nonce string
}

// OnToken — колбэк, вызываемый при успешной доставке JWT.
// Возвращённая ошибка сериализуется и летит клиенту в body ответа.
type OnToken func(ctx context.Context, raw string) error

// Server — одноразовый loopback HTTP listener.
// Zero value не валиден — используйте New().
type Server struct {
	onToken OnToken
	timeout time.Duration

	mu      sync.Mutex
	srv     *http.Server
	session *Session
	stopCh  chan struct{}
}

// New создаёт сервер. timeout = сколько держим listener до автостопа.
func New(onToken OnToken, timeout time.Duration) *Server {
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	return &Server{onToken: onToken, timeout: timeout}
}

// Running — сервер активен?
func (s *Server) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.srv != nil
}

// Start поднимает listener. Возвращает Session c портом и nonce.
// Если уже запущен — ErrAlreadyRunning.
func (s *Server) Start() (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.srv != nil {
		return Session{}, ErrAlreadyRunning
	}

	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return Session{}, fmt.Errorf("localauth: rand: %w", err)
	}
	nonce := hex.EncodeToString(nonceBytes)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return Session{}, fmt.Errorf("localauth: listen: %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	session := &Session{Port: port, Nonce: nonce}
	stopCh := make(chan struct{})

	mux.HandleFunc("/jwt", func(w http.ResponseWriter, r *http.Request) {
		// CORS headers — на всякий случай (no-cors запрос их не читает, но не повредит).
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Nonce")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		got := r.URL.Query().Get("nonce")
		if got == "" {
			got = r.Header.Get("X-Nonce")
		}
		if got != nonce {
			slog.Warn("localauth: nonce mismatch", "got_len", len(got))
			http.Error(w, "nonce mismatch", http.StatusForbidden)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
		if err != nil {
			http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}
		raw := strings.TrimSpace(string(body))
		if raw == "" {
			http.Error(w, "empty token", http.StatusBadRequest)
			return
		}
		if err := s.onToken(r.Context(), raw); err != nil {
			slog.Warn("localauth: onToken rejected", "err", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.Info("localauth: token delivered", "len", len(raw))
		w.WriteHeader(http.StatusNoContent)

		// После первой успешной доставки останавливаем сервер.
		go s.Stop()
	})

	s.srv = srv
	s.session = session
	s.stopCh = stopCh

	go func() {
		slog.Info("localauth: listening", "addr", ln.Addr().String())
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Warn("localauth: serve error", "err", err)
		}
		close(stopCh)
	}()

	go func() {
		select {
		case <-time.After(s.timeout):
			slog.Info("localauth: timeout reached, stopping")
			s.Stop()
		case <-stopCh:
		}
	}()

	return *session, nil
}

// Stop останавливает listener, если он активен.
func (s *Server) Stop() {
	s.mu.Lock()
	srv := s.srv
	s.srv = nil
	s.session = nil
	s.mu.Unlock()
	if srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
