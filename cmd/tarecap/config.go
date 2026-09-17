package main

import (
	"path/filepath"

	"github.com/TAbelhaDev/tabelhatuiui"
)

// config is tarecap's settings schema, read from
// ~/.config/tabelharecap/config.toml. Every field falls back to
// defaultConfig when the file leaves it out.
type config struct {
	Database databaseConfig `toml:"database"`
}

type databaseConfig struct {
	// Path is the SQLite file every `tarecap ipc` caller and the TUI itself
	// read/write. Many independent short-lived processes write here (cron
	// jobs, workflow steps, one-off scripts), which is why store.go opens it
	// in WAL mode rather than the single-writer default other TAbelhaDev
	// tools use.
	Path string `toml:"path"`
}

func defaultConfig() config {
	return config{
		Database: databaseConfig{
			Path: filepath.Join("~", ".local", "state", "tabelharecap", "tarecap.db"),
		},
	}
}

// configPath is resolved lazily, not in a package-level var: an init-time var
// would freeze TABELHARECAP_CONFIG/XDG_CONFIG_HOME before main (or a test)
// could set them.
func configPath() string {
	return tuiui.EnvOr("TABELHARECAP_CONFIG", tuiui.ConfigPath("tabelharecap", "config.toml"))
}

// settings is the normalized snapshot the app reads from.
var settings = defaultConfig()

// normalize expands "~" in path-shaped fields and restores any default a
// blank config file value would otherwise leave empty.
func normalize(c config) config {
	d := defaultConfig()
	if c.Database.Path == "" {
		c.Database.Path = d.Database.Path
	}
	c.Database.Path = tuiui.ExpandHome(c.Database.Path)
	return c
}

// refreshSettings re-reads config.toml from disk and returns a warning string
// (never an error) — a bad config file must not stop the app from running.
func refreshSettings() string {
	path := configPath()
	cfg := tuiui.NewConfig(path, defaultConfig())
	err := cfg.Load()
	settings = normalize(cfg.Get())
	if err != nil {
		return "erro lendo " + path + ": " + err.Error()
	}
	return ""
}

// defaultDBPath is the effective DB path: TARECAP_DB wins over config.toml,
// which wins over the compiled-in default.
func defaultDBPath() string {
	return tuiui.ExpandHome(tuiui.EnvOr("TARECAP_DB", settings.Database.Path))
}
