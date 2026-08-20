// Package credentials keeps the Kibana username/password in the OS keychain.
package credentials

import (
	"os"

	"github.com/zalando/go-keyring"
)

const (
	defaultService = "kibana-tui"
	usernameKey    = "username"
	passwordKey    = "password"
)

// ServiceEnvVar overrides the keychain service the credentials live under. Point
// it at another tool's service to reuse a credential pair already stored there
// instead of keeping a second copy.
const ServiceEnvVar = "CRTOKT_KEYCHAIN_SERVICE"

// Service returns the keychain service in use.
func Service() string {
	if s := os.Getenv(ServiceEnvVar); s != "" {
		return s
	}
	return defaultService
}

// Credentials is a username/password pair.
type Credentials struct {
	Username string
	Password string
}

// Complete reports whether both halves are present, i.e. whether a login can
// be attempted without asking the user for anything.
func (c Credentials) Complete() bool {
	return c.Username != "" && c.Password != ""
}

// Store reads and writes the credential pair. The interface exists so the TUI
// can be driven by a stub instead of the real keychain.
type Store interface {
	Load() (Credentials, error)
	Save(Credentials) error
	Clear() error
}

// Keychain is the OS-keychain backed Store.
type Keychain struct{}

// Compile-time assertion that Keychain satisfies Store.
var _ Store = Keychain{}

// Load returns an empty value (with no error) for entries that are not set,
// so a first run is indistinguishable from a cleared keychain.
func (Keychain) Load() (Credentials, error) {
	username, err := get(usernameKey)
	if err != nil {
		return Credentials{}, err
	}
	password, err := get(passwordKey)
	if err != nil {
		return Credentials{}, err
	}
	return Credentials{Username: username, Password: password}, nil
}

// Save overwrites both entries.
func (Keychain) Save(c Credentials) error {
	service := Service()
	if err := keyring.Set(service, usernameKey, c.Username); err != nil {
		return err
	}
	return keyring.Set(service, passwordKey, c.Password)
}

// Clear removes both entries, and is a no-op for entries already absent.
func (Keychain) Clear() error {
	if err := del(usernameKey); err != nil {
		return err
	}
	return del(passwordKey)
}

func get(key string) (string, error) {
	value, err := keyring.Get(Service(), key)
	if err == keyring.ErrNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func del(key string) error {
	err := keyring.Delete(Service(), key)
	if err != nil && err != keyring.ErrNotFound {
		return err
	}
	return nil
}
