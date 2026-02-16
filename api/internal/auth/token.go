package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"
)

type claims struct {
	Sub string `json:"sub"`
	Rol string `json:"rol"`
	Exp int64  `json:"exp"`
	Typ string `json:"typ"`
}

func secret() []byte {
	s := os.Getenv("STREAMWEB_JWT_SECRET")
	if s == "" {
		s = "dev-secret-change-me"
	}
	return []byte(s)
}

func sign(payload string) string {
	h := hmac.New(sha256.New, secret())
	_, _ = h.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func TokenForUser(userID, role string) string {
	hdr := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	cl := claims{Sub: userID, Rol: role, Exp: time.Now().Add(24 * time.Hour).Unix(), Typ: "access"}
	b, _ := json.Marshal(cl)
	payload := base64.RawURLEncoding.EncodeToString(b)
	sig := sign(hdr + "." + payload)
	return hdr + "." + payload + "." + sig
}

func ParseUserToken(token string) (string, string, error) {
	parts := strings.Split(token, ".")
	if len(parts) == 3 {
		payload := parts[0] + "." + parts[1]
		expected := sign(payload)
		if !hmac.Equal([]byte(expected), []byte(parts[2])) {
			return "", "", errors.New("invalid signature")
		}
		raw, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return "", "", errors.New("invalid payload")
		}
		var c claims
		if err := json.Unmarshal(raw, &c); err != nil {
			return "", "", errors.New("invalid claims")
		}
		if c.Exp < time.Now().Unix() {
			return "", "", errors.New("expired")
		}
		if c.Sub == "" || c.Rol == "" {
			return "", "", errors.New("invalid claims")
		}
		return c.Sub, c.Rol, nil
	}
	// backward compatibility for old tokens: token:user:role
	legacy := strings.Split(token, ":")
	if len(legacy) == 3 && legacy[0] == "token" {
		return legacy[1], legacy[2], nil
	}
	return "", "", errors.New("invalid token")
}
