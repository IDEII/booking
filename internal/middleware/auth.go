package middleware

import (
	"context"
	"net/http"
	"strings"

	"booking-service/internal/domain"
	"booking-service/internal/service"
)

type contextKey string

const (
	ClaimsKey contextKey = "claims"
)

func AuthMiddleware(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				domain.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				domain.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization header format")
				return
			}

			token := parts[1]
			claims, err := authService.ValidateToken(token)
			if err != nil {
				domain.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RoleMiddleware(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			if claims == nil {
				domain.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
				return
			}

			for _, role := range allowedRoles {
				if string(claims.Role) == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			domain.WriteError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
		})
	}
}

func GetClaims(ctx context.Context) *service.Claims {
	if ctx == nil {
		return nil
	}
	claims, ok := ctx.Value(ClaimsKey).(*service.Claims)
	if !ok {
		return nil
	}
	return claims
}
