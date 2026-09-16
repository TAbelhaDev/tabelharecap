package main

import (
	"github.com/TAbelhaDev/tabelhatuiui"
)

// theme mirrors the installed DankMaterialShell's own configured accent
// (falling back to a manually chosen Catppuccin accent when DMS isn't
// installed/configured) — same lookup every ianptkcs TUI uses, kept in sync
// so every tool's chrome matches. TABELHARECAP_DMS_SETTINGS/TABELHARECAP_ACCENT
// env vars override the defaults; see tuiui.NewThemeFromEnv.
var theme = tuiui.NewThemeFromEnv("TABELHARECAP")

var (
	colBase     = theme.Base
	colMantle   = theme.Mantle
	colSurface0 = theme.Surface0
	colOverlay0 = theme.Overlay0
	colOverlay1 = theme.Overlay1
	colText     = theme.Text
	colSubtext0 = theme.Subtext0
	colPrimary  = theme.Primary
	colGreen    = theme.Green
	colYellow   = theme.Yellow
	colBlue     = theme.Blue
	colPink     = theme.Pink
	colLavender = theme.Lavender
)
