// Package response provides gin-idiomatic helpers for the standard JSON contract.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// successResponse is the JSON shape for a successful request.
//
//	{ "data": <any>, "message": "optional string" }
type successResponse struct {
	Data    any    `json:"data"`
	Message string `json:"message,omitempty"`
}

// errorDetail carries per-field validation details.
type errorDetail struct {
	Field string `json:"field,omitempty"`
	Issue string `json:"issue,omitempty"`
}

// errorResponse is the JSON shape for an error.
//
//	{ "error": { "code": "UPPER_SNAKE_CODE", "message": "human readable", "details": [...] } }
type errorResponse struct {
	Error struct {
		Code    string        `json:"code"`
		Message string        `json:"message"`
		Details []errorDetail `json:"details,omitempty"`
	} `json:"error"`
}

// listMeta carries pagination metadata.
type listMeta struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

// List is the JSON shape for a paginated list response.
//
//	{ "data": [...], "meta": { "page": 1, "page_size": 20, "total": 137 } }
type listResponse struct {
	Data any      `json:"data"`
	Meta listMeta `json:"meta"`
}

// Success writes a 2xx success response.
func Success(c *gin.Context, statusCode int, data any, message string) {
	c.JSON(statusCode, successResponse{Data: data, Message: message})
}

// Error writes an error response.
func Error(c *gin.Context, statusCode int, code string, message string, details []errorDetail) {
	c.JSON(statusCode, errorResponse{
		Error: struct {
			Code    string        `json:"code"`
			Message string        `json:"message"`
			Details []errorDetail `json:"details,omitempty"`
		}{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// List writes a paginated list response.
func List(c *gin.Context, data any, page int, pageSize int, total int64) {
	c.JSON(http.StatusOK, listResponse{
		Data: data,
		Meta: listMeta{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	})
}
