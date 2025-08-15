package response

import (
	"encoding/json"
	"net/http"
	"time"
)

// APIResponse 统一API响应结构
type APIResponse struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Timestamp int64       `json:"timestamp"`
	RequestID string      `json:"requestId,omitempty"`
}

// APIError 统一错误结构
type APIError struct {
	Type    string      `json:"type"`
	Details interface{} `json:"details"`
	Path    string      `json:"path,omitempty"`
}

// Success 返回成功响应
func Success(w http.ResponseWriter, data interface{}, message string) {
	SuccessWithCode(w, data, message, http.StatusOK)
}

// SuccessWithCode 返回自定义状态码的成功响应
func SuccessWithCode(w http.ResponseWriter, data interface{}, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	
	response := APIResponse{
		Code:      code,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}
	
	json.NewEncoder(w).Encode(response)
}

// Error 返回错误响应
func Error(w http.ResponseWriter, message string, code int) {
	ErrorWithDetails(w, message, code, nil, "")
}

// ErrorWithDetails 返回带详细信息的错误响应
func ErrorWithDetails(w http.ResponseWriter, message string, code int, details interface{}, errType string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	
	if errType == "" {
		switch code {
		case http.StatusBadRequest:
			errType = "INVALID_REQUEST"
		case http.StatusUnauthorized:
			errType = "UNAUTHORIZED"
		case http.StatusForbidden:
			errType = "FORBIDDEN"
		case http.StatusNotFound:
			errType = "NOT_FOUND"
		case http.StatusUnprocessableEntity:
			errType = "VALIDATION_ERROR"
		case http.StatusInternalServerError:
			errType = "INTERNAL_SERVER_ERROR"
		default:
			errType = "UNKNOWN_ERROR"
		}
	}
	
	response := APIResponse{
		Code:      code,
		Message:   message,
		Error:     &APIError{
			Type:    errType,
			Details: details,
		},
		Timestamp: time.Now().UnixMilli(),
	}
	
	json.NewEncoder(w).Encode(response)
}

// ValidationError 返回验证错误响应
func ValidationError(w http.ResponseWriter, details string, field string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	
	errorDetails := map[string]interface{}{
		"field":   field,
		"message": details,
	}
	
	response := APIResponse{
		Code:    http.StatusUnprocessableEntity,
		Message: "输入参数验证失败",
		Error: &APIError{
			Type:    "VALIDATION_ERROR",
			Details: errorDetails,
		},
		Timestamp: time.Now().UnixMilli(),
	}
	
	json.NewEncoder(w).Encode(response)
}

// ServerError 返回服务器内部错误响应
func ServerError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	
	response := APIResponse{
		Code:    http.StatusInternalServerError,
		Message: "服务器内部错误",
		Error: &APIError{
			Type:    "INTERNAL_SERVER_ERROR",
			Details: err.Error(),
		},
		Timestamp: time.Now().UnixMilli(),
	}
	
	json.NewEncoder(w).Encode(response)
}

// NotFound 返回资源未找到响应
func NotFound(w http.ResponseWriter, message string) {
	if message == "" {
		message = "请求的资源不存在"
	}
	Error(w, message, http.StatusNotFound)
}

// Unauthorized 返回未授权响应
func Unauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		message = "身份认证失败"
	}
	Error(w, message, http.StatusUnauthorized)
}

// Forbidden 返回权限不足响应
func Forbidden(w http.ResponseWriter, message string) {
	if message == "" {
		message = "权限不足"
	}
	Error(w, message, http.StatusForbidden)
}

// BadRequest 返回错误请求响应
func BadRequest(w http.ResponseWriter, message string, details interface{}) {
	ErrorWithDetails(w, message, http.StatusBadRequest, details, "BAD_REQUEST")
}

// Created 返回创建成功响应
func Created(w http.ResponseWriter, data interface{}, message string) {
	if message == "" {
		message = "资源创建成功"
	}
	SuccessWithCode(w, data, message, http.StatusCreated)
}