package slackauth

import (
	"errors"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nlopes/slack"

	log15 "gopkg.in/inconshreveable/log15.v2"
)

const (
	// BOT scope grants permission to add the bot bundled by the app
	BOT = "bot"
	// WEBHOOK scope  allows requesting permission to post content to the user's Slack team
	WEBHOOK = "incoming-webhook"
	// COMMANDS scope allows installing slash commands bundled in the Slack app
	COMMANDS = "commands"
)

// Service is a service to authenticate on slack using the "Add to slack" button.
type Service interface {
	// SetLogOutput sets the place where logs will be written.
	SetLogOutput(io.Writer)

	// Run will run the service. This method blocks until the service crashes or stops.
	Run() error

	// OnAuth sets the handler that will be triggered every time someone authorizes slack
	// successfully.
	OnAuth(func(*slack.OAuthResponse))
}

type slackAPI interface {
	GetOAuthResponse(string, string, string, bool) (*slack.OAuthResponse, error)
}

type slackAPIWrapper struct{}

func (*slackAPIWrapper) GetOAuthResponse(id, secret, code string, debug bool) (*slack.OAuthResponse, error) {
	if debug {
		slack.SetLogger(log.New(os.Stdout, "", log.LstdFlags))
	}
	return slack.GetOAuthResponse(id, secret, code, "", debug)
}

type slackAuth struct {
	clientID     string
	clientSecret string
	addr         string
	certFile     string
	keyFile      string
	successTpl   *template.Template
	errorTpl     *template.Template
	debug        bool
	auths        chan *slack.OAuthResponse
	callback     func(*slack.OAuthResponse)
	api          slackAPI
	buttonTpl    *template.Template
	scopes       string
}

// Options has all the configurable parameters for slack authenticator.
type Options struct {
	// Addr is the address where the service will run. e.g: :8080, 0.0.0.0:8989, etc.
	Addr string
	// ClientID is the slack client ID provided to you in your app credentials.
	ClientID string
	// ClientSecret is the slack client secret provided to you in your app credentials.
	ClientSecret string
	// SuccessTpl is the path to the template that will be displayed when there is a successful
	// auth.
	SuccessTpl string
	// ErrorTpl is the path to the template that will be displayed when there is an invalid
	// auth.
	ErrorTpl string
	// Debug will print some debug logs.
	Debug bool
	// CertFile is the path to the SSL certificate file. If this and KeyFile are provided, the
	// server will be run with SSL.
	CertFile string
	// KeyFile is the path to the SSL certificate key file. If this and CertFile are provided, the
	// server will be run with SSL.
	KeyFile string
	// ButtonTpl is the path to the Slack button template
	ButtonTpl string
	// Scopes is the list of the allowed scopes
	Scopes []string
}

// New creates a new slackauth service.
func New(opts Options) (Service, error) {
	if opts.Addr == "" || opts.ClientID == "" || opts.ClientSecret == "" {
		return nil, errors.New("slackauth: addr, client id and client secret can not be empty")
	}

	successTpl, err := readTemplate(opts.SuccessTpl)
	if err != nil {
		return nil, err
	}

	errorTpl, err := readTemplate(opts.ErrorTpl)
	if err != nil {
		return nil, err
	}

	slackAuthService := &slackAuth{
		clientID:     opts.ClientID,
		clientSecret: opts.ClientSecret,
		addr:         opts.Addr,
		successTpl:   successTpl,
		errorTpl:     errorTpl,
		debug:        opts.Debug,
		certFile:     opts.CertFile,
		keyFile:      opts.KeyFile,
		auths:        make(chan *slack.OAuthResponse, 1),
		api:          &slackAPIWrapper{},
	}

	err = slackAuthService.configureButton(opts.ButtonTpl, opts.Scopes)
	if err != nil {
		return nil, err
	}
	return slackAuthService, nil
}

func (s *slackAuth) configureButton(buttonTpl string, scopes []string) error {
	if len(buttonTpl) > 0 {
		buttonTpl, err := readTemplate(buttonTpl)
		if err != nil {
			return err
		}

		if len(scopes) == 0 {
			return errors.New("At least one scope needed")
		}

		s.scopes = strings.Join(scopes, ",")
		s.buttonTpl = buttonTpl
	}

	return nil
}

func (s *slackAuth) Run() error {
	go func() {
		for auth := range s.auths {
			if s.callback != nil {
				s.callback(auth)
			} else {
				log15.Warn("auth event triggered but there was no handler")
			}
		}
	}()

	log15.Info("Starting server", "addr", s.addr)
	return s.runServer()
}

func (s *slackAuth) SetLogOutput(w io.Writer) {
	var nilWriter io.Writer

	var format = log15.LogfmtFormat()
	if w == nilWriter || w == nil {
		w = os.Stdout
		format = log15.TerminalFormat()
	}

	var maxLvl = log15.LvlInfo
	if s.debug {
		maxLvl = log15.LvlDebug
	}

	log15.Root().SetHandler(log15.LvlFilterHandler(maxLvl, log15.StreamHandler(w, format)))
}

func (s *slackAuth) OnAuth(fn func(*slack.OAuthResponse)) {
	s.callback = fn
}

func (s *slackAuth) runServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.buttonHandler)
	mux.HandleFunc("/auth", s.authorizationHandler)

	srv := &http.Server{
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 3 * time.Second,
		Addr:         s.addr,
		Handler:      mux,
	}

	if s.certFile != "" && s.keyFile != "" {
		return srv.ListenAndServeTLS(s.certFile, s.keyFile)
	}

	return srv.ListenAndServe()
}

func (s *slackAuth) authorizationHandler(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	resp, err := s.api.GetOAuthResponse(s.clientID, s.clientSecret, code, s.debug)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		log15.Error("error getting oauth response", "err", err.Error())
		if err := s.errorTpl.Execute(w, resp); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log15.Error("error displaying error tpl", "err", err.Error())
		}

		return
	}

	if err := s.successTpl.Execute(w, resp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log15.Error("error displaying success tpl", "err", err.Error())
	}

	log15.Debug("successful authorization", "team", resp.TeamName, "team id", resp.TeamID)
	s.auths <- resp
}

func (s *slackAuth) buttonHandler(w http.ResponseWriter, r *http.Request) {
	templateScope := map[string]string{
		"Scopes":   s.scopes,
		"ClientId": s.clientID,
	}
	if err := s.buttonTpl.Execute(w, templateScope); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log15.Error("error displaying button tpl", "err", err.Error())
	}
}

func readTemplate(file string) (*template.Template, error) {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return template.New("").Parse(string(bytes))
}
