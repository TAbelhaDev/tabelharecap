package main

import (
	"path/filepath"

	"github.com/TAbelhaDev/tabelhatuiui"
	"github.com/charmbracelet/bubbles/key"
)

// reg is tarecap's single source of truth for keybindings: defaults
// registered below, overrides persisted to ~/.config/tabelharecap/keybindings.json.
// Resolve() returns the effective binding, shared by dispatch, footer and
// help modal — a user rebind via the settings modal applies to all at once.
var reg = tuiui.NewKeyRegistry(filepath.Join(tuiui.ConfigDir(), "tabelharecap", "keybindings.json"))

func init() {
	reg.RegisterMany(
		tuiui.Action{ID: "quit", Help: "sair", Keys: []string{"q"}},
		tuiui.Action{ID: "help", Help: "atalhos", Keys: []string{"?"}},
		tuiui.Action{ID: "settings", Help: "reconfigurar teclas", Keys: []string{","}},
		tuiui.Action{ID: "refresh", Help: "recarregar", Keys: []string{"r"}},
		tuiui.Action{ID: "seen", Help: "marcar como visto", Keys: []string{"enter", "s"}, Label: "enter/s"},
		tuiui.Action{ID: "seen_all", Help: "marcar tudo como visto", Keys: []string{"A"}},
		tuiui.Action{ID: "toggle_unseen", Help: "só não vistos", Keys: []string{"u"}},
		tuiui.Action{ID: "cycle_source", Help: "filtrar por fonte", Keys: []string{"f"}},
		tuiui.Action{ID: "nav", Help: "mover cursor", Keys: []string{"j", "k", "up", "down"}, Label: "j/k"},
		tuiui.Action{ID: "focus", Help: "alternar foco", Keys: []string{"ctrl+h", "ctrl+l"}, Label: "ctrl+h/l"},
	)
}

// resolve is a short alias so Update reads like the old named keys.
func resolve(id string) key.Binding { return reg.Resolve(id) }
