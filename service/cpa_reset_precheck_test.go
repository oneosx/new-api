package service

import (
	"encoding/json"
	"testing"
)

// The management API returns `body` either as an object or as a JSON *string*
// carrying the upstream payload. Both must decode; only the object shape was
// handled before, so string-bodied nodes failed every reset with
// "cannot unmarshal string into Go struct field .body".
func TestDecodeResetCreditPrecheckBody(t *testing.T) {
	inner := `{"available_count":3}`

	objectEnvelope := `{"status_code":200,"body":` + inner + `}`
	stringEnvelope := `{"status_code":200,"body":` + strconvQuote(inner) + `}`

	for name, raw := range map[string]string{
		"object body": objectEnvelope,
		"string body": stringEnvelope,
	} {
		var env struct {
			StatusCode int             `json:"status_code"`
			Body       json.RawMessage `json:"body"`
		}
		if err := json.Unmarshal([]byte(raw), &env); err != nil {
			t.Fatalf("%s: envelope: %v", name, err)
		}
		body := env.Body
		var asString string
		if err := json.Unmarshal(body, &asString); err == nil {
			body = json.RawMessage(asString)
		}
		var got struct {
			AvailableCount int `json:"available_count"`
		}
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("%s: body: %v", name, err)
		}
		if got.AvailableCount != 3 {
			t.Fatalf("%s: AvailableCount = %d, want 3", name, got.AvailableCount)
		}
	}
}

func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
