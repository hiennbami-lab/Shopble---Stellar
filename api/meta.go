package api

import (
	"shopble/glib/gmeta"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Context struct {
	*gin.Context
}

func NewContext(ctx *gin.Context) *Context {
	return &Context{ctx}
}

func (c *Context) AbortWithErr(err error) {
	_ = c.Error(err)
	c.Abort()
}

func (c *Context) Ok(data any) error {
	response := &Response{
		Status: "ok",
		Data:   data,
	}
	c.JSON(http.StatusOK, response)
	return nil
}

type Response struct {
	Status     string        `json:"status"`
	Data       any           `json:"data"`
	Pagination *gmeta.Paging `json:"pagination,omitempty"`
}

func AbortWithErr(ctx *gin.Context, err error) {
	_ = ctx.Error(err)
	ctx.Abort()
}
