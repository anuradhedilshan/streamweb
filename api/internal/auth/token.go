package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func TokenForUser(userID, role, secret string, ttl time.Duration) string {
	exp := time.Now().Add(ttl).Unix()
	payload := fmt.Sprintf("%s:%s:%d", userID, role, exp)
	sig := sign(payload, secret)
	return fmt.Sprintf("at.%s.%s", base64.RawURLEncoding.EncodeToString([]byte(payload)), sig)
}

func ParseUserToken(token, secret string) (string, string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != "at" {
		return "", "", errors.New("invalid token format")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", errors.New("invalid token payload")
	}
	payload := string(payloadBytes)
	if !hmac.Equal([]byte(parts[2]), []byte(sign(payload, secret))) {
		return "", "", errors.New("invalid token signature")
	}
	fields := strings.Split(payload, ":")
	if len(fields) != 3 {
		return "", "", errors.New("invalid token fields")
	}
	exp, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return "", "", errors.New("invalid token expiry")
	}
	if time.Now().Unix() > exp {
		return "", "", errors.New("token expired")
	}
	return fields[0], fields[1], nil
}

func sign(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
