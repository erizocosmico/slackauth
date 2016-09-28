package slackauth

import (
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
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
	tplSlackButton = `ADD ME:
		<a href="https://slack.com/oauth/authorize?scope={{.Scopes}}&client_id={{.ClientId}}">
			SLACK BUTTON
		</a>`
	slackButtonRegExp = "<a[^>]+href=\"https://slack.com/oauth/authorize\\?" +
		"scope=([^&\"]+)[^\"]*&client_id=([^&\"]+)[^\"]*\"[^>]*>[\\s\\S]*</a>"
)

var slackButtonMatcher = regexp.MustCompile(slackButtonRegExp)

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
			ButtonTpl:    "invalid.txt",
			Scopes:       []string{},
		}, true},
		{Options{
			Addr:         ":8080",
			ClientID:     "foo",
			ClientSecret: "bar",
			SuccessTpl:   "valid.txt",
			ErrorTpl:     "invalid.txt",
			ButtonTpl:    "invalid.txt",
			Scopes:       []string{},
		}, true},
		{Options{
			Addr:         ":8080",
			ClientID:     "foo",
			ClientSecret: "bar",
			SuccessTpl:   "valid.txt",
			ErrorTpl:     "valid.txt",
			ButtonTpl:    "",
			Scopes:       []string{},
		}, false},
		{Options{
			Addr:         ":8080",
			ClientID:     "foo",
			ClientSecret: "bar",
			SuccessTpl:   "valid.txt",
			ErrorTpl:     "valid.txt",
			ButtonTpl:    "invalid.txt",
			Scopes:       []string{},
		}, true},
		{Options{
			Addr:         ":8080",
			ClientID:     "foo",
			ClientSecret: "bar",
			SuccessTpl:   "valid.txt",
			ErrorTpl:     "valid.txt",
			ButtonTpl:    "valid.txt",
			Scopes:       []string{},
		}, true},
		{Options{
			Addr:         ":8080",
			ClientID:     "foo",
			ClientSecret: "bar",
			SuccessTpl:   "valid.txt",
			ErrorTpl:     "valid.txt",
			ButtonTpl:    "valid.txt",
			Scopes:       []string{BOT},
		}, false},
	}

	for i, c := range cases {
		_, err := New(c.options)
		errorHint := fmt.Sprintf("fail in testcase #%d %#v", i, c.options)
		if c.err {
			assert.NotNil(t, err, errorHint)
		} else {
			assert.Nil(t, err, errorHint)
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
	testRequest(t, getURLForAuth("fooo"), tplSuccess)
	testRequest(t, getURLForAuth("invalid"), tplError)

	var auths int
	auth.OnAuth(func(auth *slack.OAuthResponse) {
		auths++
	})
	testRequest(t, getURLForAuth("fooo"), tplSuccess)
	testRequest(t, getURLForAuth("bar"), tplSuccess)
	assert.Equal(t, 2, auths)
}

func getURLForAuth(code string) string {
	return fmt.Sprintf("http://127.0.0.1:8989/auth?code=%s", code)
}

func testRequest(t *testing.T, url string, expected string) {
	assert.Equal(t, expected, string(getBody(t, url)))
}

func getBody(t *testing.T, url string) []byte {
	resp, err := http.Get(url)
	assert.Nil(t, err)
	bytes, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	return bytes
}

type buttonOptions struct {
	Scopes   []string
	ClientId string
}

func TestSlackButton(t *testing.T) {
	buttonOpts := buttonOptions{
		Scopes:   []string{BOT, COMMANDS},
		ClientId: "client-id",
	}

	assert.Nil(t, ioutil.WriteFile("valid.txt", []byte(tplSlackButton), 0777))

	auth, err := New(Options{
		Addr:         ":8080",
		ClientID:     buttonOpts.ClientId,
		ClientSecret: "bar",
		SuccessTpl:   "valid.txt",
		ErrorTpl:     "valid.txt",
		ButtonTpl:    "valid.txt",
		Scopes:       buttonOpts.Scopes,
	})
	assert.Nil(t, err)

	go auth.Run()
	<-time.After(5 * time.Millisecond)

	servedButtonCode := getBody(t, "http://127.0.0.1:8080/")
	matches := slackButtonMatcher.FindStringSubmatch(string(servedButtonCode))
	assert.NotNil(t, matches)
	assert.Equal(t, 3, len(matches))
	receibedScopes, _ := url.QueryUnescape(matches[1])
	assert.Equal(t, strings.Join(buttonOpts.Scopes, ","), receibedScopes)
	assert.Equal(t, buttonOpts.ClientId, matches[2])

	assert.Nil(t, os.Remove("valid.txt"))
}
