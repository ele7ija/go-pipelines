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

type Payload struct {
	Username string `json:"username"`
	Iat      int64  `json:"iat"`
	Exp      int64  `json:"exp"`
}

func CreateJWT(payload Payload) JWT {

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

func VerifyJWT(jwt JWT) (bool, Payload) {

	a := strings.Split(string(jwt), ".")
	if len(a) != 2 {
		return false, Payload{}
	}

	pb64, signature := a[0], a[1]

	h := hmac.New(sha256.New, []byte(SECRET))
	h.Write([]byte(pb64))
	csignature := hex.EncodeToString(h.Sum(nil))
	if csignature != signature {
		return false, Payload{}
	}

	pb, err := base64.StdEncoding.DecodeString(pb64)
	if err != nil {
		return false, Payload{}
	}
	var payload Payload
	err = json.Unmarshal(pb, &payload)
	if err != nil {
		return false, Payload{}
	}

	return true, payload
}
