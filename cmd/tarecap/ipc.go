package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/TAbelhaDev/tabelhascaff/ipc"
)

// itemJSON is the wire format for the ipc subcommand.
type itemJSON struct {
	ID        int64   `json:"id"`
	Source    string  `json:"source"`
	Title     string  `json:"title"`
	Body      string  `json:"body,omitempty"`
	Link      string  `json:"link,omitempty"`
	CreatedAt string  `json:"created_at"`
	SeenAt    *string `json:"seen_at,omitempty"`
}

func itemToJSON(it Item) itemJSON {
	out := itemJSON{
		ID:        it.ID,
		Source:    it.Source,
		Title:     it.Title,
		Body:      it.Body,
		Link:      it.Link,
		CreatedAt: it.CreatedAt.Format(time.RFC3339),
	}
	if it.SeenAt != nil {
		s := it.SeenAt.Format(time.RFC3339)
		out.SeenAt = &s
	}
	return out
}

// runIPC implements `tarecap ipc <método> [key=value...] --json`, the same
// scriptable-data-source convention as dcal/tajobs/taradar/takanban.
func runIPC(args []string) int {
	parsed, err := ipc.ParseIPCArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "uso: tarecap ipc <método> [key=value...] --json")
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro abrindo store:", err)
		return 1
	}
	defer store.close()

	switch parsed.Method {
	case "item.add":
		return ipcItemAdd(store, parsed.Filters)
	case "item.add-batch":
		return ipcItemAddBatch(store, parsed.Filters)
	case "item.list":
		return ipcItemList(store, parsed.Filters)
	case "item.seen":
		return ipcItemSeen(store, parsed.Filters)
	case "item.seen-all":
		return ipcItemSeenAll(store)
	default:
		fmt.Fprintf(os.Stderr, "método desconhecido: %q\n", parsed.Method)
		return 1
	}
}

// ipcItemAdd registers a new novidade. Filters: source= (obrigatório),
// title= (obrigatório), body= (opcional), link= (opcional).
func ipcItemAdd(store *Store, filters map[string]string) int {
	source := filters["source"]
	title := filters["title"]
	if source == "" || title == "" {
		fmt.Fprintln(os.Stderr, "erro: filtros source= e title= são obrigatórios")
		return 1
	}
	item, err := store.add(source, title, filters["body"], filters["link"])
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	return ipc.WriteJSON(itemToJSON(item))
}

// ipcItemAddBatch registers multiple novidades at once. Filters: source=
// (obrigatório), items= (JSON array of {title, body, link}). Idempotent:
// (source, title) pairs that already exist are silently skipped.
func ipcItemAddBatch(store *Store, filters map[string]string) int {
	source := filters["source"]
	if source == "" {
		fmt.Fprintln(os.Stderr, "erro: filtro source= é obrigatório")
		return 1
	}
	raw := filters["items"]
	if raw == "" {
		fmt.Fprintln(os.Stderr, "erro: filtro items= é obrigatório")
		return 1
	}
	var items []batchItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		fmt.Fprintf(os.Stderr, "erro ao interpretar items=: %v\n", err)
		return 1
	}

	result, err := store.addBatch(source, items)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	out := make([]itemJSON, 0, len(result))
	for _, it := range result {
		out = append(out, itemToJSON(it))
	}
	return ipc.WriteJSON(out)
}

// ipcItemList lists items, newest first. Filters: unseen=true (opcional),
// source= (opcional), limit= (opcional, default 50).
func ipcItemList(store *Store, filters map[string]string) int {
	unseen := filters["unseen"] == "true"
	limit := 50
	if raw, ok := filters["limit"]; ok {
		n, err := strconv.Atoi(raw)
		if err != nil {
			fmt.Fprintln(os.Stderr, "erro: limit= inválido:", err)
			return 1
		}
		limit = n
	}
	items, err := store.list(unseen, filters["source"], limit)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	out := make([]itemJSON, 0, len(items))
	for _, it := range items {
		out = append(out, itemToJSON(it))
	}
	return ipc.WriteJSON(out)
}

// ipcItemSeen marks one item as seen. Filtro: id= (obrigatório).
func ipcItemSeen(store *Store, filters map[string]string) int {
	raw, ok := filters["id"]
	if !ok {
		fmt.Fprintln(os.Stderr, "erro: filtro id= é obrigatório")
		return 1
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro: id= inválido:", err)
		return 1
	}
	item, err := store.markSeen(id)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	return ipc.WriteJSON(itemToJSON(item))
}

// ipcItemSeenAll marks every unseen item as seen.
func ipcItemSeenAll(store *Store) int {
	n, err := store.markAllSeen()
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	return ipc.WriteJSON(map[string]int{"marked": n})
}
