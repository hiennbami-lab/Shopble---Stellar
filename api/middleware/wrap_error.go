package middleware

import (
	"context"
	"shopble/api"
	"shopble/common/comerr"
	"shopble/common/comlog"
	"shopble/common/comutils"
	"net/http"

	"cloud.google.com/go/logging"
	"github.com/gin-gonic/gin"
)

type ErrorRespones struct {
	ErrorCode string `json:"error_code"`
	Msg       string `json:"message"`
}

func wrapError(ctx context.Context, req *http.Request, err error) (code int, response *api.Response) {
	var (
		errCode  string
		httpCode int
	)
	switch {
	case comutils.IsSameError(err, comerr.ErrorDataInvalid):
		errCode = comerr.ErrorDataInvalid.Code()
		httpCode = http.StatusBadRequest
	case comutils.IsSameError(err, comerr.ErrorNotFound):
		errCode = comerr.ErrorNotFound.Code()
		httpCode = http.StatusNotFound
	case comutils.IsSameError(err, comerr.ErrorTokenExpired):
		errCode = comerr.ErrorTokenExpired.Code()
		httpCode = http.StatusUnauthorized
	default:
		errCode = comerr.ErrorServerUnknown.Code()
		httpCode = http.StatusInternalServerError
	}
	var (
		msg = ErrorRespones{
			ErrorCode: errCode,
			Msg:       err.Error(),
		}
	)
	if ourError, ok := err.(comerr.Failure); ok {
		defer func() {
			var (
				logger = comlog.GetLog()
			)
			logger.Log(logging.Entry{
				HTTPRequest: &logging.HTTPRequest{
					Request: req,
				},
				Payload: map[string]any{
					"error":      ourError.Error(),
					"stacktrace": ourError.Stack(),
					"data":       ourError.Data().String(),
				},
				Severity: logging.Error,
			})
		}()
		msg.Msg = ourError.Error()
		return httpCode, &api.Response{
			Data:   msg,
			Status: "error",
		}
	}
	return http.StatusExpectationFailed, &api.Response{
		Data:   msg,
		Status: "error",
	}
}

var WrapError gin.HandlerFunc = func(ctx *gin.Context) {
	ctx.Next()
	for _, err := range ctx.Errors {
		code, errorResponse := wrapError(ctx, ctx.Request, err.Unwrap())
		ctx.JSON(code, errorResponse)
		return
	}
}
