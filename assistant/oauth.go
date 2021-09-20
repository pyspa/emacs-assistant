package assistant

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

var (
	oauthToken         *oauth2.Token
	gcp                *gcpAuthWrapper
	oauthSrv           *http.Server
	oauthRedirectURL   = "http://localhost:8080"
	oauthTokenFilename = "oauthToken.cache"
)

type JSONToken struct {
	Installed struct {
		ClientID                string   `json:"client_id"`
		ProjectID               string   `json:"project_id"`
		AuthURI                 string   `json:"auth_uri"`
		TokenURI                string   `json:"token_uri"`
		AuthProviderX509CertURL string   `json:"auth_provider_x509_cert_url"`
		ClientSecret            string   `json:"client_secret"`
		RedirectUris            []string `json:"redirect_uris"`
	} `json:"installed"`
}

type gcpAuthWrapper struct {
	Conf *oauth2.Config
}

func NewGCPAuthWrapper() *gcpAuthWrapper {
	if gcp != nil {
		return gcp
	}
	gcp = &gcpAuthWrapper{}
	return gcp
}

func (w *gcpAuthWrapper) Auth(credPath string) error {
	f, err := os.Open(credPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var token JSONToken
	if err = json.NewDecoder(f).Decode(&token); err != nil {
		return errors.Wrap(err, "failed to decode json token")
	}
	w.Conf = &oauth2.Config{
		ClientID:     token.Installed.ClientID,
		ClientSecret: token.Installed.ClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/assistant-sdk-prototype"},
		RedirectURL:  oauthRedirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://accounts.google.com/o/oauth2/token",
		},
	}

	// check if we have an oauth file on disk
	if hasCachedOauth() {
		err = loadTokenSource()
		if err == nil {
			// ok
			log.Info().Msg("You have successfully authenticated")
			return nil
		}
		log.Info().Str("error", err.Error()).Msg("Failed to load the token source")
	}

	url := w.Conf.AuthCodeURL("state", oauth2.AccessTypeOffline)

	// There are no plans to support Windows.
	if runtime.GOOS != "darwin" {
		cmd := exec.Command("xdg-open", url)
		cmd.Run()
	} else {
		cmd := exec.Command("open", url)
		cmd.Run()
	}
	oauthSrv = &http.Server{Addr: ":8080", Handler: http.DefaultServeMux}
	http.HandleFunc("/", oauthHandler)
	if err = oauthSrv.ListenAndServe(); err != http.ErrServerClosed {
		return errors.Wrap(err, "listen: %s")
	}
	log.Info().Msg("You have successfully authenticated")
	return nil
}

func oauthHandler(w http.ResponseWriter, r *http.Request) {
	permissionCode := r.URL.Query().Get("code")
	setTokenSource(permissionCode)
	http.Redirect(w, r, "http://google.com", http.StatusTemporaryRedirect)

	go func() {
		time.Sleep(time.Second * 2)
		oauthSrv.Shutdown(context.Background())
	}()
}

func hasCachedOauth() bool {
	if _, err := os.Stat(oauthTokenFilename); os.IsNotExist(err) {
		return false
	}
	return true
}

func setTokenSource(permissionCode string) {
	var err error
	ctx := context.Background()

	oauthToken, err = gcp.Conf.Exchange(ctx, permissionCode)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to retrieve the oauth2 token")
	}
	//fmt.Println(oauthToken)
	of, err := os.Create(oauthTokenFilename)
	if err != nil {
		log.Panic().Err(err).Msg("failed to retrieve the oauth2 token")
	}
	defer of.Close()
	if err = json.NewEncoder(of).Encode(oauthToken); err != nil {
		log.Panic().Err(err).Msg("Something went wrong when storing the token source")
	}
}

// type StoredSourceToken struct {
// 	SToken *oauth2.Token
// }

// func (t *StoredSourceToken) Token() (*oauth2.Token, error) {
// 	return t.SToken, nil
// }

func loadTokenSource() error {
	f, err := os.Open(oauthTokenFilename)
	if err != nil {
		return errors.Wrap(err, "failed to load the token source (deleted from disk)")
	}
	defer f.Close()
	var token oauth2.Token
	if err = json.NewDecoder(f).Decode(&token); err != nil {
		return err
	}
	oauthToken = &token
	// tokenSource = &StoredSourceToken{SToken: &token}
	return nil
}
