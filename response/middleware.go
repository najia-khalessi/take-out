package response

import (
	"context"
	"github.com/google/uuid"
	"log"
	"net/http"
	"time"
)

type contextKey string

const (
	RequestIDKey contextKey = "request_id"
)

// ResponseMiddleware 响应中间件：添加统一响应头、请求ID、日志记录等功能
func ResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// 生成请求ID
		requestID := uuid.New().String()
		
		// 将请求ID放入context
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		r = r.WithContext(ctx)
		
		// 创建响应包装器
		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		// 设置统一响应头
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		
		// 处理请求
		next.ServeHTTP(wrappedWriter, r)
		
		// 记录请求日志
		duration := time.Since(start)
		log.Printf("[%s] %s %s - %d (%v)", 
			requestID, 
			r.Method, 
			r.URL.Path, 
			wrappedWriter.statusCode, 
			duration)
	})
}

// LogMiddleware 日志中间件
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		requestID := GetRequestID(r)
		if requestID == "" {
			requestID = "unknown"
		}
		
		log.Printf("[REQUEST START] %s %s %s - RequestID: %s", 
			r.Method, 
			r.URL.Path, 
			r.RemoteAddr, 
			requestID)
		
		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrappedWriter, r)
		
		duration := time.Since(start)
		
		log.Printf("[REQUEST END] %s %s - %d - %v", 
			r.Method, 
			r.URL.Path, 
			wrappedWriter.statusCode, 
			duration)
	})
}

// GetRequestID 从context获取请求ID
func GetRequestID(r *http.Request) string {
	if requestID := r.Context().Value(RequestIDKey); requestID != nil {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

// responseWriter 包装http.ResponseWriter以捕获状态码
-type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// CORSMiddleware CORS中间件
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// RecoverMiddleware 异常恢复中间件
func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC RECOVERED] %s %s - Error: %v RequestID: %s", 
					r.Method, 
					r.URL.Path, 
					err,
					GetRequestID(r))
				
				ServerError(w, nil)
			}
		}()
		
		next.ServeHTTP(w, r)
	})
}