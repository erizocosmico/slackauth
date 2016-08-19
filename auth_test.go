package slackauth

import (
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
)

type slackAPIMock struct{}

func (*slackAPIMock) GetOAuthResponse(id, secret, code string, debug bool) (*slack.OAuthResponse, error) {
	if code == "invalid" {
		return nil, errors.New("invalid code")
	}

	return &slack.OAuthResponse{
		AccessToken: "foo",
	}, nil
}

const (
	tplSuccess = `<h1>Hello</h1>
	<p>All went ok!</p>`
	tplError = `<h1>:(</h1>
	<p>Something went wrong!</p>`
)

func TestNew(t *testing.T) {
	assert.Nil(t, ioutil.WriteFile("valid.txt", []byte("foo"), 0777))

	cases := []struct {
		options Options
		err     bool
	}{
		{Options{}, true},
		{Options{Addr: "", ClientID: "a", ClientSecret: "b"}, true},
		{Options{
			Addr:         ":8080",
			ClientID:     "foo",
			ClientSecret: "bar",
			SuccessTpl:   "invalid.txt",
			ErrorTpl:     "bar.txt",
		}, true},
		{Options{
			Addr:         ":8080",
			ClientID:     "foo",
			ClientSecret: "bar",
			SuccessTpl:   "valid.txt",
			ErrorTpl:     "invalid.txt",
		}, true},
		{Options{
			Addr:         ":8080",
			ClientID:     "foo",
			ClientSecret: "bar",
			SuccessTpl:   "valid.txt",
			ErrorTpl:     "valid.txt",
		}, false},
	}

	for _, c := range cases {
		_, err := New(c.options)
		if c.err {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
	}

	assert.Nil(t, os.Remove("valid.txt"))
}

func TestSlackAuth(t *testing.T) {
	successTpl := template.Must(template.New("success").Parse(tplSuccess))
	errorTpl := template.Must(template.New("error").Parse(tplError))
	auth := &slackAuth{
		clientID:     "aaaa",
		clientSecret: "bbbb",
		addr:         ":8989",
		successTpl:   successTpl,
		errorTpl:     errorTpl,
		debug:        true,
		certFile:     "",
		keyFile:      "",
		auths:        make(chan *slack.OAuthResponse, 1),
		api:          &slackAPIMock{},
	}
	auth.SetLogOutput(os.Stdout)
	go auth.Run()

	<-time.After(50 * time.Millisecond)

	// This will not trigger an OnAuth event
	testRequest(t, "fooo", tplSuccess)
	testRequest(t, "invalid", tplError)

	var auths int
	auth.OnAuth(func(auth *slack.OAuthResponse) {
		auths++
	})
	testRequest(t, "fooo", tplSuccess)
	testRequest(t, "bar", tplSuccess)
	assert.Equal(t, 2, auths)
}

func testRequest(t *testing.T, code string, expected string) {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:8989/?code=%s", code))
	assert.Nil(t, err)
	bytes, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, expected, string(bytes))
}
