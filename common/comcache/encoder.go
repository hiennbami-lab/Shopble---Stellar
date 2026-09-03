package comcache

import (
	"encoding/json"
)

func dumpEncoder(encoder IEncoder, fromObj, toObj any) error {
	fromBytes, err := encoder.Encode(fromObj)
	if err != nil {
		return err
	}
	return encoder.Decode(fromBytes, toObj)
}

type JsonEncoder struct{}

func (e JsonEncoder) Encode(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (e JsonEncoder) Decode(buff []byte, v any) error {
	return json.Unmarshal(buff, v)
}
