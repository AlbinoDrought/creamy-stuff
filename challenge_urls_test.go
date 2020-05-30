package main

import (
	"testing"

	"github.com/AlbinoDrought/creamy-stuff/stuff"
)

func TestViewChallenge(t *testing.T) {
	challenge := &stuff.Challenge{
		ID: "ZohAiu_wN9HmekN_qBo8ujZi0THKFr3BeAzcbJ-tBYg1I5XHZFj0NjmFlJeIH1xjMfXv_N3CYRTvc57wSvkBMQ==",
	}

	generator := &hardcodedURLGenerator{}
	expected := "/view/ZohAiu_wN9HmekN_qBo8ujZi0THKFr3BeAzcbJ-tBYg1I5XHZFj0NjmFlJeIH1xjMfXv_N3CYRTvc57wSvkBMQ%3D%3D"
	actual := generator.ViewChallenge(challenge)

	if actual != expected {
		t.Errorf("expected %s but got %s", expected, actual)
	}
}
