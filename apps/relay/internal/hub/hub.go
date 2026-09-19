// Package hub implements the relay: user accounts, host/device registration,
// session routing and encrypted-envelope forwarding between hosts and viewers.
package hub

import (
	"log"
	"net/http"
	"os"
	"sync"
)

const version = "0.2.0"

type Hub struct {
	log     *log.Logger
	dataDir string
	store   *Store

	mu                 sync.Mutex
	hosts              map[string]*hostConn
	sessions           map[string]SessionMeta
	sessionHosts       map[string]string
	viewers            map[string]*viewerConn
	rateLimits         map[string]ipCounter
	oauthStates        map[string]oauthState
	deviceLogins       map[string]*deviceLogin
	githubID           string
	githubSecret       string
	githubAuthorizeURL string
	githubRedirectURL  string
	githubTokenURL     string
	githubUserURL      string
	appURL             string
	apiHosts           []string
	unsubSecret        string
	adminKey           string
	webOrigins         []string
}

func New(logger *log.Logger, dataDir, databaseURL string) (*Hub, error) {
	store, err := OpenStore(dataDir, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Hub{
		log:          logger,
		dataDir:      dataDir,
		store:        store,
		hosts:        map[string]*hostConn{},
		sessions:     map[string]SessionMeta{},
		sessionHosts: map[string]string{},
		viewers:      map[string]*viewerConn{},
		rateLimits:   map[string]ipCounter{},
		oauthStates:  map[string]oauthState{},
		deviceLogins: map[string]*deviceLogin{},
		githubID:     os.Getenv("GITHUB_CLIENT_ID"),
		githubSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		// Overridable so the whole OAuth flow can be driven against a stub:
		// the core-path browser test (#314) needs the popup, the token
		// exchange and the user lookup to stay on the test machine.
		githubAuthorizeURL: envOr("GITHUB_AUTHORIZE_URL", "https://github.com/login/oauth/authorize"),
		// The callback GitHub redirects to is this relay's own public URL, so a
		// self-hosted relay must be able to state it (and register it with the
		// GitHub app); the tests point it at the local relay.
		githubRedirectURL: envOr("GITHUB_REDIRECT_URL", "https://api.riffpad.ai/api/auth/github/callback"),
		githubTokenURL:    envOr("GITHUB_TOKEN_URL", "https://github.com/login/oauth/access_token"),
		githubUserURL:     envOr("GITHUB_USER_URL", "https://api.github.com/user"),
		appURL:            envOr("RIFFPAD_APP_URL", "https://app.riffpad.ai"),
		apiHosts:          splitCSV(os.Getenv("RIFFPAD_API_HOSTS")),
		unsubSecret:       os.Getenv("UNSUBSCRIBE_SECRET"),
		adminKey:          os.Getenv("WAITLIST_ADMIN_KEY"),
		webOrigins:        splitCSV(envOr("RIFFPAD_WEB_ORIGINS", "https://riffpad.ai,https://www.riffpad.ai")),
	}, nil
}

func (h *Hub) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.handleRoot)
	mux.HandleFunc("/assets/", h.handleAsset)
	mux.HandleFunc("/api/status", h.handleStatus)
	mux.HandleFunc("/api/auth/register", h.handleRegister)
	mux.HandleFunc("/api/auth/login", h.handleLogin)
	mux.HandleFunc("/api/auth/logout", h.handleLogout)
	mux.HandleFunc("/api/auth/me", h.handleMe)
	mux.HandleFunc("/api/auth/github/login", h.handleGitHubLogin)
	mux.HandleFunc("/api/auth/github/callback", h.handleGitHubCallback)
	mux.HandleFunc("/api/auth/oauth/device", h.handleDeviceLogin)
	mux.HandleFunc("/api/auth/oauth/device/poll", h.handleDeviceLoginPoll)
	mux.HandleFunc("/api/auth/oauth/device/status", h.handleDeviceLoginStatus)
	mux.HandleFunc("/api/hosts", h.handleHosts)
	mux.HandleFunc("/api/hosts/register", h.handleRegisterHost)
	mux.HandleFunc("/api/pairings", h.handleCreatePairing)
	mux.HandleFunc("/api/pair", h.handlePair)
	mux.HandleFunc("/api/devices", h.handleDevices)
	mux.HandleFunc("/api/devices/", h.handleDeviceDelete)
	mux.HandleFunc("/api/hosts/", h.handleHostKillswitch)
	mux.HandleFunc("/api/sessions", h.handleSessions)
	mux.HandleFunc("/api/sessions/", h.handleSessionMeta)
	mux.HandleFunc("/api/waitlist/subscribe", h.handleWaitlistSubscribe)
	mux.HandleFunc("/api/waitlist/emails", h.handleWaitlistEmails)
	mux.HandleFunc("/api/waitlist/unsubscribe", h.handleWaitlistUnsubscribe)
	mux.HandleFunc("/api/waitlist/optouts", h.handleWaitlistOptouts)
	mux.HandleFunc("/ws/host", h.handleHostWS)
	mux.HandleFunc("/ws", h.handleViewerWS)
	return mux
}
