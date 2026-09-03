package comrestful

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"shopble/common/comerr"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

func do(req *http.Request) (status int, body []byte, err error) {
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	body, err = io.ReadAll(res.Body)
	if err != nil {
		return
	}
	status = res.StatusCode
	return
}

func Request(ctx context.Context, option RequestOption, out any) (status int, err error) {
	methodStr := methodSelector(option.Method)

	var (
		data io.Reader = nil
		en   Encoder
	)
	if option.Body != nil {
		en = encoder[option.Body.Type]
		data, err = en.Encode(option.Body.Data)
		if err != nil {
			return
		}
	}

	req, err := newRequest(ctx, methodStr, option.Url, data)
	if err != nil {
		err = fmt.Errorf("create request failed: %w", err)
		return
	}

	if option.Header != nil {
		option.Header.Attached(req)
	}

	realEncoder, ok := en.(*FormMultipartEncoder)
	if ok {
		req.Header.Set("Content-Type", realEncoder.contentType)
	}

	status, body, err := do(req)
	if err != nil {
		err = fmt.Errorf("call request failed: %w", err)
		return
	}

	if status >= 200 && status <= 299 {
		if out == nil {
			return
		}
		if err = json.Unmarshal(body, out); err != nil {
			err = fmt.Errorf("parse dst failed: %w", err)
		}
		return
	}
	length := 1024
	if len(body) < length {
		length = len(body) - 1
	}
	return status, fmt.Errorf("call request failed: response=%s", string(body[:length]))
}

func RequestRaw(ctx context.Context, option RequestOption) (_ *http.Response, err error) {
	methodStr := methodSelector(option.Method)

	var (
		data io.Reader = nil
		en   Encoder
	)
	if option.Body != nil {
		en = encoder[option.Body.Type]
		data, err = en.Encode(option.Body.Data)
		if err != nil {
			return
		}
	}

	req, err := newRequest(ctx, methodStr, option.Url, data)
	if err != nil {
		err = fmt.Errorf("create request failed: %w", err)
		return
	}

	if option.Header != nil {
		option.Header.Attached(req)
	}

	realEncoder, ok := en.(*FormMultipartEncoder)
	if ok {
		req.Header.Set("Content-Type", realEncoder.contentType)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = fmt.Errorf("execute request failed: %w", err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		err = fmt.Errorf("failed request: %d", resp.StatusCode)
		return
	}
	return resp, nil
}

type (
	RequestV2Option struct {
		// applicaton/json
		jsonBody io.Reader

		// multiplart/form-data
		body   *bytes.Buffer
		writer *multipart.Writer

		contentType string
	}
	RequestV2Result struct {
		*http.Response
		IsError bool
	}
)

func WithFormFile(originalFile *bytes.Buffer, fileName string) func(*RequestV2Option) error {
	return func(rvo *RequestV2Option) (err error) {
		if rvo.contentType != "" && !strings.Contains(rvo.contentType, "multipart/form-data") {
			return comerr.WrapMessage(comerr.ErrorDataInvalid, "content-type is invalid")
		}
		if rvo.body == nil {
			rvo.body = &bytes.Buffer{}
			rvo.writer = multipart.NewWriter(rvo.body)
		}
		part, err := rvo.writer.CreateFormFile("files", fileName)
		if err != nil {
			err = comerr.WrapMessage(err, "failed to create form file")
			return
		}
		_, err = io.Copy(part, originalFile)
		if err != nil {
			err = comerr.WrapMessage(err, "failed to copy file data")
			return
		}
		rvo.contentType = rvo.writer.FormDataContentType()
		return
	}
}

func WithFormField(field string, value string) func(*RequestV2Option) error {
	return func(rvo *RequestV2Option) (err error) {
		if rvo.contentType != "" && !strings.Contains(rvo.contentType, "multipart/form-data") {
			return comerr.WrapMessage(comerr.ErrorDataInvalid, "content-type is invalid")
		}
		if rvo.body == nil {
			rvo.body = &bytes.Buffer{}
			rvo.writer = multipart.NewWriter(rvo.body)
		}
		err = rvo.writer.WriteField(field, value)
		if err != nil {
			err = comerr.WrapMessage(err, "failed to write field")
			return
		}
		rvo.contentType = rvo.writer.FormDataContentType()
		return
	}
}

func WithJSON(data any) func(*RequestV2Option) error {
	return func(rvo *RequestV2Option) (err error) {
		if rvo.contentType != "" && rvo.contentType != "application/json" {
			return comerr.WrapMessage(comerr.ErrorDataInvalid, "content-type is invalid")
		}
		buff, err := json.Marshal(data)
		if err != nil {
			return comerr.WrapMessage(err, "encode json failed")
		}
		rvo.jsonBody = bytes.NewBuffer(buff)
		rvo.contentType = "application/json"
		return nil
	}
}

func (r *RequestV2Result) Decode(out any) (err error) {
	if r.Response == nil {
		return
	}
	defer r.Response.Body.Close()
	buff, err := io.ReadAll(r.Response.Body)
	if err != nil {
		err = comerr.WrapMessage(err, "read body failed")
		return
	}
	if r.IsError {
		err = comerr.WrapMessage(comerr.ErrorServerUnknown, "request failed: "+string(buff))
		return
	}
	err = json.Unmarshal(buff, out)
	if err != nil {
		err = comerr.WrapMessage(err, "decode response failed")
	}
	return
}

func (r *RequestV2Result) Close() {
	if r.Response == nil {
		return
	}
	r.Response.Body.Close()
}

func RequestV2(ctx context.Context, option RequestOption, optSetters ...func(*RequestV2Option) error) (result *RequestV2Result, err error) {
	var (
		method = methodSelector(option.Method)
		opt    RequestV2Option
	)
	for _, setter := range optSetters {
		err = setter(&opt)
		if err != nil {
			return
		}
	}
	if opt.writer != nil {
		err = opt.writer.Close()
		if err != nil {
			err = comerr.WrapMessage(err, "close writer failed")
			return
		}
	}

	var body io.Reader = nil
	if opt.body != nil {
		body = opt.body
	}
	if opt.jsonBody != nil {
		body = opt.jsonBody
	}
	request, err := http.NewRequestWithContext(ctx, method, option.Url, body)
	if err != nil {
		err = fmt.Errorf("create request failed: %w", err)
		return
	}
	if option.Header != nil {
		option.Header.Attached(request)
	}
	if opt.contentType != "" {
		request.Header.Set("Content-Type", opt.contentType)
	}
	client := &http.Client{
		Timeout: 120 * time.Second,
	}
	response, err := client.Do(request)
	if err != nil {
		err = comerr.WrapMessage(err, "failed to call AI mockup service")
		return
	}
	result = &RequestV2Result{
		Response: response,
	}
	if response.StatusCode != http.StatusOK {
		result.IsError = true
	}
	return result, nil
}
