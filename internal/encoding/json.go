package encoding

import "encoding/json"

func EncodeJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

func EncodeJSONString(v any) (string, error) {
	buf, err := EncodeJSON(v)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}
