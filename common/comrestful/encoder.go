package comrestful

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/url"
	"strings"
)

type JsonEncoder struct{}

func (e *JsonEncoder) Encode(data any) (io.Reader, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(body), nil
}

type FormUrlEncoder struct{}

func (e *FormUrlEncoder) Encode(data any) (io.Reader, error) {
	mapData := data.(map[string]string)
	body := url.Values{}
	for key, element := range mapData {
		body.Set(key, element)
	}
	return strings.NewReader(body.Encode()), nil
}

type FormMultipartEncoder struct {
	fileKeys    []string
	contentType string
}

func (e *FormMultipartEncoder) getFileKey(key string) string {
	var defaultKey = "image"
	for _, fileKey := range e.fileKeys {
		if strings.Contains(key, fileKey) {
			return fileKey
		}
	}
	return defaultKey
}

func (e *FormMultipartEncoder) Encode(data any) (_ io.Reader, err error) {
	mapData := data.(map[string]string)
	var (
		buffer bytes.Buffer
		writer = multipart.NewWriter(&buffer)
	)
	defer writer.Close()
	for key, value := range mapData {
		var (
			fieldWriter io.Writer
			imgBuff     []byte
			valuePart   = strings.Split(value, "@")
		)
		if len(valuePart) > 1 {
			partType, partSrc := valuePart[0], valuePart[1]
			if !strings.Contains(partType, "file") {
				continue
			}
			fieldWriter, err = writer.CreateFormFile(e.getFileKey(key), partType)
			if err != nil {
				return
			}
			imgBuff, err = base64.StdEncoding.DecodeString(partSrc)
			if err != nil {
				return
			}
			_, err = io.Copy(fieldWriter, bytes.NewBuffer(imgBuff))
			if err != nil {
				return
			}
			continue
		}
		fieldWriter, err = writer.CreateFormField(key)
		if err != nil {
			return
		}
		_, err = fieldWriter.Write([]byte(value))
		if err != nil {
			return
		}
	}
	e.contentType = writer.FormDataContentType()
	return &buffer, err
}

var encoder = map[EncoderType]Encoder{
	JSON:    &JsonEncoder{},
	FormURL: &FormUrlEncoder{},
	FormMultipart: &FormMultipartEncoder{
		fileKeys: []string{"image", "file"},
	},
}
