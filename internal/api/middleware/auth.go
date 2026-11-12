package middleware

import (
	"context"
	"net/http"
	"strings"

	"rest-api-scheduler-task/internal/auth"
)

type authKey struct{}

// Auth returns a middleware that checks the JWT token
func Auth(m *auth.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := ""

			if header := r.Header.Get("Authorization"); header != "" {
				parts := strings.SplitN(header, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
					tokenString = parts[1]
				}
			}
			if tokenString == "" {
				if cookie, err := r.Cookie("token"); err == nil {
					tokenString = cookie.Value
				}
			}
			if tokenString == "" {
				writeJSONError(w, http.StatusUnauthorized, "missing token")
				return
			}
			claims, err := m.Parse(tokenString)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			ctx := context.WithValue(r.Context(), authKey{}, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserID(ctx context.Context) string {
	id, _ := ctx.Value(authKey{}).(string)
	return id
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"error":"` + msg + `"}`))
}
