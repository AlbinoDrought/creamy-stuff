package main

//go:generate go get -u github.com/valyala/quicktemplate/qtc
//go:generate qtc -dir=templates

import (
	"log"
	"net/http"
	"net/url"
	"path"
	"sort"

	"github.com/AlbinoDrought/creamy-stuff/stuff"
	"github.com/AlbinoDrought/creamy-stuff/templates"
	"github.com/julienschmidt/httprouter"
)

const challengeIDLength = 64
const challengeRandomPasswordLength = 128

var dataDirectory = "data"

var challengeRepository stuff.ChallengeRepository

func init() {
	challengeRepository = stuff.NewArrayChallengeRepository()

	challengeRepository.Set(&stuff.Challenge{
		ID:         "foo",
		Public:     true,
		SharedPath: "data",
	})

	challenge := &stuff.Challenge{
		ID:         "bar",
		Public:     false,
		SharedPath: "data-private",
	}
	challenge.SetPassword("foo")
	challengeRepository.Set(challenge)
}

func renderServerError(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("500 Internal Server Error"))
}

func renderUnauthorized(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("401 Unauthorized"))
}

func renderChallengeNotFound(w http.ResponseWriter, r *http.Request, ID string) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Challenge not found"))
}

func handleChallengesIndex(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// todo: allow controlling pagination
	challenges := challengeRepository.All(10, 0)

	challengeResources := make([]*templates.ChallengeResource, len(challenges))
	for i, challenge := range challenges {
		challengeURL := url.URL{Path: "/view/" + challenge.ID}

		challengeResources[i] = &templates.ChallengeResource{
			Challenge: challenge,

			ViewLink: challengeURL.String(),
		}
	}

	templates.WritePageTemplate(w, &templates.ChallengeIndexPage{
		Challenges: challengeResources,

		Page: 1,
	}, &templates.PrivateNav{})
}

func handleStuffIndex(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	filePath := path.Clean(ps.ByName("filepath"))

	dir := http.Dir(dataDirectory)
	file, err := dir.Open(filePath)
	if err != nil {
		log.Printf("Error opening file %v: %v", filePath, err)
		renderServerError(w, r, err)
		return
	}

	stat, err := file.Stat()
	if err != nil {
		log.Printf("Error stat'ing file %v: %v", filePath, err)
		renderServerError(w, r, err)
		return
	}

	if !stat.IsDir() {
		http.ServeFile(w, r, path.Join(dataDirectory, filePath))
		return
	}

	dirs, err := file.Readdir(-1)
	if err != nil {
		log.Printf("Error reading directory %v: %v", filePath, err)
		renderServerError(w, r, err)
		return
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })

	files := make([]templates.File, len(dirs))
	for i, dir := range dirs {
		name := dir.Name()
		if dir.IsDir() {
			name += "/"
		}

		pathRelativeToDataDir := path.Join(filePath, name)

		browseURL := url.URL{Path: "/stuff/browse" + pathRelativeToDataDir}
		shareURL := url.URL{Path: "/stuff/share" + pathRelativeToDataDir}

		files[i].Label = name
		files[i].BrowseLink = browseURL.String()
		files[i].ShareLink = shareURL.String()
	}

	upwardsURL := url.URL{Path: "/stuff/browse" + path.Join(filePath, "..")}

	atRoot := filePath == "" || filePath == "/" || filePath == "."
	directoryName := filePath
	if atRoot {
		directoryName = "/"
	}

	browsePage := &templates.BrowsePage{
		DirectoryName: directoryName,
		Files:         files,

		CanTravelUpwards: !atRoot,
		UpwardsLink:      upwardsURL.String(),
	}
	templates.WritePageTemplate(w, browsePage, &templates.PrivateNav{})
}

func handleStuffShowForm(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	filePath := path.Clean(ps.ByName("filepath"))

	dir := http.Dir(dataDirectory)
	_, err := dir.Open(filePath)
	if err != nil {
		log.Printf("Error opening file %v: %v", filePath, err)
		renderServerError(w, r, err)
		return
	}

	csrfToken, err := getOrCreateCSRF(w, r)
	if err != nil {
		log.Printf("Error with getOrCreateCSRF: %v", err)
		renderServerError(w, r, err)
		return
	}

	randomPassword, err := RandomString(challengeRandomPasswordLength)
	if err != nil {
		log.Printf("Error generating random challenge password: %v", err)
		renderServerError(w, r, err)
		return
	}

	cancelURL := url.URL{Path: "/stuff/browse" + path.Join(filePath, "..")}

	sharePage := &templates.SharePage{
		Path:           filePath,
		CSRF:           csrfToken,
		RandomPassword: randomPassword,

		CancelLink: cancelURL.String(),
	}
	templates.WritePageTemplate(w, sharePage, &templates.PrivateNav{})
}

func handleStuffReceiveForm(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	filePath := path.Clean(ps.ByName("filepath"))

	dir := http.Dir(dataDirectory)
	_, err := dir.Open(filePath)
	if err != nil {
		log.Printf("Error opening file %v: %v", filePath, err)
		renderServerError(w, r, err)
		return
	}

	if err := validCSRF(r, r.FormValue("_token")); err != nil {
		log.Printf("Error validating CSRF token: %v", err)
		renderServerError(w, r, err)
		return
	}

	challengeID, err := RandomString(challengeIDLength)
	if err != nil {
		log.Printf("Error generating challenge ID: %v", err)
		renderServerError(w, r, err)
		return
	}

	challenge := &stuff.Challenge{
		ID:         challengeID,
		Public:     r.FormValue("public") == "1",
		SharedPath: filePath,
	}
	if challengePassword := r.FormValue("challenge-password"); challengePassword != "" {
		if err = challenge.SetPassword(challengePassword); err != nil {
			log.Printf("Error setting challenge password: %v", err)
			renderServerError(w, r, err)
			return
		}
	}

	challengeRepository.Set(challenge)

	challengeURL := url.URL{Path: "/view/" + challenge.ID}

	sharedChallengePage := &templates.SharedChallengePage{
		Challenge: challenge,

		ViewLink: challengeURL.String(),
	}
	templates.WritePageTemplate(w, sharedChallengePage, &templates.PrivateNav{})
}

func handleChallengeFilepath(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	challengeID := ps.ByName("challenge")
	filePath := path.Clean(ps.ByName("filepath"))

	challenge := challengeRepository.Get(challengeID)
	if challenge == nil {
		renderChallengeNotFound(w, r, challengeID)
		return
	}

	if !challenge.Accessible(r) {
		if challenge.HasPassword {
			csrfToken, err := getOrCreateCSRF(w, r)
			if err != nil {
				log.Printf("Error with getOrCreateCSRF: %v", err)
				renderServerError(w, r, err)
				return
			}

			w.WriteHeader(http.StatusUnauthorized)
			templates.WritePageTemplate(w, &templates.UnlockPage{
				Challenge: challenge,
				CSRF:      csrfToken,
			}, &templates.EmptyNav{})
		} else {
			renderUnauthorized(w, r)
		}

		return
	}

	challengeBasePath := path.Join(dataDirectory, path.Clean(challenge.SharedPath))
	dir := http.Dir(challengeBasePath)
	file, err := dir.Open(filePath)
	if err != nil {
		log.Printf("Error opening file %v: %v", filePath, err)
		renderServerError(w, r, err)
		return
	}

	stat, err := file.Stat()
	if err != nil {
		log.Printf("Error stat'ing file %v: %v", filePath, err)
		renderServerError(w, r, err)
		return
	}

	if !stat.IsDir() {
		http.ServeFile(w, r, path.Join(challengeBasePath, filePath))
		return
	}

	dirs, err := file.Readdir(-1)
	if err != nil {
		log.Printf("Error reading directory %v: %v", filePath, err)
		renderServerError(w, r, err)
		return
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })

	files := make([]templates.File, len(dirs))
	for i, dir := range dirs {
		name := dir.Name()
		if dir.IsDir() {
			name += "/"
		}

		browseURL := url.URL{Path: "/view/" + challenge.ID + "/" + path.Join(filePath, name)}

		files[i].Label = name
		files[i].BrowseLink = browseURL.String()
	}

	upwardsURL := url.URL{Path: "/view/" + challenge.ID + "/" + path.Join(filePath, "..")}

	atRoot := filePath == "" || filePath == "/" || filePath == "."
	directoryName := filePath
	if atRoot {
		directoryName = "/"
	}

	browsePage := &templates.BrowsePage{
		DirectoryName: directoryName,
		Files:         files,

		CanTravelUpwards: !atRoot,
		UpwardsLink:      upwardsURL.String(),
	}
	templates.WritePageTemplate(w, browsePage, &templates.EmptyNav{})
}

func handleChallengeAuthentication(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	challengeID := ps.ByName("challenge")

	challenge := challengeRepository.Get(challengeID)
	if challenge == nil {
		renderChallengeNotFound(w, r, challengeID)
		return
	}

	// already has access, no need for auth
	if challenge.Accessible(r) {
		http.Redirect(w, r, r.URL.String(), http.StatusFound)
		return
	}

	if err := validCSRF(r, r.FormValue("_token")); err != nil {
		log.Printf("Error validating CSRF token: %v", err)
		renderServerError(w, r, err)
		return
	}

	if challenge.HasPassword {
		postedPassword := r.FormValue("challenge-password")
		if challenge.CheckPassword(postedPassword) == nil {
			challenge.StorePassword(postedPassword, w, r)
			http.Redirect(w, r, r.URL.String(), http.StatusFound)
			return
		}
	}

	handleChallengeFilepath(w, r, ps)
}

func handleHome(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	templates.WritePageTemplate(w, &templates.HomePage{}, &templates.PrivateNav{})
}

func main() {
	router := httprouter.New()

	router.GET("/", handleHome)

	router.GET("/challenges", handleChallengesIndex)

	router.GET("/stuff/browse/*filepath", handleStuffIndex)
	router.GET("/stuff/share/*filepath", handleStuffShowForm)
	router.POST("/stuff/share/*filepath", handleStuffReceiveForm)

	router.GET("/view/:challenge", handleChallengeFilepath)
	router.GET("/view/:challenge/*filepath", handleChallengeFilepath)
	router.POST("/view/:challenge", handleChallengeAuthentication)
	router.POST("/view/:challenge/*filepath", handleChallengeAuthentication)

	log.Fatal(http.ListenAndServe(":8080", router))
}
