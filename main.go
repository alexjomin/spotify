package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/alexjomin/spotify/storage"
	"github.com/alexjomin/spotify/storage/bolt"
	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"github.com/zmb3/spotify"
)

var auth spotify.Authenticator
var client spotify.Client
var s storage.Storage
var logged bool
var lastChannel chan string
var lastValue string

type config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	DeviceID     string
	Port         string
}

type setPayload struct {
	URI string `json:"uri"`
}

var c config

func main() {
	err := envconfig.Process("spotify", &c)
	if err != nil {
		log.Fatal(err.Error())
	}

	s, err = bolt.New("./db", "spotify")

	if err != nil {
		log.Fatal(err)
	}

	lastChannel = make(chan string)

	go func() {
		for value := range lastChannel {
			lastValue = value
		}
	}()

	r := mux.NewRouter()
	r.HandleFunc("/callback", redirectHandler).Methods(http.MethodGet)
	r.HandleFunc("/play/{id}", play).Methods(http.MethodGet)
	r.HandleFunc("/set-last", setLast).Methods(http.MethodPut)
	r.HandleFunc("/set/{id}", set).Methods(http.MethodPut)
	r.HandleFunc("/login", login).Methods(http.MethodGet)
	http.Handle("/", r)

	log.Println("listen on", c.Port)
	log.Fatal(http.ListenAndServe(c.Port, nil))
}

func play(w http.ResponseWriter, r *http.Request) {

	if !logged {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	vars := mux.Vars(r)
	var content setPayload

	p, err := s.Get(vars["id"])

	json.Unmarshal(p, &content)

	// send last id to the buffer via a channel
	go func(id string) {
		lastChannel <- id
	}(vars["id"])

	if err != nil {
		http.Error(w, "Can't find the request id", http.StatusNotFound)
		return
	}

	uri := spotify.URI(content.URI)

	logrus.WithField("uri", content.URI).Info("Play")

	var deviceID spotify.ID
	deviceID = spotify.ID(c.DeviceID)

	opt := spotify.PlayOptions{
		DeviceID:        &deviceID,
		PlaybackContext: &uri,
	}

	err = client.PlayOpt(&opt)

	logrus.
		WithField("uri", content.URI).
		Info("Last uri value strored")

	if err != nil {
		http.Error(w, "Error playing album", http.StatusInternalServerError)
		log.Println(err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func set(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	var p setPayload

	if r.Body == nil {
		http.Error(w, "Please send a request body", http.StatusBadRequest)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	err = s.Insert(vars["id"], p)

	if err != nil {
		http.Error(w, "Can't store data", http.StatusInternalServerError)
		return
	}

	logrus.
		WithField("key", lastValue).
		WithField("uri", p.URI).
		Info("Stored")

	w.WriteHeader(http.StatusNoContent)
}

func setLast(w http.ResponseWriter, r *http.Request) {
	if lastValue == "" {
		http.Error(w, "Not value set", http.StatusConflict)
		return
	}

	var p setPayload

	if r.Body == nil {
		http.Error(w, "Please send a request body", http.StatusBadRequest)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	err = s.Insert(string(lastValue), p)

	if err != nil {
		http.Error(w, "Can't store data", http.StatusInternalServerError)
		return
	}

	logrus.
		WithField("key", lastValue).
		WithField("uri", p.URI).
		Info("Stored")

	w.WriteHeader(http.StatusNoContent)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	token, err := auth.Token("", r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusNotFound)
		return
	}
	client = auth.NewClient(token)
	logged = true
	fmt.Fprint(w, "Auth Ok")
}

func login(w http.ResponseWriter, r *http.Request) {
	auth = spotify.NewAuthenticator(c.RedirectURI,
		spotify.ScopeUserReadPrivate,
		spotify.ScopeUserReadPlaybackState,
		spotify.ScopeUserModifyPlaybackState)

	auth.SetAuthInfo(c.ClientID, c.ClientSecret)
	url := auth.AuthURL("")
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}
