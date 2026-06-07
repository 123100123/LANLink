package protocol

import "encoding/json"

func EncodePayload(v any) (json.RawMessage, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(data), nil
}

func DecodePayload(payload json.RawMessage, v any) error {
	return json.Unmarshal(payload, v)
}
