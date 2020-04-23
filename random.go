package main

import (
	"crypto/rand"
	"encoding/base64"
)

func RandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

func RandomString(n int) (string, error) {
	bytes, err := RandomBytes(n)
	return base64.URLEncoding.EncodeToString(bytes), err
}
