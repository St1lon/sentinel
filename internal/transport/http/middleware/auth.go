package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/St1lon/sentinel/internal/transport/http/apierrors"
	"github.com/St1lon/sentinel/internal/transport/http/dto"
)

const bearerPrefix = "Bearer "

type TokenParser interface {
	ParseUserID(token string) (string, error)
}

func Auth(parser TokenParser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")

			if !strings.HasPrefix(header, bearerPrefix) {
				writeUnauthorized(w)

				return
			}

			userID, err := parser.ParseUserID(strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix)))
			if err != nil {
				writeUnauthorized(w)

				return
			}

			next.ServeHTTP(w, r.WithContext(WithUserID(r.Context(), userID)))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.WriteHeader(http.StatusUnauthorized)

	_ = json.NewEncoder(w).Encode(dto.ErrorResponse{
		Code:        apierrors.CodeUnauthenticated,
		Description: "valid bearer token is required",
	})
}
