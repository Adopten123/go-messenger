package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDKey contextKey = "UserID"

func (h *UserHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := ""
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			headerParts := strings.Split(authHeader, " ")
			if len(headerParts) == 2 && headerParts[0] == "Bearer" {
				tokenString = headerParts[1]
			}
		}

		if tokenString == "" {
			tokenString = r.URL.Query().Get("token")
		}

		if tokenString == "" {
			http.Error(w, "unauthorized: token required", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(h.tokenSecret), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "invalid token claims", http.StatusUnauthorized)
			return
		}

		userID, ok := claims["user_id"].(string)
		if !ok {
			http.Error(w, "user_id not found in token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
