package handlers

import (
	"context"
	"net/http"
	"strings"
	"take-out/logging"
	"time"

	"github.com/sirupsen/logrus"
)

// LoggingMiddleware logs the incoming HTTP request
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Call the next handler
		next.ServeHTTP(w, r)

		logging.Info("Request processed", logrus.Fields{
			"method":   r.Method,
			"url":      r.URL.String(),
			"duration": time.Since(start).String(),
		})
	})
}

// AuthMiddleware validates the access token
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization Header", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, "Invalid Authorization Header format", http.StatusUnauthorized)
			return
		}

		claims, err := ParseToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid access token", http.StatusUnauthorized)
			return
		}

		if claims.Type != "access" {
			http.Error(w, "Invalid access token type", http.StatusUnauthorized)
			return
		}

		// Add user/shop ID to context
		var ctx context.Context
		if claims.ShopID > 0 {
			ctx = context.WithValue(r.Context(), "shopID", claims.ShopID)
		} else {
			ctx = context.WithValue(r.Context(), "userID", claims.UserID)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
