package main

import (
	"log"
	"os"

	log15 "gopkg.in/inconshreveable/log15.v2"

	"github.com/mvader/slackauth"
	"github.com/nlopes/slack"
)

func main() {
	service, err := slackauth.New(slackauth.Options{
		Addr:         ":8080",
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		SuccessTpl:   "success.html",
		ErrorTpl:     "error.html",
		Debug:        true,
	})
	if err != nil {
		log.Fatal(err)
	}

	service.OnAuth(func(auth *slack.OAuthResponse) {
		log15.Info("someone was authorized!", "team", auth.TeamName)
	})

	log.Fatal(service.Run())
}
