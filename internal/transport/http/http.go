package http

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	nethttp "net/http"
	"strings"
	"time"

	"AgentOS/internal/agent"
	"AgentOS/pkg/schema"
)

type chatPayload struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

//go:embed static/index.html
var indexHTML []byte

// Run starts the HTTP adapter and shuts it down gracefully when the process
// context is cancelled.
func Run(ctx context.Context, addr string, service *agent.Service) error {
	mux := nethttp.NewServeMux()
	mux.HandleFunc("/", index)
	mux.HandleFunc("/healthz", healthz)
	mux.HandleFunc("/api/chat", chat(service))
	mux.HandleFunc("/api/chat/stream", chatStream(service))

	server := &nethttp.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("AgentOS web mode listening on http://127.0.0.1%s\n", addr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err == nethttp.ErrServerClosed {
			return nil
		}
		return err
	}
}

func healthz(w nethttp.ResponseWriter, _ *nethttp.Request) {
	writeJSON(w, nethttp.StatusOK, map[string]string{"status": "ok"})
}

func chat(service *agent.Service) nethttp.HandlerFunc {
	return func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodPost {
			nethttp.Error(w, "method not allowed", nethttp.StatusMethodNotAllowed)
			return
		}

		var payload chatPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			nethttp.Error(w, err.Error(), nethttp.StatusBadRequest)
			return
		}

		resp, err := service.Handle(r.Context(), schema.ChatRequest{
			SessionID: payload.SessionID,
			Messages: []schema.Message{{
				Role:    schema.RoleUser,
				Content: payload.Message,
			}},
		})
		if err != nil {
			writeJSON(w, nethttp.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, nethttp.StatusOK, map[string]any{
			"reply":    resp.Message.Content,
			"metadata": resp.Metadata,
		})
	}
}

func chatStream(service *agent.Service) nethttp.HandlerFunc {
	return func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodPost {
			nethttp.Error(w, "method not allowed", nethttp.StatusMethodNotAllowed)
			return
		}

		flusher, ok := w.(nethttp.Flusher)
		if !ok {
			nethttp.Error(w, "streaming not supported", nethttp.StatusInternalServerError)
			return
		}

		var payload chatPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			nethttp.Error(w, err.Error(), nethttp.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		resp, err := service.HandleStream(r.Context(), schema.ChatRequest{
			SessionID: payload.SessionID,
			Messages: []schema.Message{{
				Role:    schema.RoleUser,
				Content: payload.Message,
			}},
		}, func(delta string) error {
			if _, err := fmt.Fprintf(w, "data: %s\n\n", encodeSSEData(delta)); err != nil {
				return err
			}
			flusher.Flush()
			return nil
		})
		if err != nil {
			_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", encodeSSEData(err.Error()))
			flusher.Flush()
			return
		}

		meta, _ := json.Marshal(resp.Metadata)
		_, _ = fmt.Fprintf(w, "event: done\ndata: %s\n\n", encodeSSEData(string(meta)))
		flusher.Flush()
	}
}

func writeJSON(w nethttp.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func encodeSSEData(value string) string {
	var builder strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(value))
	first := true
	for scanner.Scan() {
		if !first {
			builder.WriteString("\ndata: ")
		}
		builder.WriteString(scanner.Text())
		first = false
	}
	if first {
		return ""
	}
	return builder.String()
}

func index(w nethttp.ResponseWriter, _ *nethttp.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(indexHTML)
}
