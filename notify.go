package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// alertState persists the last alerted item ID across runs.
type alertState struct {
	LastAlertedID int64 `json:"last_alerted_id"`
}

// alertStatePath returns the path to the alert state file, derived from
// the database directory (same directory as tarecap.db).
func alertStatePath() string {
	return filepath.Join(filepath.Dir(defaultDBPath()), "alert.state")
}

func loadAlertState() alertState {
	var st alertState
	data, err := os.ReadFile(alertStatePath())
	if err != nil {
		return st
	}
	_ = json.Unmarshal(data, &st)
	return st
}

func saveAlertState(st alertState) error {
	dir := filepath.Dir(alertStatePath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return os.WriteFile(alertStatePath(), data, 0o644)
}

// runNotify checks for new items since the last alert and fires a desktop
// notification if there are any. Intended to be called by a systemd timer
// every ~15 minutes.
func runNotify() int {
	_ = refreshSettings()

	store, err := openStore()
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro abrindo store:", err)
		return 1
	}
	defer store.close()

	st := loadAlertState()

	// Baseline: first run — seed the state without alerting.
	if st.LastAlertedID == 0 {
		maxID, err := store.maxID()
		if err != nil {
			fmt.Fprintln(os.Stderr, "erro obtendo max ID:", err)
			return 1
		}
		st.LastAlertedID = maxID
		if err := saveAlertState(st); err != nil {
			fmt.Fprintln(os.Stderr, "erro salvando estado:", err)
			return 1
		}
		fmt.Fprintf(os.Stderr, "tarecap notify: baseline seed, last_alerted_id=%d\n", maxID)
		return 0
	}

	items, err := store.listSince(st.LastAlertedID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro listando itens novos:", err)
		return 1
	}

	if len(items) == 0 {
		return 0
	}

	// Build notification message.
	sources := make(map[string]bool)
	titles := make([]string, 0, min(len(items), 2))
	for _, it := range items {
		sources[it.Source] = true
		if len(titles) < 2 {
			titles = append(titles, it.Title)
		}
	}

	srcList := make([]string, 0, len(sources))
	for s := range sources {
		srcList = append(srcList, s)
	}

	msg := fmt.Sprintf("%d novidade(s) em %d fonte(s): %s",
		len(items), len(srcList), strings.Join(srcList, ", "))
	if len(titles) > 0 {
		msg += " — " + strings.Join(titles, " | ")
	}

	// Fire persistent desktop notification (critical urgency = stays until
	// manual dismiss; --expire-time=0 = never auto-dismiss).
	cmd := exec.Command("notify-send",
		"-a", "tarecap",
		"-u", "critical",
		"--expire-time=0",
		"tarecap", msg)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tarecap notify: notify-send falhou: %v\n", err)
		// Don't advance state — next poll will retry.
		return 1
	}

	// Advance state on success.
	st.LastAlertedID = items[len(items)-1].ID
	if err := saveAlertState(st); err != nil {
		fmt.Fprintln(os.Stderr, "erro salvando estado:", err)
		return 1
	}

	fmt.Fprintf(os.Stderr, "tarecap notify: %d novidade(s) alertada(s)\n", len(items))
	return 0
}
