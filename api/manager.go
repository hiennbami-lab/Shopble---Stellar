package api

import (
	"shopble/config"

	"github.com/gin-gonic/gin"
)

func New() *gin.Engine {
	if config.ReleaseMode == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	return gin.New()
}

type ExecFunc func(ctx *Context) error

func HandlerFunc(executeFn ExecFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		err := executeFn(NewContext(ctx))
		if err != nil {
			_ = ctx.Error(err)
			return
		}
	}
}
