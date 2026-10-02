package middleware

import (
	"context"
	"net/http"
)

type userIDKey struct{}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey{}).(string)

	return userID, ok && userID != ""
}

func UserIDFromRequest(r *http.Request) (string, bool) {
	return UserIDFromContext(r.Context())
}
