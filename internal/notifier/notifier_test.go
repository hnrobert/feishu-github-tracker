package notifier

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
	"github.com/hnrobert/feishu-github-tracker/internal/logger"
)

func TestResolveURL(t *testing.T) {
	// initialize logger for tests
	_ = logger.Init("debug", t.TempDir())
	defer logger.Close()

	cfg := config.FeishuBotsConfig{
		FeishuBots: []config.FeishuBot{{Alias: "dev", URL: "https://example.com/webhook"}},
	}
	n := New(cfg)

	if got := n.resolveURL("dev"); got != "https://example.com/webhook" {
		t.Fatalf("expected alias to resolve, got %s", got)
	}

	if got := n.resolveURL("https://direct.example/hook"); got != "https://direct.example/hook" {
		t.Fatalf("expected direct URL passthrough, got %s", got)
	}

	if got := n.resolveURL("unknown"); got != "" {
		t.Fatalf("expected empty for unknown target, got %s", got)
	}
}

func TestSend_SuccessAndFailure(t *testing.T) {
	// initialize logger for tests
	_ = logger.Init("debug", t.TempDir())
	defer logger.Close()

	// Success server
	srvOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = body
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srvOK.Close()

	n := &Notifier{bots: map[string]string{}, client: srvOK.Client()}
	if err := n.Send([]string{srvOK.URL}, map[string]any{"hello": "world"}); err != nil {
		t.Fatalf("expected send success, got error: %v", err)
	}

	// Failure server
	srvFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srvFail.Close()

	n2 := &Notifier{bots: map[string]string{}, client: srvFail.Client()}
	err := n2.Send([]string{srvFail.URL}, map[string]any{"hello": "world"})
	if err == nil {
		t.Fatalf("expected error when server returns non-2xx")
	}
}

func TestSend_RejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(make([]byte, maxWebhookResponseBytes+1))
	}))
	defer server.Close()

	n := &Notifier{bots: map[string]string{}, client: server.Client()}
	if err := n.Send([]string{server.URL}, map[string]any{"hello": "world"}); err == nil {
		t.Fatal("expected oversized response to fail")
	}
}

func TestSend_ErrorIncludesResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("{\"code\":19021,\"msg\":\"sign match fail\"}"))
	}))
	defer server.Close()

	n := &Notifier{bots: map[string]string{}, client: server.Client()}
	err := n.Send([]string{server.URL}, map[string]any{"hello": "world"})
	if err == nil {
		t.Fatal("expected non-2xx response to fail")
	}
	if !strings.Contains(err.Error(), "19021") || !strings.Contains(err.Error(), "sign match fail") {
		t.Fatalf("error should include the Feishu response body for debugging, got: %v", err)
	}
}

func TestSummarizeBody(t *testing.T) {
	if got := summarizeBody(nil); got != "<empty response body>" {
		t.Fatalf("empty body = %q", got)
	}
	if got := summarizeBody([]byte("line1\nline2\twith\ttabs")); got != "line1 line2 with tabs" {
		t.Fatalf("whitespace not collapsed: %q", got)
	}
	long := strings.Repeat("x", 500)
	if got := summarizeBody([]byte(long)); len(got) > 220 || !strings.Contains(got, "truncated") {
		t.Fatalf("long body not truncated: %q (len=%d)", got, len(got))
	}
}
