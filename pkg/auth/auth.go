// Package auth хранит JWT-токен для доступа к WAVIOT API и валидирует его
// срок действия. Подпись токена мы не проверяем — это делает сам сервис;
// нам достаточно распарсить поле exp, чтобы понимать, когда токен истёк.
package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Sentinel errors клиенты различают через errors.Is.
var (
	ErrNoToken      = errors.New("auth: no token stored")
	ErrExpired      = errors.New("auth: token expired")
	ErrInvalidToken = errors.New("auth: invalid token format")
	ErrUnauthorized = errors.New("auth: unauthorized (upstream 401)")
)

// Token — распарсенный JWT. Raw хранит исходную строку; Exp — время истечения.
type Token struct {
	Raw string
	Exp time.Time
}

// Expired возвращает true если токен просрочен относительно now.
func (t Token) Expired(now time.Time) bool {
	return !t.Exp.IsZero() && !now.Before(t.Exp)
}

// Valid возвращает true если токен не пустой и не просрочен.
func (t Token) Valid(now time.Time) bool {
	return t.Raw != "" && !t.Expired(now)
}

// Parse декодирует JWT и извлекает поле exp из payload.
// Подпись НЕ проверяется.
func Parse(raw string) (Token, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Token{}, ErrInvalidToken
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return Token{}, fmt.Errorf("%w: expected 3 segments, got %d", ErrInvalidToken, len(parts))
	}
	payloadBytes, err := decodeSegment(parts[1])
	if err != nil {
		return Token{}, fmt.Errorf("%w: payload decode: %v", ErrInvalidToken, err)
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return Token{}, fmt.Errorf("%w: payload json: %v", ErrInvalidToken, err)
	}
	var exp time.Time
	if claims.Exp > 0 {
		exp = time.Unix(claims.Exp, 0)
	}
	return Token{Raw: raw, Exp: exp}, nil
}

// decodeSegment поддерживает и standard, и raw url-safe base64 варианты.
func decodeSegment(seg string) ([]byte, error) {
	if b, err := base64.RawURLEncoding.DecodeString(seg); err == nil {
		return b, nil
	}
	if b, err := base64.URLEncoding.DecodeString(seg); err == nil {
		return b, nil
	}
	if b, err := base64.RawStdEncoding.DecodeString(seg); err == nil {
		return b, nil
	}
	return base64.StdEncoding.DecodeString(seg)
}

// Provider хранит и выдаёт токен потребителям (HTTP-клиенту).
// Реализации обязаны быть потокобезопасны.
type Provider interface {
	// Get возвращает валидный (не просроченный) токен.
	// ErrNoToken — токен не задан, ErrExpired — задан, но истёк.
	Get(ctx context.Context) (Token, error)
	// Set сохраняет сырой JWT, предварительно распарсив его.
	Set(ctx context.Context, raw string) error
	// Clear удаляет сохранённый токен.
	Clear(ctx context.Context) error
}

// Status — удобное представление состояния провайдера для UI.
type Status string

const (
	StatusMissing Status = "missing"
	StatusValid   Status = "valid"
	StatusExpired Status = "expired"
)

// DescribeStatus возвращает текущий статус провайдера — для фронта.
func DescribeStatus(ctx context.Context, p Provider, now time.Time) (Status, time.Time, error) {
	tok, err := p.Get(ctx)
	switch {
	case errors.Is(err, ErrNoToken):
		return StatusMissing, time.Time{}, nil
	case errors.Is(err, ErrExpired):
		return StatusExpired, tok.Exp, nil
	case err != nil:
		return StatusMissing, time.Time{}, err
	case tok.Valid(now):
		return StatusValid, tok.Exp, nil
	default:
		return StatusMissing, time.Time{}, nil
	}
}
