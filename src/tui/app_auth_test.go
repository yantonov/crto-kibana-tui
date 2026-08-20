package tui

import (
	"context"
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yantonov/crtokt/src/config"
	"github.com/yantonov/crtokt/src/credentials"
	"github.com/yantonov/crtokt/src/models"
	"github.com/yantonov/crtokt/src/opensearch"
)

// ── stubs ────────────────────────────────────────────────────────────────────

type stubConfig struct{}

func (stubConfig) DataCenters(string) ([]string, error) { return []string{"dc1"}, nil }
func (stubConfig) KibanaURL(string, string) string      { return "https://kibana.example" }
func (stubConfig) Applications() []string               { return []string{"app"} }
func (stubConfig) Timeframes() []config.TimeframeOption {
	return []config.TimeframeOption{{Label: "1 hour", Value: "1h"}}
}
func (stubConfig) Environments() map[string]config.EnvironmentConfig {
	return map[string]config.EnvironmentConfig{"preprod": {DataCenters: []string{"dc1"}}}
}
func (stubConfig) IndexPattern() string        { return "kestrel-*" }
func (stubConfig) QueryTimeout() time.Duration { return time.Second }

type stubSearcher struct {
	loginErr error
	calls    []credentials.Credentials
}

func (s *stubSearcher) Login(_ context.Context, _, username, password string) error {
	s.calls = append(s.calls, credentials.Credentials{Username: username, Password: password})
	return s.loginErr
}
func (s *stubSearcher) IsAuthenticated() bool { return s.loginErr == nil }
func (s *stubSearcher) SearchAll(context.Context, config.Provider, models.Filter) models.CombinedResult {
	return models.CombinedResult{}
}

type stubStore struct {
	creds    credentials.Credentials
	loadErr  error
	saveErr  error
	saved    []credentials.Credentials
	clearHit int
}

func (s *stubStore) Load() (credentials.Credentials, error) { return s.creds, s.loadErr }
func (s *stubStore) Save(c credentials.Credentials) error {
	s.saved = append(s.saved, c)
	if s.saveErr != nil {
		return s.saveErr
	}
	s.creds = c
	return nil
}
func (s *stubStore) Clear() error {
	s.clearHit++
	s.creds = credentials.Credentials{}
	return nil
}

// run drains a tea.Cmd into the model, one message deep, which is all the login
// flow needs.
func run(t *testing.T, a App, cmd tea.Cmd) App {
	t.Helper()
	if cmd == nil {
		return a
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if m := c(); m != nil {
				a = update(t, a, m)
			}
		}
		return a
	}
	return update(t, a, msg)
}

func update(t *testing.T, a App, msg tea.Msg) App {
	t.Helper()
	next, _ := updateCmd(t, a, msg)
	return next
}

func updateCmd(t *testing.T, a App, msg tea.Msg) (App, tea.Cmd) {
	t.Helper()
	model, cmd := a.Update(msg)
	next, ok := model.(App)
	if !ok {
		t.Fatalf("Update returned %T, want App", model)
	}
	return next, cmd
}

// dispatch applies a message and then drains the command it produced, for the
// handlers that answer a message by scheduling more work.
func dispatch(t *testing.T, a App, msg tea.Msg) App {
	t.Helper()
	next, cmd := updateCmd(t, a, msg)
	return run(t, next, cmd)
}

func loginScreen(t *testing.T, a App) LoginScreen {
	t.Helper()
	login, ok := a.screen.(LoginScreen)
	if !ok {
		t.Fatalf("active screen is %T, want LoginScreen", a.screen)
	}
	return login
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestStoredCredentialsLogInWithoutTheForm(t *testing.T) {
	store := &stubStore{creds: credentials.Credentials{Username: "u", Password: "p"}}
	client := &stubSearcher{}

	a := New(stubConfig{}, client, store)
	if !a.loading {
		t.Fatal("a complete stored pair should start an automatic login")
	}
	if got := loginScreen(t, a).usernameInput.Value(); got != "u" {
		t.Fatalf("username prefill = %q, want %q", got, "u")
	}

	a = run(t, a, a.Init())

	if len(client.calls) != 1 || client.calls[0] != store.creds {
		t.Fatalf("login calls = %v, want one call with the stored pair", client.calls)
	}
	if _, ok := a.screen.(ResultsScreen); !ok {
		t.Fatalf("active screen is %T, want ResultsScreen", a.screen)
	}
	if len(store.saved) != 0 {
		t.Fatalf("unchanged credentials were rewritten: %v", store.saved)
	}
}

func TestMissingCredentialsShowTheForm(t *testing.T) {
	client := &stubSearcher{}
	// A username with no password is not enough to log in with.
	a := New(stubConfig{}, client, &stubStore{creds: credentials.Credentials{Username: "u"}})

	if a.loading {
		t.Fatal("a half-filled pair should not start a login")
	}
	loginScreen(t, a)

	a = run(t, a, a.Init())
	if len(client.calls) != 0 {
		t.Fatalf("login calls = %v, want none", client.calls)
	}
}

func TestUnreadableKeychainShowsTheFormWithTheError(t *testing.T) {
	store := &stubStore{
		creds:   credentials.Credentials{Username: "u", Password: "p"},
		loadErr: errors.New("keychain locked"),
	}

	a := New(stubConfig{}, &stubSearcher{}, store)

	if a.loading {
		t.Fatal("an unreadable keychain should not start a login")
	}
	if got := loginScreen(t, a).errMsg; got != "keychain: keychain locked" {
		t.Fatalf("errMsg = %q", got)
	}
}

func TestRejectedStoredCredentialsShowTheFormWithoutThePassword(t *testing.T) {
	store := &stubStore{creds: credentials.Credentials{Username: "u", Password: "stale"}}
	client := &stubSearcher{loginErr: opensearch.ErrInvalidCredentials}

	a := New(stubConfig{}, client, store)
	a = run(t, a, a.Init())

	login := loginScreen(t, a)
	if login.errMsg == "" {
		t.Fatal("want the auth failure reported on the form")
	}
	if got := login.passwordInput.Value(); got != "" {
		t.Fatalf("rejected password was put back: %q", got)
	}
	if got := login.usernameInput.Value(); got != "u" {
		t.Fatalf("username prefill = %q, want %q", got, "u")
	}
}

func TestUnreachableKibanaKeepsTheStoredPasswordForRetry(t *testing.T) {
	store := &stubStore{creds: credentials.Credentials{Username: "u", Password: "p"}}
	client := &stubSearcher{loginErr: errors.New("dial tcp: no route to host")}

	a := New(stubConfig{}, client, store)
	a = run(t, a, a.Init())

	login := loginScreen(t, a)
	if login.errMsg == "" {
		t.Fatal("want the transport failure reported on the form")
	}
	if got := login.passwordInput.Value(); got != "p" {
		t.Fatalf("password = %q, want it restored for a one-keypress retry", got)
	}
}

func TestSubmittedCredentialsReachTheKeychain(t *testing.T) {
	store := &stubStore{}
	client := &stubSearcher{}

	a := New(stubConfig{}, client, store)
	a = update(t, a, LoginSubmitMsg{Username: "new", Password: "secret"})
	a = run(t, a, a.doLogin(a.pending))

	want := credentials.Credentials{Username: "new", Password: "secret"}
	if len(store.saved) != 1 || store.saved[0] != want {
		t.Fatalf("saved = %v, want one write of %v", store.saved, want)
	}
	if _, ok := a.screen.(ResultsScreen); !ok {
		t.Fatalf("active screen is %T, want ResultsScreen", a.screen)
	}
}

func TestFailedKeychainWriteStillLetsTheUserIn(t *testing.T) {
	store := &stubStore{saveErr: errors.New("access denied")}
	a := New(stubConfig{}, &stubSearcher{}, store)
	a = update(t, a, LoginSubmitMsg{Username: "new", Password: "secret"})
	a = run(t, a, a.doLogin(a.pending))

	results, ok := a.screen.(ResultsScreen)
	if !ok {
		t.Fatalf("active screen is %T, want ResultsScreen", a.screen)
	}
	if results.notice != "keychain: access denied" {
		t.Fatalf("notice = %q", results.notice)
	}
}

func TestForgettingCredentialsWipesTheKeychainAndTheForm(t *testing.T) {
	store := &stubStore{creds: credentials.Credentials{Username: "u", Password: "p"}}
	a := New(stubConfig{}, &stubSearcher{loginErr: opensearch.ErrInvalidCredentials}, store)
	a = run(t, a, a.Init())

	a = dispatch(t, a, ClearStoredCredentialsMsg{})

	if store.clearHit != 1 {
		t.Fatalf("store.Clear called %d times, want 1", store.clearHit)
	}
	login := loginScreen(t, a)
	if login.usernameInput.Value() != "" || login.passwordInput.Value() != "" {
		t.Fatalf("form still holds %q/%q", login.usernameInput.Value(), login.passwordInput.Value())
	}
	if login.noteMsg == "" {
		t.Fatal("want the removal confirmed on the form")
	}
	if a.stored.Complete() || a.pending.Complete() {
		t.Fatal("app still holds the cleared credentials")
	}
}

func TestQIsTypedIntoTheLoginFormRatherThanQuitting(t *testing.T) {
	t.Setenv("USER", "") // keep the focus on the username field
	a := New(stubConfig{}, &stubSearcher{}, &stubStore{})

	next, cmd := updateCmd(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if isQuit(cmd) {
		t.Fatal("q should not quit while the login form is up")
	}
	if got := loginScreen(t, next).usernameInput.Value(); got != "q" {
		t.Fatalf("username field holds %q, want %q", got, "q")
	}
}
