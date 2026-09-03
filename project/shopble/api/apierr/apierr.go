package apierr

import (
	"net/http"

	"shopble/api"

	"github.com/gin-gonic/gin"
)

// Error codes — phải khớp với FE spec.
const (
	CodeDataInvalid   = "DATA_INVALID"
	CodeNotFound      = "NOT_FOUND"
	CodeConflict      = "CONFLICT"
	CodeInvalidStatus = "INVALID_STATUS"
	CodeInternal      = "INTERNAL_ERROR"
)

// Abort — thoát gin chain với JSON {status:"error", data:{error_code, message, ...extra}}.
func Abort(c *gin.Context, httpStatus int, errCode, message string, extra ...gin.H) {
	data := gin.H{
		"error_code": errCode,
		"message":    message,
	}
	for _, m := range extra {
		for k, v := range m {
			data[k] = v
		}
	}
	c.AbortWithStatusJSON(httpStatus, &api.Response{
		Status: "error",
		Data:   data,
	})
}

func BadRequest(c *gin.Context, errCode, msg string, extra ...gin.H) {
	Abort(c, http.StatusBadRequest, errCode, msg, extra...)
}

func Conflict(c *gin.Context, errCode, msg string, extra ...gin.H) {
	Abort(c, http.StatusConflict, errCode, msg, extra...)
}

func NotFound(c *gin.Context, msg string) {
	Abort(c, http.StatusNotFound, CodeNotFound, msg)
}

func Internal(c *gin.Context, msg string) {
	Abort(c, http.StatusInternalServerError, CodeInternal, msg)
}
