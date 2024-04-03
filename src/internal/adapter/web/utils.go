package web

import (
	"encoding/json"
	"fmt"
	"goyav/internal/core/domain"
	"log/slog"
	"net/http"
)

type ObjectMessage struct {
	Message     string              `json:"message"`
	ID          string              `json:"id,omitempty"`
	Version     string              `json:"version,omitempty"`
	Information string              `json:"information,omitempty"`
	Document    *domain.DocumentDTO `json:"document,omitempty"`
}

// methodNotAllowed sends a method not allowed response.
func methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	msg := fmt.Sprintf("Method %v is not allowed on %v.", r.Method, r.URL.Path)
	writeError(w, http.StatusMethodNotAllowed, msg, &ObjectMessage{})
}

// writeJson marshals and writes a JSON response
func writeJson(w http.ResponseWriter, code int, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "inernal server error", http.StatusInternalServerError)
		slog.Error("handler.printjson", "error", err.Error())
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(b)
}

// writeError shortcut for writing error responses in JSON format. Uses writeJson.
func writeError(w http.ResponseWriter, code int, msg string, obj *ObjectMessage) {
	if obj == nil {
		obj = &ObjectMessage{}
	}
	obj.Message = msg
	writeJson(w, code, obj)
}
