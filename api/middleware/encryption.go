package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"shopble/api"
	"shopble/common/comerr"
	"shopble/glib/gsec"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type RequestTimestamp struct {
	Timestamp int64 `json:"timestamp"`
}

func validateTimestamp(timestamp int64) error {
	const maxAgeMs = 30000 // 30 seconds

	if timestamp == 0 {
		return errors.New("missing timestamp")
	}

	now := time.Now().UnixMilli()
	diff := now - timestamp

	if diff < 0 {
		diff = -diff // Handle future timestamps (clock skew)
	}

	if diff > maxAgeMs {
		return errors.New("request expired or invalid timestamp")
	}

	return nil
}

var VerifyRequest gin.HandlerFunc = func(ctx *gin.Context) {
	switch ctx.Request.Method {
	case http.MethodGet:
		ctx.Next()
		return
	}
	switch ctx.ContentType() {
	case binding.MIMEPlain:
		break
	case binding.MIMEMultipartPOSTForm:
		ctx.Next()
		return
	default:
		api.AbortWithErr(ctx, comerr.WrapStack(comerr.ErrorDataInvalid, "decode body failed"))
		return
	}
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		api.AbortWithErr(ctx, comerr.WrapStack(err, "read request body failed"))
		return
	}
	if len(body) > 0 {
		encrypter := gsec.NewAesGcm()
		decryptedData, err := encrypter.Decrypt(string(body), "")
		if err != nil {
			api.AbortWithErr(ctx, comerr.WrapStack(err, "decrypt request failed"))
			return
		}
		var (
			timestampres RequestTimestamp
		)
		err = json.Unmarshal(decryptedData, &timestampres)
		if err != nil {
			api.AbortWithErr(ctx, comerr.WrapStack(err, "timestamp invalid"))
			return
		}
		err = validateTimestamp(timestampres.Timestamp)
		if err != nil {
			api.AbortWithErr(ctx, comerr.WrapStack(err, "timestamp invalid"))
			return
		}

		//TODO: logging request
		ctx.Request.Body = io.NopCloser(bytes.NewBuffer(decryptedData))
		ctx.Request.Header.Set("Content-Type", "application/json")
	}

	newBodySize := len(body)
	ctx.Request.Header.Set("Content-Length", strconv.Itoa(newBodySize))
	ctx.Request.ContentLength = int64(newBodySize)
	ctx.Next()
}
