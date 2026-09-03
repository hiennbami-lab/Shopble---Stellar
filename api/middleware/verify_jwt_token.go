package middleware

import (
	"context"
	"errors"
	"shopble/api"
	"shopble/common/comerr"
	"shopble/glib/gauth"
	"strings"

	"github.com/gin-gonic/gin"
)

func verifyToken(ctx context.Context, token string) (_ *gauth.JwtClaimData, err error) {
	if token == "" {
		err = comerr.ErrorTokenExpired.Wrap(errors.New(("token not found")))
		return
	}
	claim, err := gauth.IsValid(token)
	if err != nil {
		err = comerr.ErrorTokenExpired.Wrap(err)
		return
	}
	return claim, nil
}

var VerifyToken gin.HandlerFunc = func(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	if len(token) > 7 && strings.EqualFold(token[:7], "Bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	claim, err := verifyToken(ctx, token)
	if err != nil {
		api.AbortWithErr(ctx, err)
		return
	}
	ctx.Set(gauth.JwtClaimDataKey, &gauth.JwtClaimData{
		Token:            token,
		RegisteredClaims: claim.RegisteredClaims,
		Roles:            claim.Roles,
		Permissions:      claim.Permissions,
	})
	ctx.Next()
}
