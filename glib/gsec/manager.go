package gsec

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"shopble/common/comerr"
	"io"
)

func Encrypt(plaintext string) (_ string, err error) {
	var (
		conf = GetConfig()
	)
	// AES requires key length 16, 24, or 32 bytes
	block, err := aes.NewCipher([]byte(conf.SecretKey))
	if err != nil {
		err = comerr.WrapMessage(err, "ciper: new failed")
		return
	}

	// GCM for authentication
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		err = comerr.WrapMessage(err, "gcm: authenticate failed")
		return
	}

	// Generate random nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		err = comerr.WrapMessage(err, "nounce: generate random failed")
		return
	}

	// Encrypt
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(cipherText string) (_ string, err error) {
	var (
		conf = GetConfig()
	)
	data, err := base64.RawURLEncoding.DecodeString(cipherText)
	if err != nil {
		err = comerr.WrapMessage(err, "ciper text: decode failed")
		return
	}

	block, err := aes.NewCipher([]byte(conf.SecretKey))
	if err != nil {
		err = comerr.WrapMessage(err, "ciper: new block failed")
		return
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		err = comerr.WrapMessage(err, "gcm: authenticate failed")
		return
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		err = comerr.WrapMessage(err, "nonce: check failed")
		return
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		err = comerr.WrapMessage(err, "ciper: decode failed")
		return
	}
	return string(plaintext), nil
}
