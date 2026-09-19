package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

// newServer 组装 HTTP 路由：只对外提供计算与取回。
func newServer(store *Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/v1/demo", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, demoNetwork())
	})
	mux.HandleFunc("POST /api/v1/jobs", func(w http.ResponseWriter, r *http.Request) {
		var in NetworkInput
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		if err := dec.Decode(&in); err != nil {
			writeError(w, newError(ErrTypeBadRequest, "请求体不是合法的 JSON：%v", err))
			return
		}
		job, err := computeJob(&in)
		if err != nil {
			writeError(w, err)
			return
		}
		id, err := store.Save(job)
		if err != nil {
			writeError(w, newError(ErrTypeInternal, "保存作业失败：%v", err))
			return
		}
		w.Header().Set("Location", "/api/v1/jobs/"+id)
		writeJSON(w, http.StatusCreated, job)
	})
	mux.HandleFunc("GET /api/v1/jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		job, err := store.Get(r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, job)
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		log.Printf("写响应失败: %v", err)
	}
}

func writeError(w http.ResponseWriter, err error) {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		apiErr = newError(ErrTypeInternal, "%v", err)
	}
	writeJSON(w, statusOf(apiErr.Type), map[string]*APIError{"error": apiErr})
}
