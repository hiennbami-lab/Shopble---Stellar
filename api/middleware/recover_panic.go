package middleware

import (
	"shopble/common/comlog"
	"net/http"
	"runtime/debug"

	"cloud.google.com/go/logging"
	"github.com/gin-gonic/gin"
)

var RecoverPanic gin.RecoveryFunc = func(ctx *gin.Context, err any) {
	defer func() {
		var (
			logger = comlog.GetLog()
		)
		logger.Log(logging.Entry{
			HTTPRequest: &logging.HTTPRequest{
				Request: ctx.Request,
			},
			Payload: map[string]any{
				"error": err,
				"stack": string(debug.Stack()),
			},
			Severity: logging.Error,
		})
	}()
	ctx.String(http.StatusInternalServerError, "something went wrong")
}
