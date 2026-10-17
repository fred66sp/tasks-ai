// Package settings reads and writes the settings the product keeps in the
// project's own folder.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// FileName is the name of the settings file inside the product folder.
const FileName = "settings.json"

// The values a fresh project starts with.
const (
	DefaultPrefix   = "tk"
	DefaultDB       = "tasks.db"
	DefaultHTTPHost = "127.0.0.1"
	DefaultHTTPPort = 8890
	DefaultLogLevel = "info"
)

// prefixPattern is the shape a prefix may have: it is written in every reference.
var prefixPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,31}$`)

// logLevels are the levels the registry accepts.
var logLevels = []string{"debug", "info", "warn", "error"}

// Settings is the content of the settings file.
type Settings struct {
	Prefix string `json:"prefix"`
	DB     string `json:"db"`
	HTTP   HTTP   `json:"http"`
	Log    Log    `json:"log"`
}

// HTTP is where the interface is served.
type HTTP struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// Log is how much the product writes.
type Log struct {
	Level string `json:"level"`
}

// Default returns the settings a fresh project starts with.
func Default() Settings {
	return Settings{
		Prefix: DefaultPrefix,
		DB:     DefaultDB,
		HTTP:   HTTP{Host: DefaultHTTPHost, Port: DefaultHTTPPort},
		Log:    Log{Level: DefaultLogLevel},
	}
}

// DatabasePath is where the base of the project folder dir lives.
func DatabasePath(dir string, cfg Settings) string {
	return filepath.Join(dir, cfg.DB)
}

// Load reads the settings of the product folder dir. A project without the file
// keeps the default values of whatever the file does not set, so a fresh
// installation works untouched.
func Load(dir string) (Settings, error) {
	path := filepath.Join(dir, FileName)

	raw, err := os.ReadFile(path) //nolint:gosec // la ruta la trae el propio proyecto, no la compone el producto
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Default(), nil
		}

		return Settings{}, fmt.Errorf("reading %s: %w", path, err)
	}

	cfg := Default()
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Settings{}, fmt.Errorf("%s is not a valid settings file: %w", FileName, err)
	}

	if err := cfg.validate(); err != nil {
		return Settings{}, err
	}

	return cfg, nil
}

// Save writes the settings in dir, replacing what is there.
func Save(dir string, cfg Settings) error {
	if err := cfg.validate(); err != nil {
		return err
	}

	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("preparing %s: %w", FileName, err)
	}

	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil { //nolint:gosec // los ajustes se escriben con los permisos de lectura del proyecto
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

// Exists reports whether dir holds a settings file.
func Exists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, FileName))

	return err == nil
}

// validate refuses a settings file that would leave the product unusable, with
// a message that says what to fix.
func (cfg Settings) validate() error {
	if !prefixPattern.MatchString(cfg.Prefix) {
		return fmt.Errorf(
			"%s: the prefix %q must start with a letter or a digit and go on with letters, digits, hyphens or underscores, up to thirty-two characters",
			FileName,
			cfg.Prefix,
		)
	}

	if !filepath.IsLocal(cfg.DB) {
		return fmt.Errorf("%s: the database %q must be a file inside the product folder", FileName, cfg.DB)
	}

	if cfg.HTTP.Port < 1 || cfg.HTTP.Port > 65535 {
		return fmt.Errorf("%s: port %d is outside the port range", FileName, cfg.HTTP.Port)
	}

	if cfg.HTTP.Host == "" {
		return fmt.Errorf("%s: the interface address cannot be left empty", FileName)
	}

	if !slices.Contains(logLevels, cfg.Log.Level) {
		return fmt.Errorf("%s: the log level %q is not one of %s", FileName, cfg.Log.Level, strings.Join(logLevels, ", "))
	}

	return nil
}
