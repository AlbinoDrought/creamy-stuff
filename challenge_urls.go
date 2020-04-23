package main

import (
	"net/url"
	"path"

	"github.com/AlbinoDrought/creamy-stuff/stuff"
)

type ChallengeURLGenerator interface {
	ViewChallenge(challenge *stuff.Challenge) string
	ViewChallengePath(challenge *stuff.Challenge, filePath string) string
}

type BrowseURLGenerator interface {
	BrowsePath(filePath string) string
	SharePath(filePath string) string
}

type hardcodedURLGenerator struct{}

func (generator *hardcodedURLGenerator) ViewChallenge(challenge *stuff.Challenge) string {
	challengeURL := url.URL{Path: "/view/" + challenge.ID}
	return challengeURL.String()
}

func (generator *hardcodedURLGenerator) ViewChallengePath(challenge *stuff.Challenge, filePath string) string {
	browseURL := url.URL{Path: "/view/" + challenge.ID + "/" + path.Clean(filePath)}
	return browseURL.String()
}

func (generator *hardcodedURLGenerator) BrowsePath(filePath string) string {
	browseURL := url.URL{Path: "/stuff/browse" + path.Clean(filePath)}
	return browseURL.String()
}

func (generator *hardcodedURLGenerator) SharePath(filePath string) string {
	browseURL := url.URL{Path: "/stuff/share" + path.Clean(filePath)}
	return browseURL.String()
}
