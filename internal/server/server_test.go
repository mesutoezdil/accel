package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mesutoezdil/accel/internal/collect"
	"github.com/mesutoezdil/accel/internal/config"
	"github.com/mesutoezdil/accel/internal/provider"
	"github.com/mesutoezdil/accel/internal/provider/sim"
)

func TestServer(t *testing.T) {
	e := collect.New([]provider.Provider{sim.Provider(2)}, config.Default(), nil, true)
	e.Detect()
	e.Collect(context.Background())
	srv := httptest.NewServer(Handler(e, "secret"))
	defer srv.Close()

	get := func(path, token string) (int, string) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		var b strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			b.Write(buf[:n])
			if err != nil {
				break
			}
		}
		return resp.StatusCode, b.String()
	}
	if code, _ := get("/api/snapshot", ""); code != http.StatusUnauthorized {
		t.Fatalf("no token: %d", code)
	}
	if code, _ := get("/healthz", ""); code != http.StatusOK {
		t.Fatalf("healthz needs no token: %d", code)
	}
	code, body := get("/metrics", "secret")
	if code != http.StatusOK || !strings.Contains(body, `accel_device_util_percent{node=`) || !strings.Contains(body, "accel_device_health_score") {
		t.Fatalf("metrics %d %s", code, body[:min(len(body), 300)])
	}
	if code, body := get("/api/snapshot", "secret"); code != http.StatusOK || !strings.Contains(body, `"devices"`) {
		t.Fatalf("snapshot %d", code)
	}
}
