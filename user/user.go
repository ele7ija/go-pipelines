package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/ele7ija/go-pipelines/user/jwt"
	"github.com/open-policy-agent/opa/rego"
	log "github.com/sirupsen/logrus"
	"time"
)

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type JWTPayload struct {
	Username string `json:"username"`
	Iat      int64  `json:"iat"`
	Exp      int64  `json:"exp"`
}

type Service interface {
	Login(ctx context.Context, user User) (jwt.JWT, error)
	GetUser(ctx context.Context, receivedJwt jwt.JWT) (User, error)
	IsAdmin(ctx context.Context, username string) (bool, error)
}

func NewService(db *sql.DB) Service {
	return service{db: db}
}

type service struct {
	db *sql.DB
}

func (s service) Login(ctx context.Context, user User) (jwt.JWT, error) {
	// Check if user exists (auth)
	row := s.db.QueryRowContext(ctx, "SELECT id FROM \"user\" WHERE username = $1 AND password = $2", user.Username, user.Password)
	var id int
	err := row.Scan(&id)
	if err != nil {
		return "", err
	}

	iat := time.Now()
	exp := time.Now().Add(time.Hour * 2)
	payload := JWTPayload{
		Username: user.Username,
		Iat:      iat.Unix(),
		Exp:      exp.Unix(),
	}
	createdJwt := jwt.CreateJWT(payload)

	return createdJwt, nil
}

func (s service) GetUser(ctx context.Context, receivedJwt jwt.JWT) (User, error) {

	b, pI := jwt.VerifyJWT(receivedJwt)
	if !b {
		return User{}, fmt.Errorf("couldn't verify JWT")
	}

	pb, _ := json.Marshal(pI)
	var p JWTPayload
	if err := json.Unmarshal(pb, &p); err != nil {
		return User{}, fmt.Errorf("bad payload")
	}
	expiration := time.Unix(p.Exp, 0)
	if expiration.Before(time.Now()) {
		return User{}, fmt.Errorf("jwt expired")
	}

	row := s.db.QueryRowContext(ctx, "SELECT id FROM \"user\" WHERE username = $1", p.Username)
	var id int
	err := row.Scan(&id)
	if err != nil {
		return User{}, err
	}

	return User{
		ID:       id,
		Username: p.Username,
	}, nil

}

func (s service) IsAdmin(ctx context.Context, username string) (bool, error) {

	path := "/home/bp/go/src/github.com/ele7ija/go-pipelines/user/rego"

	query, err := rego.New(
		rego.Query("data.rbac.authz.allow"),
		rego.Load([]string{path}, nil),
	).PrepareForEval(ctx)
	if err != nil {
		log.Errorf("bad Rego: %s", err)
		return false, err
	}

	input := map[string]interface{}{
		"username": username,
	}

	results, err := query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		log.Errorf("error evaluating policy: %s", err)
		return false, err
	}
	if !results.Allowed() {
		return false, nil
	}
	return true, nil
}
