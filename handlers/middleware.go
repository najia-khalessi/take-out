package handlers

import (
	"net/http"
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
