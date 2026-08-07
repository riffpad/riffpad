package daemon

import (
	"crypto/subtle"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// LocalTokenHeader carries the daemon's local API token on HTTP requests.
// Being a custom (non-simple) header, it also forces a CORS preflight on any
// browser cross-origin request, so malicious pages cannot even reach the API.
const LocalTokenHeader = "X-Riffpad-Token"

// localAuth guards every non-static route: the request must target a loopback
// Host, come from a loopback Origin (when an Origin header is present), and
// present the local API token either via LocalTokenHeader or — for browser
// WebSocket handshakes and Claude hook URLs, which cannot set headers — via
// the "token" query parameter. The token is generated on first run and
// persisted in config.json (0600), so only local processes able to read the
// user's config (the riffpad CLI, hook URLs written by `riffpad attach`, the
// pairing URL) can call the API.
func (s *Server) localAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !loopbackHost(r.Host) {
			writeError(w, http.StatusForbidden, "forbidden host")
			return
		}
		// Static webui assets are public; everything else needs the token.
		if r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/assets/") {
			next.ServeHTTP(w, r)
			return
		}
		if o := r.Header.Get("Origin"); o != "" && !loopbackOrigin(o) {
			writeError(w, http.StatusForbidden, "forbidden origin")
			return
		}
		if !s.validToken(r) {
			writeError(w, http.StatusUnauthorized, "missing or invalid local token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// validToken reports whether the request carries the local API token.
func (s *Server) validToken(r *http.Request) bool {
	tok := r.Header.Get(LocalTokenHeader)
	if tok == "" {
		tok = r.URL.Query().Get("token")
	}
	return tok != "" && subtle.ConstantTimeCompare([]byte(tok), []byte(s.token)) == 1
}

// loopbackHost reports whether a Host header value (with optional port)
// names the loopback interface. Anything else indicates DNS rebinding or a
// request forwarded from elsewhere.
func loopbackHost(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

// loopbackOrigin reports whether an Origin header value is a loopback URL.
// Any local port is accepted: a process that can serve pages on loopback can
// already read the token from config.json, so port-restricting buys nothing.
func loopbackOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return loopbackHost(u.Host)
}
