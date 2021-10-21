package user

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/ele7ija/go-pipelines/user/jwt"
	"time"
)

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Service interface {
	Login(ctx context.Context, user User) (jwt.JWT, error)
}

type ServiceDefault struct {
	DB *sql.DB
}

func (s ServiceDefault) Login(ctx context.Context, user User) (jwt.JWT, error) {

	// Check if user exists (auth)
	row := s.DB.QueryRowContext(ctx, "SELECT id FROM \"user\" WHERE username = $1 AND password = $2", user.Username, user.Password)
	var id int
	err := row.Scan(&id)
	if err != nil {
		return "", err
	}

	iat := time.Now()
	exp := time.Now().Add(time.Hour * 2)
	payload := jwt.Payload{
		Username: user.Username,
		Iat:      iat.Unix(),
		Exp:      exp.Unix(),
	}
	createdJwt := jwt.CreateJWT(payload)

	return createdJwt, nil
}

func (s ServiceDefault) GetUser(ctx context.Context, receivedJwt jwt.JWT) (User, error) {

	b, p := jwt.VerifyJWT(receivedJwt)
	if !b {
		return User{}, fmt.Errorf("couldn't verify JWT")
	}
	expiration := time.Unix(p.Exp, 0)
	if expiration.Before(time.Now()) {
		return User{}, fmt.Errorf("jwt expired")
	}

	row := s.DB.QueryRowContext(ctx, "SELECT id FROM \"user\" WHERE username = $1", p.Username)
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
