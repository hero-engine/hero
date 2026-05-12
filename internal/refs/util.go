package refs

import "encoding/json"

func unmarshalArgs(s string, out *map[string]any) error {
	if s == "" {
		return nil
	}
	return json.Unmarshal([]byte(s), out)
}
