package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/smallyunet/echoevm/internal/differential"
	"github.com/smallyunet/echoevm/internal/eth/hexutil"
	"github.com/smallyunet/echoevm/internal/replay"
)

type readinessRPC struct {
	calls       int
	recentCalls int
	err         error
}

func (r *readinessRPC) CallContext(_ context.Context, result any, method string, _ ...any) error {
	r.calls++
	if r.err != nil && method == "debug_traceCall" {
		return r.err
	}
	switch method {
	case "eth_chainId":
		*result.(*hexutil.Uint64) = 1
	case "debug_traceCall":
		*result.(*json.RawMessage) = json.RawMessage(`{}`)
	case "eth_getBlockByNumber":
		r.recentCalls++
		return json.Unmarshal([]byte(`{"number":"0x2a","transactions":["0x0000000000000000000000000000000000000000000000000000000000000001","0x0000000000000000000000000000000000000000000000000000000000000002"]}`), result)
	default:
		return errors.New("unexpected method " + method)
	}
	return nil
}

func TestRecentTransactionsAPIUsesShortOnDemandCache(t *testing.T) {
	rpc := &readinessRPC{}
	server := NewServer(":0")
	server.verification = replay.NewVerificationServiceWithCaller(rpc)
	for range 2 {
		recorder := httptest.NewRecorder()
		server.serveRecentTransactions(recorder, httptest.NewRequest(http.MethodGet, "/api/recent-transactions", nil))
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"blockNumber":42`) || !strings.Contains(recorder.Body.String(), `"transactionIndex":1`) {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("Cache-Control=%q", got)
		}
	}
	if rpc.recentCalls != 1 {
		t.Fatalf("latest block RPC calls = %d, want 1", rpc.recentCalls)
	}

	recorder := httptest.NewRecorder()
	server.serveRecentTransactions(recorder, httptest.NewRequest(http.MethodPost, "/api/recent-transactions", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status=%d", recorder.Code)
	}
}

func TestExecutionAPI(t *testing.T) {
	server := NewDifferentialServer(":0", differential.DefaultEngine())
	req := httptest.NewRequest(http.MethodPost, "/api/diff", strings.NewReader(`{"fork":"Cancun","bytecode":"60026003015f5260205ff3","calldata":"0x","gasLimit":1000000}`))
	recorder := httptest.NewRecorder()
	server.serveDiff(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var result differential.ExecutionResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != differential.StatusSuccess {
		t.Fatalf("unexpected execution: %+v", result)
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

func TestDifferentialReadinessCachesTraceCapability(t *testing.T) {
	rpc := &readinessRPC{}
	server := NewServer(":0")
	server.verification = replay.NewVerificationServiceWithCaller(rpc)
	for range 2 {
		recorder := httptest.NewRecorder()
		server.serveReady(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"status":"ready"`) {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	}
	if rpc.calls != 3 {
		t.Fatalf("RPC calls = %d, want 3 after cached readiness", rpc.calls)
	}
}

func TestDifferentialReadinessReportsTraceUnavailable(t *testing.T) {
	server := NewServer(":0")
	server.verification = replay.NewVerificationServiceWithCaller(&readinessRPC{err: errors.New("trace method disabled")})
	recorder := httptest.NewRecorder()
	server.serveReady(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), `"status":"not_ready"`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestReplayHTTPStatusClassification(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{errors.New("bad input"), http.StatusBadRequest},
		{replay.NewError(replay.ErrorNotFound, errors.New("missing")), http.StatusNotFound},
		{replay.NewError(replay.ErrorConflict, errors.New("pending")), http.StatusConflict},
		{replay.NewError(replay.ErrorUpstream, errors.New("RPC failed")), http.StatusBadGateway},
		{replay.NewError(replay.ErrorUnavailable, errors.New("trace disabled")), http.StatusServiceUnavailable},
		{context.DeadlineExceeded, http.StatusGatewayTimeout},
	}
	for _, test := range tests {
		if got := replayHTTPStatus(test.err); got != test.want {
			t.Fatalf("replayHTTPStatus(%v) = %d, want %d", test.err, got, test.want)
		}
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

func TestDifferentialIndexSupportsShareableTransactionPath(t *testing.T) {
	server := NewDifferentialServer(":0", differential.DefaultEngine())
	hash := "0x" + strings.Repeat("a", 64)
	for _, path := range []string{"/tx/" + hash, "/tx/" + hash + "/"} {
		recorder := httptest.NewRecorder()
		server.serveDifferentialIndex(recorder, httptest.NewRequest(http.MethodGet, path+"?profile=revert", nil))
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "Transaction Verification") {
			t.Fatalf("path=%s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
	}
	for _, path := range []string{"/tx/0x1234", "/tx/" + strings.Repeat("z", 66), "/other"} {
		recorder := httptest.NewRecorder()
		server.serveDifferentialIndex(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("path=%s status=%d", path, recorder.Code)
		}
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
	server.verifySlots = make(chan struct{}, 1)
	recorder := httptest.NewRecorder()
	server.serveVerify(recorder, httptest.NewRequest(http.MethodPost, "/api/verify", strings.NewReader(`{"input":"0x00"}`)))
	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), "trace-capable RPC") {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	recorder = httptest.NewRecorder()
	server.serveVerify(recorder, httptest.NewRequest(http.MethodGet, "/api/verify", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status=%d", recorder.Code)
	}
}

func TestReplayAPIRejectsUnboundedEvidencePresentation(t *testing.T) {
	rpc := &readinessRPC{}
	server := NewServer(":0")
	server.verification = replay.NewVerificationServiceWithCaller(rpc)
	server.verifySlots = make(chan struct{}, 1)
	for _, body := range []string{
		`{"input":"0x00","profile":"auto","limit":0,"maxMemoryBytes":256}`,
		`{"input":"0x00","profile":"auto","limit":201,"maxMemoryBytes":256}`,
		`{"input":"0x00","profile":"auto","limit":40,"maxMemoryBytes":4097}`,
	} {
		recorder := httptest.NewRecorder()
		server.serveVerify(recorder, httptest.NewRequest(http.MethodPost, "/api/verify", strings.NewReader(body)))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("body=%s status=%d response=%s", body, recorder.Code, recorder.Body.String())
		}
	}
	if rpc.calls != 0 {
		t.Fatalf("RPC calls = %d, want 0 for rejected presentation bounds", rpc.calls)
	}
}
