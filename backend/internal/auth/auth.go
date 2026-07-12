// Package auth implements "Sign in with Notion" (OAuth 2.0) and cookie
// sessions. The Notion access token obtained at login doubles as the API
// credential for the Notion sync feature (internal/notion).
//
// Auth is optional: when NOTION_CLIENT_ID/SECRET are unset the middleware is a
// passthrough and the app behaves exactly as before (local dev unchanged).
// Tokens are secrets, so they live in a separate auth.json file — never in the
// backup snapshot (which users download and move across devices).
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"moneytracker/backend/internal/storage"
)

const (
	sessionCookie = "mt_session"
	stateCookie   = "mt_oauth_state"
	sessionTTL    = 30 * 24 * time.Hour

	authorizeEndpoint = "https://api.notion.com/v1/oauth/authorize"
	tokenEndpoint     = "https://api.notion.com/v1/oauth/token"
)

// Config is read from the environment in cmd/server.
type Config struct {
	ClientID      string
	ClientSecret  string
	RedirectURI   string
	FrontendURL   string // where the callback redirects after login (default "/")
	SessionSecret string // HMAC key; random per boot when empty
	AllowedEmails []string
	CrossSite     bool // SameSite=None + Secure, for frontend and API on different domains
}

// Service holds auth config, the session signer, and the persisted accounts.
type Service struct {
	cfg           Config
	secret        []byte
	allowedEmails map[string]bool
	accounts      *Accounts
}

func New(cfg Config, accountsBlob storage.Blob) (*Service, error) {
	s := &Service{cfg: cfg, allowedEmails: map[string]bool{}}
	for _, e := range cfg.AllowedEmails {
		if e = strings.ToLower(strings.TrimSpace(e)); e != "" {
			s.allowedEmails[e] = true
		}
	}
	if cfg.SessionSecret != "" {
		s.secret = []byte(cfg.SessionSecret)
	} else {
		s.secret = make([]byte, 32)
		if _, err := rand.Read(s.secret); err != nil {
			return nil, err
		}
		if s.Enabled() {
			log.Println("auth: SESSION_SECRET not set; using a random key (sessions reset on restart)")
		}
	}
	accounts, err := loadAccounts(accountsBlob)
	if err != nil {
		return nil, fmt.Errorf("load accounts: %w", err)
	}
	s.accounts = accounts
	if s.Enabled() && len(s.allowedEmails) == 0 {
		log.Println("auth: ALLOWED_NOTION_EMAILS not set; any Notion user can log in")
	}
	return s, nil
}

// Enabled reports whether Notion login is configured. When false the app runs
// open (no login), matching the pre-auth behavior.
func (s *Service) Enabled() bool { return s.cfg.ClientID != "" && s.cfg.ClientSecret != "" }

func (s *Service) Accounts() *Accounts { return s.accounts }
func (s *Service) FrontendURL() string { return s.cfg.FrontendURL }
func (s *Service) emailAllowed(e string) bool {
	return len(s.allowedEmails) == 0 || s.allowedEmails[strings.ToLower(e)]
}

// ---- sessions (HMAC-signed token in an HttpOnly cookie) ----

func (s *Service) signSession(userID string, expires time.Time) string {
	payload := base64.RawURLEncoding.EncodeToString([]byte(userID)) + "." + strconv.FormatInt(expires.Unix(), 10)
	return payload + "." + s.sign(payload)
}

func (s *Service) verifySession(token string) (userID string, ok bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", false
	}
	payload := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(s.sign(payload)), []byte(parts[2])) {
		return "", false
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return "", false
	}
	uid, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}
	return string(uid), true
}

func (s *Service) sign(payload string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Service) setCookie(w http.ResponseWriter, name, value string, maxAge int) {
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	if s.cfg.CrossSite {
		c.SameSite = http.SameSiteNoneMode
		c.Secure = true
	}
	http.SetCookie(w, c)
}

// ---- middleware ----

type ctxKey struct{}

// UserID returns the authenticated Notion user id, or "" when auth is off.
func UserID(ctx context.Context) string {
	uid, _ := ctx.Value(ctxKey{}).(string)
	return uid
}

// Middleware rejects requests without a valid session when auth is enabled.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.Enabled() {
			next.ServeHTTP(w, r)
			return
		}
		c, err := r.Cookie(sessionCookie)
		if err == nil {
			if uid, ok := s.verifySession(c.Value); ok {
				if _, exists := s.accounts.Get(uid); exists {
					next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, uid)))
					return
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "authentication required"})
	})
}

// ---- OAuth flow ----

// BeginLogin redirects the browser to Notion's consent screen.
func (s *Service) BeginLogin(w http.ResponseWriter, r *http.Request) {
	if !s.Enabled() {
		http.Error(w, "Notion login is not configured", http.StatusNotFound)
		return
	}
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	state := hex.EncodeToString(buf)
	s.setCookie(w, stateCookie, state, 600)

	q := url.Values{
		"client_id":     {s.cfg.ClientID},
		"response_type": {"code"},
		"owner":         {"user"},
		"redirect_uri":  {s.cfg.RedirectURI},
		"state":         {state},
	}
	http.Redirect(w, r, authorizeEndpoint+"?"+q.Encode(), http.StatusFound)
}

// HandleCallback exchanges the code, enforces the email allowlist, stores the
// account (with its Notion access token) and issues the session cookie.
// Errors redirect back to the SPA with ?authError=... so the login page can
// show them.
func (s *Service) HandleCallback(w http.ResponseWriter, r *http.Request) {
	fail := func(msg string) {
		http.Redirect(w, r, s.cfg.FrontendURL+"?authError="+url.QueryEscape(msg), http.StatusFound)
	}
	if !s.Enabled() {
		http.Error(w, "Notion login is not configured", http.StatusNotFound)
		return
	}
	if e := r.URL.Query().Get("error"); e != "" {
		fail("Notion login was cancelled or denied")
		return
	}
	stateC, err := r.Cookie(stateCookie)
	if err != nil || stateC.Value == "" || stateC.Value != r.URL.Query().Get("state") {
		fail("login session expired, please try again")
		return
	}
	s.setCookie(w, stateCookie, "", -1)

	tok, err := s.exchangeCode(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		log.Printf("auth: token exchange failed: %v", err)
		fail("could not complete Notion login")
		return
	}
	email := tok.Owner.User.Person.Email
	if !s.emailAllowed(email) {
		fail("this Notion account is not allowed to access this app")
		return
	}

	acc := Account{
		UserID:        tok.Owner.User.ID,
		Name:          tok.Owner.User.Name,
		Email:         email,
		AvatarURL:     tok.Owner.User.AvatarURL,
		AccessToken:   tok.AccessToken,
		BotID:         tok.BotID,
		WorkspaceID:   tok.WorkspaceID,
		WorkspaceName: tok.WorkspaceName,
		ConnectedAt:   time.Now().UTC(),
	}
	if err := s.accounts.Put(acc); err != nil {
		log.Printf("auth: persist account: %v", err)
		fail("could not save login")
		return
	}

	expires := time.Now().Add(sessionTTL)
	s.setCookie(w, sessionCookie, s.signSession(acc.UserID, expires), int(sessionTTL.Seconds()))
	http.Redirect(w, r, s.cfg.FrontendURL, http.StatusFound)
}

// Logout clears the session cookie.
func (s *Service) Logout(w http.ResponseWriter, _ *http.Request) {
	s.setCookie(w, sessionCookie, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

// CurrentUser resolves the session cookie to an account, if any.
func (s *Service) CurrentUser(r *http.Request) (Account, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return Account{}, false
	}
	uid, ok := s.verifySession(c.Value)
	if !ok {
		return Account{}, false
	}
	return s.accounts.Get(uid)
}

// tokenResponse mirrors Notion's OAuth token endpoint response.
type tokenResponse struct {
	AccessToken   string `json:"access_token"`
	BotID         string `json:"bot_id"`
	WorkspaceID   string `json:"workspace_id"`
	WorkspaceName string `json:"workspace_name"`
	Owner         struct {
		User struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			AvatarURL string `json:"avatar_url"`
			Person    struct {
				Email string `json:"email"`
			} `json:"person"`
		} `json:"user"`
	} `json:"owner"`
}

func (s *Service) exchangeCode(ctx context.Context, code string) (*tokenResponse, error) {
	if code == "" {
		return nil, errors.New("missing code")
	}
	body, _ := json.Marshal(map[string]string{
		"grant_type":   "authorization_code",
		"code":         code,
		"redirect_uri": s.cfg.RedirectURI,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.cfg.ClientID, s.cfg.ClientSecret)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(res.Body).Decode(&e)
		return nil, fmt.Errorf("notion oauth: %s (%s)", res.Status, e.Error)
	}
	var tok tokenResponse
	if err := json.NewDecoder(res.Body).Decode(&tok); err != nil {
		return nil, err
	}
	if tok.AccessToken == "" || tok.Owner.User.ID == "" {
		return nil, errors.New("notion oauth: incomplete token response")
	}
	return &tok, nil
}
