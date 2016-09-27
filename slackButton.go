package slackauth

import (
	"net/http"

	log15 "gopkg.in/inconshreveable/log15.v2"
)

const (
	// BOT scope grant permission to add the bot bundled by the app
	BOT = "bot"
	// WEBHOOK scope allows to request permission to post content into the user's Slack team
	WEBHOOK = "incoming-webhook"
	// COMMANDS scope allows to install slash commands bundled in the Slack app
	COMMANDS = "commands"
)

func (s *slackAuth) buttonHandler(w http.ResponseWriter, r *http.Request) {
	templateScope := map[string]string{
		"Scopes":   s.scopes,
		"ClientId": s.clientID,
	}
	if err := s.buttonTpl.Execute(w, templateScope); err != nil {
		log15.Error("error displaying button tpl", "err", err.Error())
	}
}
