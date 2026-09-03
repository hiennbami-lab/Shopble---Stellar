package gauth

import (
	"context"

	"github.com/golang-jwt/jwt/v4"
)

var (
	JwtClaimDataKey string = "claim_key"
)

// TODO: support refresh token
type JwtClaimData struct {
	Token       string   `json:"token"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	*jwt.RegisteredClaims
}

func SubjectFromContext(ctx context.Context) string {
	var (
		value         = ctx.Value(JwtClaimDataKey)
		realStruct, _ = value.(*JwtClaimData)
	)
	return realStruct.Subject
}

func RolesFromContext(ctx context.Context) []string {
	var (
		value         = ctx.Value(JwtClaimDataKey)
		realStruct, _ = value.(*JwtClaimData)
	)
	return realStruct.Roles
}

func PermissionsFromContext(ctx context.Context) []string {
	var (
		value         = ctx.Value(JwtClaimDataKey)
		realStruct, _ = value.(*JwtClaimData)
	)
	return realStruct.Permissions
}
