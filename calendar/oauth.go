package calendar

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

var (
	gcp                *gcpAuthWrapper
	oauthRedirectURL   = "http://localhost:8080"
	oauthTokenFilename = "calender.oauthToken.cache"
)

type server struct {
	Server *http.Server
	Config *oauth2.Config
	Token  *oauth2.Token
}

func (s *server) oauthHandler(w http.ResponseWriter, r *http.Request) {
	permissionCode := r.URL.Query().Get("code")
	ctx := context.Background()
	tok, err := s.Config.Exchange(ctx, permissionCode)
	if err != nil {
		panic(err)
	}
	s.Token = tok

	// save tok
	d, _ := os.UserCacheDir()
	of, err := os.Create(filepath.Join(d, oauthTokenFilename))
	if err != nil {
		log.Panic().Err(err).Msg("failed to retrieve the oauth2 token")
	}
	defer of.Close()
	if err = json.NewEncoder(of).Encode(tok); err != nil {
		log.Panic().Err(err).Msg("Something went wrong when storing the token source")
	}

	http.Redirect(w, r, "http://google.com", http.StatusTemporaryRedirect)
	go func() {
		time.Sleep(time.Second * 2)
		s.Server.Shutdown(context.Background())
	}()
}

type gcpAuthWrapper struct {
	Config *oauth2.Config
}

func NewGCPAuthWrapper() *gcpAuthWrapper {
	if gcp != nil {
		return gcp
	}
	gcp = &gcpAuthWrapper{}
	return gcp
}

func (w *gcpAuthWrapper) Auth(credPath string) (*oauth2.Token, error) {
	f, err := os.Open(credPath)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	defer f.Close()

	b, err := ioutil.ReadFile(credPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed read credential")
	}

	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		return nil, errors.Wrap(err, "failed read credential")
	}

	w.Config = config
	// check if we have an oauth file on disk
	if hasCachedOauth() {
		tok, err := loadTokenSource()
		if err == nil {
			// ok
			log.Info().Msg("You have successfully authenticated")
			return tok, nil
		}
		log.Info().Str("error", err.Error()).Msg("Failed to load the token source")
	}

	config.RedirectURL = oauthRedirectURL
	url := w.Config.AuthCodeURL("state", oauth2.AccessTypeOffline)

	// There are no plans to support Windows.
	if runtime.GOOS != "darwin" {
		cmd := exec.Command("xdg-open", url)
		cmd.Run()
	} else {
		cmd := exec.Command("open", url)
		cmd.Run()
	}
	oauthSrv := &http.Server{Addr: ":8080", Handler: http.DefaultServeMux}
	server := &server{
		Server: oauthSrv,
		Config: w.Config,
	}

	http.HandleFunc("/", server.oauthHandler)
	if err = server.Server.ListenAndServe(); err != http.ErrServerClosed {
		return nil, errors.Wrap(err, "listen: %s")
	}
	log.Info().Msg("You have successfully authenticated")
	return server.Token, nil
}

func hasCachedOauth() bool {
	d, _ := os.UserCacheDir()
	if _, err := os.Stat(filepath.Join(d, oauthTokenFilename)); os.IsNotExist(err) {
		return false
	}
	return true
}

func loadTokenSource() (*oauth2.Token, error) {
	d, _ := os.UserCacheDir()
	f, err := os.Open(filepath.Join(d, oauthTokenFilename))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load the token source (deleted from disk)")
	}
	defer f.Close()
	var token oauth2.Token
	if err = json.NewDecoder(f).Decode(&token); err != nil {
		return nil, errors.Wrap(err, "failed json decode")
	}
	return &token, nil
}
