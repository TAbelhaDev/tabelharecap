<div align="center">

# TAbelhaRecap

**A passive inbox for "what's new"** — other tools drop a note in via IPC,
you just check what showed up.

**English** · [Português](README.pt-BR.md)

[![Go Version](https://img.shields.io/github/go-mod/go-version/TAbelhaDev/tabelharecap?style=flat-square&logo=go&logoColor=white&color=00ADD8)](go.mod)
[![Built with Bubble Tea](https://img.shields.io/badge/built%20with-Bubble%20Tea-ff69b4?style=flat-square)](https://github.com/charmbracelet/bubbletea)
[![Powered by tabelhatuiui](https://img.shields.io/badge/theme-tabelhatuiui-d6b4f7?style=flat-square)](https://github.com/TAbelhaDev/tabelhatuiui)
[![License: AGPL-3.0](https://img.shields.io/badge/license-AGPL--3.0-blue?style=flat-square)](LICENSE)

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/ianptkcs)

</div>

---

## What it is

With enough automated workflows, recurring jobs and projects running in
parallel, it's easy to lose track of what actually happened. `tarecap` is
not another dashboard that polls all of those tools for you — it's the
opposite: a small, source-agnostic inbox that anything can write to.

Any tool, script, cron job, or workflow step can register a "novidade"
(an item worth knowing about) by shelling out to `tarecap ipc item.add`.
`tarecap` itself never reaches out to fetch anything — it only stores what
it's told and shows it back to you in a single chronological feed, so
opening it is a quick "what's new since I last checked", not a scavenger
hunt across five other TUIs.

The theme and shared chrome (header/footer/panels, keybinding registry, the
`ipc ... --json` helpers) come from
[`tabelhatuiui`](https://github.com/TAbelhaDev/tabelhatuiui) /
[`tabelhascaff`](https://github.com/TAbelhaDev/tabelhascaff), shared across
my Bubble Tea TUIs.

## Contents

- [Installation](#installation)
- [Usage](#usage)
- [IPC](#ipc)
- [Configuration](#configuration)
- [License](#license)

## Installation

Requires Go 1.26+.

```bash
go install github.com/TAbelhaDev/tabelharecap@latest
```

That installs the binary as `tabelharecap` (matching the module name). To get
the short `tarecap` name used throughout this README, build from source
instead:

```bash
git clone https://github.com/TAbelhaDev/tabelharecap.git
cd tabelharecap
go build -o tarecap .
```

## Usage

Running `tarecap` with no arguments opens the TUI: a three-panel layout
(wide terminals) with a chronological feed on the left, metadata on the
top-right, and a scrollable markdown description on the bottom-right. A
`●`/`○` marker indicates unseen/seen items. On narrow terminals it falls
back to the single-panel table. Keys:

| Key | Action |
| --- | --- |
| `ctrl+h` / `ctrl+l` | switch focus between the list and description panels |
| `enter` / `s` | mark the selected item as seen |
| `A` | mark every item as seen |
| `u` | toggle "unseen only" |
| `f` | cycle through the sources currently present |
| `r` | reload from the database (picks up anything another process just added) |
| `?` | help |
| `,` | rebind keys |
| `q` | quit |

It's read-only beyond that: creating and editing items is what the IPC
write endpoint below is for, not the TUI.

## IPC

Any external caller — a bash job script, a workflow step, a one-off command
— registers a novidade the same way every other `ianptkcs` TUI exposes a
scriptable data source:

```bash
tarecap ipc item.add source=post-suggestions title="3 sugestões novas revisadas" \
  body="opcional, texto mais longo" link="opcional, comando ou path pra abrir a origem" --json

tarecap ipc item.list --json                # every item, newest first
tarecap ipc item.list unseen=true --json    # only unseen items
tarecap ipc item.list source=tajobs --json  # only items from one source

tarecap ipc item.seen id=42 --json          # mark one item as seen (idempotent)
tarecap ipc item.seen-all --json            # mark every unseen item as seen
```

`source` and `title` are the only required fields on `item.add`. `link` is
stored as an opaque string — `tarecap` never interprets or executes it, it
only displays it back; what it means (a shell command, a file path, a URL)
is up to whatever reads it later.

Wiring any of `taglue`/`tajobs`/`taradar`/`tabelhakanban` (or ad-hoc
markdown notes/notifications) to actually call `item.add` is a deliberate
follow-up, not something this tool does on its own — `tarecap` has no
knowledge of, and never talks to, any other tool's IPC.

## Configuration

`~/.config/tabelharecap/config.toml`:

```toml
[database]
path = "~/.local/state/tabelharecap/tarecap.db"
```

`TARECAP_DB` overrides the database path outright (config file included) —
useful for pointing a test run at a scratch file. The database is SQLite in
WAL mode, since it's written by many independent short-lived processes
(cron jobs, workflow steps) rather than a single long-lived one.

Keybindings live in `~/.config/tabelharecap/keybindings.json`, edited
through the in-app settings modal (`,`) rather than by hand.

## License

[GNU AGPL-3.0](LICENSE) — free and open source. If you run a modified version of
this project, including as a network service, you also have to make the modified
source available under the same license.
