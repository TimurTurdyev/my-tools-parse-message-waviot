package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// storedToken — сериализуемое представление на диске.
type storedToken struct {
	Raw string `json:"raw"`
	Exp int64  `json:"exp"`
}

// FileProvider хранит токен в JSON-файле под os.UserConfigDir.
// Запись атомарная (tmp + rename), права 0600.
type FileProvider struct {
	path string
	now  func() time.Time

	mu  sync.RWMutex
	tok Token
}

// NewFileProvider создаёт провайдер с путём <UserConfigDir>/parse-api-messages/jwt.json.
// Если файл существует — подгружает токен при создании.
func NewFileProvider() (*FileProvider, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("auth: resolve user config dir: %w", err)
	}
	appDir := filepath.Join(dir, "parse-api-messages")
	if err := os.MkdirAll(appDir, 0o700); err != nil {
		return nil, fmt.Errorf("auth: create config dir: %w", err)
	}
	p := &FileProvider{
		path: filepath.Join(appDir, "jwt.json"),
		now:  time.Now,
	}
	if err := p.load(); err != nil && !errors.Is(err, ErrNoToken) {
		// Ошибки чтения логируем, но не фейлим конструктор —
		// пользователь сможет ввести токен заново.
		slog.Warn("auth: failed to load stored token", "err", err)
	}
	return p, nil
}

// Path возвращает путь к файлу хранилища (для диагностики).
func (p *FileProvider) Path() string { return p.path }

func (p *FileProvider) load() error {
	data, err := os.ReadFile(p.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNoToken
		}
		return fmt.Errorf("auth: read %s: %w", p.path, err)
	}
	var st storedToken
	if err := json.Unmarshal(data, &st); err != nil {
		return fmt.Errorf("auth: parse stored token: %w", err)
	}
	if st.Raw == "" {
		return ErrNoToken
	}
	tok := Token{Raw: st.Raw, Exp: time.Unix(st.Exp, 0)}
	p.mu.Lock()
	p.tok = tok
	p.mu.Unlock()
	slog.Info("auth: token loaded", "exp", tok.Exp.Format(time.RFC3339))
	return nil
}

// Get implements Provider.
func (p *FileProvider) Get(_ context.Context) (Token, error) {
	p.mu.RLock()
	tok := p.tok
	p.mu.RUnlock()
	if tok.Raw == "" {
		return Token{}, ErrNoToken
	}
	if tok.Expired(p.now()) {
		return tok, ErrExpired
	}
	return tok, nil
}

// Set implements Provider. Парсит raw, проверяет формат, атомарно пишет на диск.
func (p *FileProvider) Set(_ context.Context, raw string) error {
	tok, err := Parse(raw)
	if err != nil {
		return err
	}
	st := storedToken{Raw: tok.Raw, Exp: tok.Exp.Unix()}
	data, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("auth: marshal token: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(p.path), "jwt-*.json.tmp")
	if err != nil {
		return fmt.Errorf("auth: create tmp: %w", err)
	}
	tmpPath := tmp.Name()
	// Выставить права 0600 до записи, чтобы никакое мгновенное окно не дало read access.
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("auth: chmod tmp: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("auth: write tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("auth: close tmp: %w", err)
	}
	if err := os.Rename(tmpPath, p.path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("auth: rename tmp: %w", err)
	}

	p.mu.Lock()
	p.tok = tok
	p.mu.Unlock()
	slog.Info("auth: token stored", "exp", tok.Exp.Format(time.RFC3339), "len", len(tok.Raw))
	return nil
}

// Clear implements Provider.
func (p *FileProvider) Clear(_ context.Context) error {
	p.mu.Lock()
	p.tok = Token{}
	p.mu.Unlock()
	if err := os.Remove(p.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("auth: remove %s: %w", p.path, err)
	}
	slog.Info("auth: token cleared")
	return nil
}
