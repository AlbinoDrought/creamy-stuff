package main

//go:generate go get -u github.com/valyala/quicktemplate/qtc
//go:generate qtc -dir=templates

import (
	"fmt"
	"html"
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

	challengeRepository.Set(&stuff.Challenge{
		ID:         "bar",
		Public:     false,
		SharedPath: "data-private",
	})
}

func handleStuffIndex(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	filePath := path.Clean(ps.ByName("filepath"))

	dir := http.Dir(dataDirectory)
	file, err := dir.Open(filePath)
	if err != nil {
		log.Printf("Error opening file %v: %v", filePath, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
		return
	}

	stat, err := file.Stat()
	if err != nil {
		log.Printf("Error stat'ing file %v: %v", filePath, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
		return
	}

	if !stat.IsDir() {
		http.ServeFile(w, r, path.Join(dataDirectory, filePath))
		return
	}

	dirs, err := file.Readdir(-1)
	if err != nil {
		log.Printf("Error reading directory %v: %v", filePath, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
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

	atRoot := filePath == "" || filePath == "/"
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
	templates.WritePageTemplate(w, browsePage)
}

func handleStuffShowForm(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	filePath := path.Clean(ps.ByName("filepath"))

	dir := http.Dir(dataDirectory)
	_, err := dir.Open(filePath)
	if err != nil {
		log.Printf("Error opening file %v: %v", filePath, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
		return
	}

	csrfToken, err := getOrCreateCSRF(w, r)
	if err != nil {
		log.Printf("Error with getOrCreateCSRF: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
		return
	}

	randomPassword, err := RandomString(challengeRandomPasswordLength)
	if err != nil {
		log.Printf("Error generating random challenge password: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
		return
	}

	cancelURL := url.URL{Path: "/stuff/browse" + path.Join(filePath, "..")}

	sharePage := &templates.SharePage{
		Path:           filePath,
		CSRF:           csrfToken,
		RandomPassword: randomPassword,

		CancelLink: cancelURL.String(),
	}
	templates.WritePageTemplate(w, sharePage)
}

func handleStuffReceiveForm(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	filePath := path.Clean(ps.ByName("filepath"))

	dir := http.Dir(dataDirectory)
	_, err := dir.Open(filePath)
	if err != nil {
		log.Printf("Error opening file %v: %v", filePath, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
		return
	}

	if err := validCSRF(r, r.FormValue("_token")); err != nil {
		log.Printf("Error validating CSRF token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
		return
	}

	challengeID, err := RandomString(challengeIDLength)
	if err != nil {
		log.Printf("Error generating challenge ID: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
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
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 Internal Server Error"))
			return
		}
	}

	challengeRepository.Set(challenge)

	challengeURL := url.URL{Path: "/view/" + challenge.ID}

	sharedChallengePage := &templates.SharedChallengePage{
		Challenge: challenge,

		ViewLink: challengeURL.String(),
	}
	templates.WritePageTemplate(w, sharedChallengePage)
}

func handleChallengeFilepath(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	challengeID := ps.ByName("challenge")
	filePath := ps.ByName("filepath")

	challenge := challengeRepository.Get(challengeID)
	if challenge == nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Challenge not found"))
		return
	}

	if !challenge.Accessible(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)

		if challenge.HasPassword {
			csrfToken, err := getOrCreateCSRF(w, r)
			if err != nil {
				log.Printf("Error with getOrCreateCSRF: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("500 Internal Server Error"))
				return
			}

			fmt.Fprintln(w, "<form method=\"POST\">")
			fmt.Fprintf(w, "<input type=\"hidden\" name=\"_token\" value=\"%s\">\n", html.EscapeString(csrfToken))
			fmt.Fprintf(w, "<div><label for=\"challenge-password\">Password</label> <input type=\"text\" name=\"challenge-password\"></div>")
			fmt.Fprintln(w, "<div><button type=\"submit\">Unlock</button></div>")
			fmt.Fprintln(w, "</form>")
		} else {
			w.Write([]byte("Unauthorized"))
		}

		return
	}

	// force trailing slash
	// todo: fix direct file link bug
	if filePath == "" {
		http.Redirect(w, r, r.URL.String()+"/", http.StatusFound)
		return
	}

	http.StripPrefix(
		"/view/"+challenge.ID+"/",
		http.FileServer(http.Dir(
			path.Join(dataDirectory, challenge.SharedPath),
		)),
	).ServeHTTP(w, r)
}

func handleChallengeAuthentication(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	challengeID := ps.ByName("challenge")

	challenge := challengeRepository.Get(challengeID)
	if challenge == nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Challenge not found"))
		return
	}

	// already has access, no need for auth
	if challenge.Accessible(r) {
		http.Redirect(w, r, r.URL.String(), http.StatusFound)
		return
	}

	if err := validCSRF(r, r.FormValue("_token")); err != nil {
		log.Printf("Error validating CSRF token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
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

	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("401 Unauthorized"))
}

func handleHome(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	templates.WritePageTemplate(w, &templates.HomePage{})
}

func main() {
	router := httprouter.New()

	router.GET("/", handleHome)

	router.GET("/stuff/browse/*filepath", handleStuffIndex)
	router.GET("/stuff/share/*filepath", handleStuffShowForm)
	router.POST("/stuff/share/*filepath", handleStuffReceiveForm)

	router.GET("/view/:challenge", handleChallengeFilepath)
	router.GET("/view/:challenge/*filepath", handleChallengeFilepath)
	router.POST("/view/:challenge", handleChallengeAuthentication)
	router.POST("/view/:challenge/*filepath", handleChallengeAuthentication)

	log.Fatal(http.ListenAndServe(":8080", router))
}
