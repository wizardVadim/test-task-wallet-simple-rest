package auth_http

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type contextKey int

const userIDKey contextKey = iota

func Authenticate(validator TokenValidator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.Context().Err(); err != nil {
			return
		}

		auth := r.Header.Get("Authorization")
		fields := strings.Fields(auth)
		if len(fields) != 2 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if !strings.EqualFold(fields[0], "Bearer") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		id, err := validator.Validate(fields[1])
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		userID, err := uuid.Parse(id)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if userID == uuid.Nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}
