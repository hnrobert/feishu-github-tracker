package notifier

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
	"github.com/hnrobert/feishu-github-tracker/internal/logger"
)

func TestResolveURL(t *testing.T) {
	// initialize logger for tests
	_ = logger.Init("debug", t.TempDir())

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
