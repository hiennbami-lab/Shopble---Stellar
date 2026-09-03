package gmeta

import (
	"golang.org/x/crypto/bcrypt"
)

type Password string

func (p Password) EncodeHashed() (string, error) {
	hashPassword, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashPassword), nil
}

func (p Password) SameAs(hashedPass string) (err error) {
	return bcrypt.CompareHashAndPassword([]byte(hashedPass), []byte(p))
}
