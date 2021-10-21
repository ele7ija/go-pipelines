package jwt

import (
	"testing"
)

func TestCreateJWT(t *testing.T) {
	testUsername := "bojan"

	t.Run("default", func(t *testing.T) {
		p := Payload{
			Username: testUsername,
			Iat:      0,
			Exp:      0,
		}
		jwt := CreateJWT(p)
		b, p2 := VerifyJWT(jwt)
		if !b {
			t.Fatalf("jwt couldn't be verified")
		}
		if p != p2 {
			t.Fatalf("payloads changeds")
		}
	})
}

func TestVerifyJWT(t *testing.T) {

}
