package gauth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func NewClaimWithWrapperInfo(roles []string, permisisons []string) func(subject string) *JwtClaimData {
	return func(subject string) *JwtClaimData {
		var (
			now  = time.Now()
			conf = GetConfig()
		)
		return &JwtClaimData{
			Roles:       roles,
			Permissions: permisisons,
			RegisteredClaims: &jwt.RegisteredClaims{
				Subject:   subject,
				ExpiresAt: jwt.NewNumericDate(now.Add(conf.TokenTimeout * time.Second)),
				IssuedAt:  jwt.NewNumericDate(now),
				NotBefore: jwt.NewNumericDate(now),
				Issuer:    conf.Issuer,
			},
		}
	}
}

func IsValid(token string) (_ *JwtClaimData, err error) {
	var (
		claim = &JwtClaimData{}
		conf  = GetConfig()
	)
	_, err = jwt.ParseWithClaims(token, claim, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("invalid signing method")
		}
		return []byte(conf.SecretKey), nil
	})
	if err != nil {
		return
	}
	if !claim.VerifyIssuer(conf.Issuer, true) {
		err = fmt.Errorf("mismatch issuer")
		return
	}
	return claim, nil
}

func Token(datagenerator func() *JwtClaimData) (token string, err error) {
	var (
		signedClaim = jwt.NewWithClaims(jwt.SigningMethodHS256, datagenerator())
		conf        = GetConfig()
	)
	return signedClaim.SignedString([]byte(conf.SecretKey))
}
