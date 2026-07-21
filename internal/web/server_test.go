package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/smallyunet/echoevm/internal/differential"
)

func TestDifferentialAPI(t *testing.T) {
	server := NewDifferentialServer(":0", differential.DefaultEngine())
	req := httptest.NewRequest(http.MethodPost, "/api/diff", strings.NewReader(`{"fork":"Cancun","bytecode":"60026003015f5260205ff3","calldata":"0x","gasLimit":1000000}`))
	recorder := httptest.NewRecorder()
	server.serveDiff(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var result differential.ComparisonResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Match {
		t.Fatalf("unexpected divergence: %+v", result.FirstDivergence)
	}
}

func TestDifferentialAPIRejectsInvalidRequests(t *testing.T) {
	server := NewDifferentialServer(":0", differential.DefaultEngine())
	for _, body := range []string{`{}`, `{"bytecode":"zz"}`, `{"bytecode":"00","extra":true}`, `{"bytecode":"00"}{"bytecode":"00"}`} {
		recorder := httptest.NewRecorder()
		server.serveDiff(recorder, httptest.NewRequest(http.MethodPost, "/api/diff", strings.NewReader(body)))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("body=%s status=%d response=%s", body, recorder.Code, recorder.Body.String())
		}
	}
	recorder := httptest.NewRecorder()
	server.serveDiff(recorder, httptest.NewRequest(http.MethodGet, "/api/diff", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status=%d", recorder.Code)
	}
}

func TestDifferentialHealth(t *testing.T) {
	server := NewDifferentialServer(":0", differential.DefaultEngine())
	recorder := httptest.NewRecorder()
	server.serveHealth(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"status":"ok"`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestDifferentialIndexVersionsAssets(t *testing.T) {
	server := NewDifferentialServer(":0", differential.DefaultEngine())
	recorder := httptest.NewRecorder()
	server.serveDifferentialIndex(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control=%q", got)
	}
	body := recorder.Body.String()
	for _, asset := range []string{"diff.css", "diff.js"} {
		want := "/assets/" + asset + "?v=" + server.assetVersion
		if !strings.Contains(body, want) {
			t.Fatalf("index does not reference %q", want)
		}
	}
	if strings.Contains(body, "{{ASSET_VERSION}}") {
		t.Fatal("asset version placeholder was not replaced")
	}
}

func TestVersionedAssetsAreImmutable(t *testing.T) {
	handler := cacheVersionedAsset(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/assets/diff.js?v=test", nil))
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control=%q", got)
	}
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/assets/diff.js", nil))
	if got := recorder.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("unversioned Cache-Control=%q", got)
	}
}

func TestReplayAPIRequiresConfiguredService(t *testing.T) {
	server := NewServer(":0")
	server.replaySlots = make(chan struct{}, 1)
	recorder := httptest.NewRecorder()
	server.serveReplay(recorder, httptest.NewRequest(http.MethodPost, "/api/replay", strings.NewReader(`{"input":"0x00"}`)))
	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), "trace-capable RPC") {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	recorder = httptest.NewRecorder()
	server.serveReplay(recorder, httptest.NewRequest(http.MethodGet, "/api/replay", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status=%d", recorder.Code)
	}
}
