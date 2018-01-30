package botServer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/sessions"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

var permissions = struct {
	readMessages int
	sendMessages int
	connect      int
	speak        int
}{
	readMessages: 1024,
	sendMessages: 2048,
	connect:      1048576,
	speak:        2097152,
}

var ()

// BotServer .
type BotServer struct {
	log  *zap.Logger
	port string

	store         *sessions.CookieStore
	oauthConf     *oauth2.Config
	htmlIndexPage string
	apiBaseURL    string
}

// NewBotServer returns everything to auth a bot
func NewBotServer(id, secret, url, port string, log *zap.Logger) *BotServer {

	data, err := ioutil.ReadFile("templates/index.html")
	if err != nil {
		log.Error("failed to open template", zap.Error(err))
		return &BotServer{}
	}
	htmlIndexPage := string(data)
	apiBaseURL := "https://discordapp.com/api"

	// Create a cookie store
	store := sessions.NewCookieStore([]byte(secret))

	// Setup the OAuth2 Configuration
	endpoint := oauth2.Endpoint{
		AuthURL:  apiBaseURL + "/oauth2/authorize",
		TokenURL: apiBaseURL + "/oauth2/token",
	}

	oauthConf := &oauth2.Config{
		ClientID:     id,
		ClientSecret: secret,
		Scopes:       []string{"bot", "identify"},
		Endpoint:     endpoint,
		RedirectURL:  fmt.Sprintf("%s/callback", url),
	}

	b := &BotServer{
		log:           log,
		port:          port,
		store:         store,
		oauthConf:     oauthConf,
		htmlIndexPage: htmlIndexPage,
		apiBaseURL:    apiBaseURL,
	}
	return b
}

// Returns the current session or aborts the request
func (b *BotServer) getSessionOrAbort(w http.ResponseWriter, r *http.Request) *sessions.Session {
	session, err := b.store.Get(r, "session")

	if session == nil {
		b.log.Error("unable to create session", zap.Error(err))
		http.Error(w, "invalid or corrupted session", http.StatusInternalServerError)
		return nil
	}

	return session
}

// Redirects to the oauth2
func (b *BotServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	session := b.getSessionOrAbort(w, r)
	if session == nil {
		return
	}

	// Create a random state
	session.Values["state"] = randSeq(32)
	session.Save(r, w)

	// OR the permissions we want
	perms := permissions.readMessages | permissions.sendMessages | permissions.connect | permissions.speak

	// Return a redirect to the oauth provider
	url := b.oauthConf.AuthCodeURL(session.Values["state"].(string), oauth2.AccessTypeOnline)
	http.Redirect(w, r, url+fmt.Sprintf("&permissions=%v", perms), http.StatusTemporaryRedirect)
}

func (b *BotServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	session := b.getSessionOrAbort(w, r)
	if session == nil {
		return
	}

	// Check the state string is correct
	state := r.FormValue("state")
	if state != session.Values["state"] {
		b.log.Error("invalid oauth session state", zap.String("state", state))
		http.Redirect(w, r, "/?key_to_success=0", http.StatusTemporaryRedirect)
		return
	}

	errorMsg := r.FormValue("error")
	if errorMsg != "" {
		b.log.Error("received oauth error from provider", zap.String("error", errorMsg))
		http.Redirect(w, r, "/?key_to_success=0", http.StatusTemporaryRedirect)
		return
	}

	token, err := b.oauthConf.Exchange(oauth2.NoContext, r.FormValue("code"))
	if err != nil {
		b.log.Error("failed to exchange token with provider", zap.String("error", errorMsg))
		http.Redirect(w, r, "/?key_to_success=0", http.StatusTemporaryRedirect)
		return
	}

	body, _ := json.Marshal(map[interface{}]interface{}{})
	req, err := http.NewRequest("GET", b.apiBaseURL+"/users/@me", bytes.NewBuffer(body))
	if err != nil {
		b.log.Error("failed to create @me request", zap.Error(err))
		http.Error(w, "failed to retrieve user profile", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Authorization", token.Type()+" "+token.AccessToken)
	client := &http.Client{Timeout: (20 * time.Second)}
	resp, err := client.Do(req)
	if err != nil {
		b.log.Error("failed to request @me data", zap.Error(err))
		http.Error(w, "Failed to retrieve user profile", http.StatusInternalServerError)
		return
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		b.log.Error("failed to read data from http response", zap.Error(err))
		http.Error(w, "failed to retrieve user profile", http.StatusInternalServerError)
		return
	}

	user := discordgo.User{}
	err = json.Unmarshal(respBody, &user)
	if err != nil {
		b.log.Error("failed to parse JSON payload from HTTP response", zap.Error(err))
		http.Error(w, "failed to retrieve user profile", http.StatusInternalServerError)
		return
	}

	// Finally write some information to the session store
	session.Values["token"] = token.AccessToken
	session.Values["username"] = user.Username
	session.Values["tag"] = user.Discriminator
	delete(session.Values, "state")
	session.Save(r, w)

	// And redirect the user back to the dashboard
	http.Redirect(w, r, "/?key_to_success=1", http.StatusTemporaryRedirect)
}

func (b *BotServer) handleMe(w http.ResponseWriter, r *http.Request) {
	session, _ := b.store.Get(r, "session")

	body, err := json.Marshal(map[string]interface{}{
		"username": session.Values["username"],
		"tag":      session.Values["tag"],
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func (b *BotServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(b.htmlIndexPage))
}

// Run auth server
func (b *BotServer) Run() {
	server := http.NewServeMux()
	server.HandleFunc("/", b.handleIndex)
	server.HandleFunc("/me", b.handleMe)
	server.HandleFunc("/login", b.handleLogin)
	server.HandleFunc("/callback", b.handleCallback)

	b.log.Info("starting server", zap.String("port", b.port))

	err := http.ListenAndServe(fmt.Sprintf(":%s", b.port), server)
	if err != nil {
		b.log.Error("failed serving", zap.Error(err))
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// Return a random character sequence of n length
func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
