package main

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

//go:embed web
var webFS embed.FS

func main() {
	addr := envOr("ADDR", ":8080")
	dataFile := envOr("DATA_FILE", "data/tasks.json")

	store, err := NewStore(dataFile)
	if err != nil {
		log.Fatalf("erro ao abrir %s: %v", dataFile, err)
	}

	static, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServer(http.FS(static)))

	mux.HandleFunc("GET /api/tasks", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, store.List())
	})

	mux.HandleFunc("POST /api/tasks", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Title    string `json:"title"`
			Priority int    `json:"priority"`
		}
		if !readJSON(w, r, &in) {
			return
		}
		t, err := store.Create(in.Title, in.Priority)
		respond(w, http.StatusCreated, t, err)
	})

	mux.HandleFunc("PATCH /api/tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		var in Patch
		if !readJSON(w, r, &in) {
			return
		}
		t, err := store.Update(id, in)
		respond(w, http.StatusOK, t, err)
	})

	mux.HandleFunc("DELETE /api/tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		if err := store.Delete(id); err != nil {
			respond(w, 0, nil, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	log.Printf("ouvindo em %s (dados em %s)", addr, dataFile)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return 0, false
	}
	return id, true
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return false
	}
	return true
}

func respond(w http.ResponseWriter, status int, v any, err error) {
	switch {
	case err == nil:
		writeJSON(w, status, v)
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrInvalid):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("erro interno: %v", err)
		writeError(w, http.StatusInternalServerError, "erro ao salvar")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
