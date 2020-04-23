package main

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const csrfTokenLength = 64
const csrfCookieName = "CSRF-TOKEN"

func getCSRF(r *http.Request) (string, error) {
	if csrfCookie, _ := r.Cookie(csrfCookieName); csrfCookie != nil {
		return csrfCookie.Value, nil
	}

	return "", fmt.Errorf("No %s cookie found", csrfCookieName)
}

func validCSRF(r *http.Request, passedToken string) error {
	realToken, err := getCSRF(r)
	if err != nil {
		return err
	}

	if subtle.ConstantTimeCompare([]byte(realToken), []byte(passedToken)) != 1 {
		return errors.New("CSRF token mismatch")
	}

	return nil
}

func getOrCreateCSRF(w http.ResponseWriter, r *http.Request) (string, error) {
	var err error
	csrfCookie, _ := r.Cookie(csrfCookieName)
	if csrfCookie == nil {
		csrfCookie = &http.Cookie{Name: csrfCookieName, Path: "/"}
		if csrfCookie.Value, err = RandomString(csrfTokenLength); err != nil {
			return "", err
		}
	}

	csrfCookie.Expires = time.Now().Add(1 * time.Hour)
	csrfCookie.MaxAge = int(time.Hour.Seconds())
	http.SetCookie(w, csrfCookie)

	return csrfCookie.Value, nil
}
