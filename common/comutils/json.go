package comutils

import "encoding/json"

func DecodeJson(buf []byte, val any) error {
	err := json.Unmarshal(buf, val)
	if err != nil {
		return err
	}
	return nil
}

func JsonEncodeBytes(v any) ([]byte, error) {
	return json.Marshal(v)
}

func JsonDecodeBytes(buff []byte, v any) error {
	return json.Unmarshal(buff, v)
}
