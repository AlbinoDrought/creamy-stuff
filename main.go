package main

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"path"
	"sort"

	"github.com/julienschmidt/httprouter"
)

const challengeIDLength = 64
const challengeRandomPasswordLength = 128

var dataDirectory = "data"

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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	fmt.Fprintln(w, "<ul>")
	for _, d := range dirs {
		name := d.Name()
		if d.IsDir() {
			name += "/"
		}

		fmt.Fprintln(w, "<li>")

		browseURL := url.URL{Path: name}
		fmt.Fprintf(w, "<a href=\"%s\">%s</a>\n", browseURL.String(), html.EscapeString(name))

		shareURL := url.URL{Path: "/stuff/share/" + name}
		fmt.Fprintf(w, "(<a href=\"%s\">share</a>)\n", shareURL.String())

		fmt.Fprintln(w, "</li>")
	}
	fmt.Fprintln(w, "</ul>")
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	fmt.Fprintln(w, "<form method=\"POST\">")
	fmt.Fprintf(w, "<input type=\"hidden\" name=\"_token\" value=\"%s\">\n", html.EscapeString(csrfToken))
	fmt.Fprintf(w, "<div><label for=\"public\">Public <input type=\"checkbox\" name=\"public\" value=\"1\"></label></div>")
	fmt.Fprintf(w, "<div><label for=\"challenge-password\">Password</label> <input type=\"text\" name=\"challenge-password\" value=\"%s\"></div>", randomPassword)
	fmt.Fprintln(w, "<div><button type=\"submit\">Share</button></div>")
	fmt.Fprintln(w, "</form>")
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

	challenge := &Challenge{
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	challengeURL := url.URL{Path: "/view/" + challenge.ID}
	challengeURLString := html.EscapeString(challengeURL.String())

	fmt.Fprintf(w, "<a href=\"%s\">%s</a>", challengeURLString, challengeURLString)
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

func main() {
	router := httprouter.New()

	router.GET("/stuff/browse/*filepath", handleStuffIndex)
	router.GET("/stuff/share/*filepath", handleStuffShowForm)
	router.POST("/stuff/share/*filepath", handleStuffReceiveForm)

	router.GET("/view/:challenge", handleChallengeFilepath)
	router.GET("/view/:challenge/*filepath", handleChallengeFilepath)
	router.POST("/view/:challenge", handleChallengeAuthentication)
	router.POST("/view/:challenge/*filepath", handleChallengeAuthentication)

	log.Fatal(http.ListenAndServe(":8080", router))
}
