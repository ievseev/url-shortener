package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

const CookieName = "user_id"

var (
	errInvalidCookieValue = errors.New("invalid cookie value")
	secretKey             = []byte("url-shortener-auth-secret")
)

type contextKey struct{}

func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, contextKey{}, userID)
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(contextKey{}).(string)
	if !ok || userID == "" {
		return "", false
	}

	return userID, true
}

func GenerateUserID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

func EncodeCookieValue(userID string) string {
	encodedUserID := base64.RawURLEncoding.EncodeToString([]byte(userID))
	signature := sign([]byte(userID))

	return encodedUserID + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func DecodeCookieValue(value string) (string, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return "", errInvalidCookieValue
	}

	userID, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", errInvalidCookieValue
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", errInvalidCookieValue
	}

	expectedSignature := sign(userID)
	if !hmac.Equal(signature, expectedSignature) {
		return "", errInvalidCookieValue
	}

	return string(userID), nil
}

func sign(data []byte) []byte {
	mac := hmac.New(sha256.New, secretKey)
	mac.Write(data)

	return mac.Sum(nil)
}
