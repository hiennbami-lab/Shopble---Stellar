package gsec

import (
	"crypto/aes"
	"crypto/cipher"
	"shopble/common/comutils"
)

var (
	NonceSize = 12
)

type AesGmc struct {
	secretKey string
	encoder   Base64Encoder
}

func NewAesGcm() *AesGmc {
	return &AesGmc{
		encoder:   Base64Encoder{},
		secretKey: GetConfig().SecretKey,
	}
}

func NewAesGcmWithKey(secretKey string) *AesGmc {
	return &AesGmc{
		encoder:   Base64Encoder{},
		secretKey: secretKey,
	}
}

func (a *AesGmc) GetSalt() string {
	return ""
}

func (a *AesGmc) Encrypt(buff []byte) (_ string, err error) {
	c, err := aes.NewCipher([]byte(a.secretKey))
	if err != nil {
		return
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return
	}
	dumpNonce := comutils.RandomBytesF(NonceSize)
	encrypted := gcm.Seal(dumpNonce, dumpNonce, buff, nil)
	return a.encoder.EncodeRaw(encrypted), nil
}

func (a *AesGmc) Decrypt(encodedData string, encodedSalt string) (_ []byte, err error) {
	buff, err := a.encoder.DecodeRaw(encodedData)
	if err != nil {
		return
	}
	c, err := aes.NewCipher([]byte(a.secretKey))
	if err != nil {
		return
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return
	}
	nonce, cipherText := buff[:gcm.NonceSize()], buff[gcm.NonceSize():]
	decryptedMsg, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return
	}
	return decryptedMsg, nil
}
