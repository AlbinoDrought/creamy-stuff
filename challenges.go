package main

import (
	"encoding/hex"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Challenge struct {
	ID         string
	Public     bool
	SharedPath string

	HasPassword  bool
	PasswordHash string
}

func (challenge *Challenge) CookieName() string {
	return hex.EncodeToString([]byte(challenge.ID))
}

func (challenge *Challenge) SetPassword(password string) error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	challenge.HasPassword = true
	challenge.PasswordHash = string(passwordHash)
	return nil
}

func (challenge *Challenge) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(challenge.PasswordHash), []byte(password))
}

func (challenge *Challenge) StorePassword(password string, w http.ResponseWriter, r *http.Request) {
	passwordCookie := &http.Cookie{
		Name:    challenge.CookieName(),
		Path:    "/",
		Value:   hex.EncodeToString([]byte(password)),
		MaxAge:  int(time.Hour.Seconds()),
		Expires: time.Now().Add(time.Hour),
	}
	http.SetCookie(w, passwordCookie)
}

func (challenge *Challenge) Accessible(r *http.Request) bool {
	if challenge.Public {
		return true
	}

	if challenge.HasPassword {
		if cookie, _ := r.Cookie(challenge.CookieName()); cookie != nil {
			if decoded, err := hex.DecodeString(cookie.Value); err == nil {
				return challenge.CheckPassword(string(decoded)) == nil
			}
		}
	}

	return false
}

type ChallengeRepository interface {
	Get(ID string) *Challenge
	Set(challenge *Challenge)
	Remove(challenge *Challenge)
}

type ArrayChallengeRepository struct {
	challenges map[string]*Challenge
}

func (repo *ArrayChallengeRepository) Get(ID string) *Challenge {
	challenge, _ := repo.challenges[ID]
	return challenge
}

func (repo *ArrayChallengeRepository) Set(challenge *Challenge) {
	repo.challenges[challenge.ID] = challenge
}

func (repo *ArrayChallengeRepository) Remove(challenge *Challenge) {
	delete(repo.challenges, challenge.ID)
}

var challengeRepository ChallengeRepository

func init() {
	challengeRepository = &ArrayChallengeRepository{
		challenges: map[string]*Challenge{
			"foo": &Challenge{
				ID:         "foo",
				Public:     true,
				SharedPath: "data",
			},
			"bar": &Challenge{
				ID:         "bar",
				Public:     false,
				SharedPath: "data-private",
			},
		},
	}
}
