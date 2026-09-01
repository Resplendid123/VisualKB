package sandbox

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"learn/internal/domain"
)

// Mimics controller ndjson exec protocol.
type fakeExecHandler struct {
	expectedNamespace string
	expectedName      string
	expectedCommand   string
	stdout            []string
	stderr            []string
	exitCode          int
	t                 *testing.T
}

func (h *fakeExecHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Path layout: /v1/sandboxes/{ns}/{name}/exec.
	const prefix = "/v1/sandboxes/"
	if !strings.HasPrefix(r.URL.Path, prefix) || !strings.HasSuffix(r.URL.Path, "/exec") {
		http.NotFound(w, r)
		return
	}
	tail := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, prefix), "/exec")
	parts := strings.SplitN(tail, "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}
	ns, name := parts[0], parts[1]
	if ns != h.expectedNamespace || name != h.expectedName {
		http.Error(w, fmt.Sprintf("unexpected path: ns=%s name=%s", ns, name), http.StatusBadRequest)
		return
	}

	var req struct {
		Command string        `json:"command"`
		Timeout time.Duration `json:"timeout"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Command != h.expectedCommand {
		http.Error(w, fmt.Sprintf("unexpected command: %q", req.Command), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)

	for _, line := range h.stdout {
		writeJSON(w, map[string]any{"stream": "stdout", "data": line})
	}
	for _, line := range h.stderr {
		writeJSON(w, map[string]any{"stream": "stderr", "data": line})
	}
	if h.exitCode == 124 {
		writeJSON(w, map[string]any{"stream": "error", "error": "timeout"})
	}
	writeJSON(w, map[string]any{"stream": "exit", "code": h.exitCode})
	if flusher != nil {
		flusher.Flush()
	}
}

func writeJSON(w io.Writer, v any) {
	_ = json.NewEncoder(w).Encode(v)
}

// Point executor at httptest server.
func newTestExecutor(t *testing.T, h http.Handler) *ControllerExecutor {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &ControllerExecutor{
		execEndpoint: srv.URL,
		httpClient:   &http.Client{},
	}
}

func TestControllerExecutor_Exec_HappyPath(t *testing.T) {
	h := &fakeExecHandler{
		expectedNamespace: "sandbox-u-42",
		expectedName:      "sandbox-proj-1",
		expectedCommand:   "ls -la",
		stdout:            []string{"file1\n", "file2\n"},
		exitCode:          0,
		t:                 t,
	}
	exec := newTestExecutor(t, h)

	out, err := exec.Exec(context.Background(), "sandbox-proj-1", "sandbox-u-42", "ls -la", 30*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "file1\nfile2\n"
	if out != want {
		t.Fatalf("output mismatch: got %q want %q", out, want)
	}
}

func TestControllerExecutor_Exec_StderrMerged(t *testing.T) {
	h := &fakeExecHandler{
		expectedNamespace: "ns",
		expectedName:      "sb",
		expectedCommand:   "echo out; echo err >&2",
		stdout:            []string{"out\n"},
		stderr:            []string{"err\n"},
		exitCode:          0,
		t:                 t,
	}
	exec := newTestExecutor(t, h)

	out, err := exec.Exec(context.Background(), "sb", "ns", "echo out; echo err >&2", 30*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// stdout first, then stderr sep by newline.
	want := "out\n\nerr\n"
	if out != want {
		t.Fatalf("output mismatch: got %q want %q", out, want)
	}
}

func TestControllerExecutor_Exec_NonZeroExit(t *testing.T) {
	h := &fakeExecHandler{
		expectedNamespace: "ns",
		expectedName:      "sb",
		expectedCommand:   "false",
		stdout:            []string{},
		stderr:            []string{"something failed\n"},
		exitCode:          7,
		t:                 t,
	}
	exec := newTestExecutor(t, h)

	out, err := exec.Exec(context.Background(), "sb", "ns", "false", 30*time.Second)
	if err == nil {
		t.Fatalf("expected error for non-zero exit, got nil")
	}
	if !strings.Contains(err.Error(), "exited 7") {
		t.Fatalf("error message missing exit code: %v", err)
	}
	if !strings.Contains(out, "something failed") {
		t.Fatalf("output missing stderr payload: %q", out)
	}
}

func TestControllerExecutor_Exec_TimeoutMappedToDomainErr(t *testing.T) {
	h := &fakeExecHandler{
		expectedNamespace: "ns",
		expectedName:      "sb",
		expectedCommand:   "sleep 9999",
		exitCode:          124,
		t:                 t,
	}
	exec := newTestExecutor(t, h)

	_, err := exec.Exec(context.Background(), "sb", "ns", "sleep 9999", 30*time.Second)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	if !errors.Is(err, domain.ErrExecTimeout) {
		t.Fatalf("expected ErrExecTimeout, got %v", err)
	}
}

func TestControllerExecutor_Exec_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "sandbox not ready", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	exec := &ControllerExecutor{
		execEndpoint: srv.URL,
		httpClient:   &http.Client{},
	}

	_, err := exec.Exec(context.Background(), "sb", "ns", "ls", 30*time.Second)
	if err == nil {
		t.Fatalf("expected HTTP error, got nil")
	}
	if !strings.Contains(err.Error(), "404") || !strings.Contains(err.Error(), "sandbox not ready") {
		t.Fatalf("error should carry status + body, got %v", err)
	}
}

func TestControllerExecutor_Exec_ChunkedBody(t *testing.T) {
	// Client must read flushed chunks early.
	var observedChunks int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)

		// Three writes, three flushes, then exit.
		for i := range 3 {
			writeJSON(w, map[string]any{"stream": "stdout", "data": fmt.Sprintf("chunk-%d\n", i)})
			flusher.Flush()
			observedChunks++
		}
		writeJSON(w, map[string]any{"stream": "exit", "code": 0})
		flusher.Flush()
	}))
	t.Cleanup(srv.Close)

	exec := &ControllerExecutor{
		execEndpoint: srv.URL,
		httpClient:   &http.Client{},
	}

	out, err := exec.Exec(context.Background(), "sb", "ns", "anything", 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "chunk-0") || !strings.Contains(out, "chunk-2") {
		t.Fatalf("missing chunks: %q", out)
	}
	if observedChunks < 3 {
		t.Fatalf("expected at least 3 chunk writes, got %d", observedChunks)
	}
}

func TestControllerExecutor_Exec_NoExitEvent(t *testing.T) {
	// Stream ends without exit event.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		writeJSON(w, map[string]any{"stream": "stdout", "data": "partial\n"})
		// Close without writing exit.
	}))
	t.Cleanup(srv.Close)

	exec := &ControllerExecutor{
		execEndpoint: srv.URL,
		httpClient:   &http.Client{},
	}

	_, err := exec.Exec(context.Background(), "sb", "ns", "anything", 5*time.Second)
	if err == nil {
		t.Fatalf("expected error for missing exit event, got nil")
	}
	if !strings.Contains(err.Error(), "without exit event") {
		t.Fatalf("error should mention missing exit: %v", err)
	}
}

func TestControllerExecutor_Exec_TrimsURLSlash(t *testing.T) {
	// Trailing slash must not double.
	var seenPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		writeJSON(w, map[string]any{"stream": "exit", "code": 0})
	}))
	t.Cleanup(srv.Close)

	exec := &ControllerExecutor{
		execEndpoint: srv.URL + "/",
		httpClient:   &http.Client{},
	}

	if _, err := exec.Exec(context.Background(), "sb", "ns", "ls", 5*time.Second); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(seenPath, "//") {
		t.Fatalf("double slash in path: %q", seenPath)
	}
	if !strings.HasSuffix(seenPath, "/v1/sandboxes/ns/sb/exec") {
		t.Fatalf("wrong path: %q", seenPath)
	}
}

// JSON line reader survives partial flushes.
func TestReadExecStream_PartialLines(t *testing.T) {
	pr, pw := io.Pipe()
	go func() {
		// Full line, then half-line.
		_, _ = pw.Write([]byte(`{"stream":"stdout","data":"a"}` + "\n"))
		_, _ = pw.Write([]byte(`{"stream":"stdout","data`))
		_ = pw.Close()
	}()

	c := &ControllerExecutor{}
	out, err := c.readExecStream(pr)
	// Must surface missing-exit, not swallow.
	if err == nil || !strings.Contains(err.Error(), "without exit event") {
		t.Fatalf("expected missing-exit error, got out=%q err=%v", out, err)
	}
}

// bufio.Scanner keeps trailing non-newline line.
func TestReadExecStream_LastLineNoNewline(t *testing.T) {
	body := `{"stream":"stdout","data":"hi"}` + "\n" + `{"stream":"exit","code":0}`
	c := &ControllerExecutor{}
	out, err := c.readExecStream(bufio.NewReader(strings.NewReader(body)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "hi" {
		t.Fatalf("got %q want %q", out, "hi")
	}
}
