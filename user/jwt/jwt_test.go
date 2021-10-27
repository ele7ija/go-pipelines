package jwt

import (
	"encoding/json"
	"testing"
)

type JWTPayload struct {
	Username string `json:"username"`
	Iat      int64  `json:"iat"`
	Exp      int64  `json:"exp"`
}

func TestCreateJWT(t *testing.T) {
	testUsername := "bojan"

	t.Run("default", func(t *testing.T) {
		p := JWTPayload{
			Username: testUsername,
			Iat:      0,
			Exp:      0,
		}
		jwt := CreateJWT(p)
		b, p2I := VerifyJWT(jwt)
		pb, _ := json.Marshal(p2I)
		var p2 JWTPayload
		if err := json.Unmarshal(pb, &p2); err != nil {
			t.Fatalf("bad payload structure")
		}

		if !b {
			t.Fatalf("jwt couldn't be verified")
		}
		if p != p2 {
			t.Fatalf("payloads changed")
		}
	})
}

func TestVerifyJWT(t *testing.T) {

}
