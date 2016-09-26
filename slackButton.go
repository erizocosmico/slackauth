package slackauth

import (
	"html/template"
	"net/http"
	"strings"

	log15 "gopkg.in/inconshreveable/log15.v2"
)

const (
	//BOOT scope grant permission to add the bot bundled by the app
	BOT = "bot"
	//WEBHOOK scope allows to request permission to post content into the user's Slack team
	WEBHOOK = "incoming-webhook"
	//COMMANDS scope allows to install slash commands bundled in the Slack app
	COMMANDS = "commands"
)

// SlackButtonOptions has all the configurable parameters for Slack button.
type SlackButtonOptions struct {
	//Scopes is the list of the allowed scopes
	Scopes []string
	//ClientId for the app bot
	ClientID string
	//ButtonTpl is the path to the Slack button template
	ButtonTpl string
}

type slackButtonHandler struct {
	options  *SlackButtonOptions
	template *template.Template
}

//HTTPHandler is an HTTP Handler that will serve the Slack Button
type HTTPHandler func(w http.ResponseWriter, r *http.Request)

//GetButtonHandler will return an HTTPHandler
func GetButtonHandler(options *SlackButtonOptions) (HTTPHandler, error) {
	successTpl, err := readTemplate(options.ButtonTpl)
	if err != nil {
		return nil, err
	}
	handler := &slackButtonHandler{options, successTpl}

	return handler.get, nil
}

func (s *slackButtonHandler) get(w http.ResponseWriter, r *http.Request) {
	templateScope := map[string]string{
		"Scopes":   strings.Join(s.options.Scopes, ","),
		"ClientId": s.options.ClientID,
	}
	if err := s.template.Execute(w, templateScope); err != nil {
		log15.Error("error displaying button tpl", "err", err.Error())
	}
}
