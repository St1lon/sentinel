// Package middleware содержит сквозные обработчики HTTP-цепочки.
package middleware

import (
	"context"
	"net/http"
)

type userIDKey struct{}

// WithUserID кладёт идентификатор аутентифицированного пользователя в контекст.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// UserIDFromContext достаёт идентификатор пользователя из контекста.
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey{}).(string)

	return userID, ok && userID != ""
}

// UserIDFromRequest — сокращение для хендлеров.
func UserIDFromRequest(r *http.Request) (string, bool) {
	return UserIDFromContext(r.Context())
}
