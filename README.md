<img src="https://rawgit.com/mvader/slackauth/master/logo.svg" alt="slackauth" />

[![godoc reference](https://cdn.rawgit.com/mvader/2faf5060e6cb109617ef5548836532aa/raw/2f5e2f2e934f6dde4ec4652ff0ae6d5c83cbfd6a/godoc.svg)](https://godoc.org/github.com/mvader/slackauth) [![Build Status](https://travis-ci.org/mvader/slackauth.svg?branch=master)](https://travis-ci.org/mvader/slackauth) [![codecov](https://codecov.io/gh/mvader/slackauth/branch/master/graph/badge.svg)](https://codecov.io/gh/mvader/slackauth) [![License](http://img.shields.io/:license-mit-blue.svg)](http://doge.mit-license.org)

**slackauth** is a package to implement the ["Add to Slack"](https://api.slack.com/docs/slack-button) button functionality in an easy way.

## Install

```
go get gopkg.in/mvader/slackauth.v1
```

## Example

```go
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
```

[See full example in the examples folder](https://github.com/mvader/slackauth/tree/master/examples).
