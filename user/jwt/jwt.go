package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
)

const SECRET = "go-pipelines"

type JWT string

// CreateJWT uses the payload you passed on to create a HS256-signed JWT
func CreateJWT(payload interface{}) JWT {

	var jwt strings.Builder

	pb, _ := json.Marshal(payload)
	pb64 := base64.StdEncoding.EncodeToString(pb)
	h := hmac.New(sha256.New, []byte(SECRET))

	h.Write([]byte(pb64))

	signature := hex.EncodeToString(h.Sum(nil))

	jwt.WriteString(pb64)
	jwt.WriteRune('.')
	jwt.WriteString(signature)
	return JWT(jwt.String())
}

// VerifyJWT verifies the JWT and returns payload unmarshalled to an interface{}.
// The interface{} needs to then be marshalled and unmarshalled again.
func VerifyJWT(jwt JWT) (bool, interface{}) {

	a := strings.Split(string(jwt), ".")
	if len(a) != 2 {
		return false, nil
	}

	pb64, signature := a[0], a[1]

	h := hmac.New(sha256.New, []byte(SECRET))
	h.Write([]byte(pb64))
	csignature := hex.EncodeToString(h.Sum(nil))
	if csignature != signature {
		return false, nil
	}

	pb, err := base64.StdEncoding.DecodeString(pb64)
	if err != nil {
		return false, nil
	}
	var p interface{}
	err = json.Unmarshal(pb, &p)
	if err != nil {
		return false, nil
	}

	return true, p
}
