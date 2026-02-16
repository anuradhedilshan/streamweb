package auth

import (
	"errors"
	"fmt"
	"strings"
)

func TokenForUser(userID, role string) string {
	return fmt.Sprintf("token:%s:%s", userID, role)
}

func ParseUserToken(token string) (string, string, error) {
	parts := strings.Split(token, ":")
	if len(parts) != 3 || parts[0] != "token" {
		return "", "", errors.New("invalid token")
	}
	return parts[1], parts[2], nil
}
