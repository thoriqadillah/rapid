package helpers

import (
	"encoding/json"
	"fmt"
)

func Jsonify(data interface{}) string {
	v, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return string(v)
}
