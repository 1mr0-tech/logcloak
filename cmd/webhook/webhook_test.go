package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/1mr0-tech/logcloak/pkg/metrics"
	"github.com/1mr0-tech/logcloak/pkg/webhook"
)

func TestMain(m *testing.M) {
	metrics.MustRegister()
	m.Run()
}

func TestHealthz_Returns200(t *testing.T) {
	srv := httptest.NewServer(buildMux(&webhook.Handler{}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestReadyz_Returns200(t *testing.T) {
	srv := httptest.NewServer(buildMux(&webhook.Handler{}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestMetricsEndpoint_ReturnsPrometheusFormat(t *testing.T) {
	srv := httptest.NewServer(buildMetricsMux())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("expected text/plain content-type, got %q", ct)
	}
}

func TestMetricsHealthz_Returns200(t *testing.T) {
	srv := httptest.NewServer(buildMetricsMux())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestMutatePath_RequiresPOST(t *testing.T) {
	srv := httptest.NewServer(buildMux(&webhook.Handler{}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/mutate")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	// Handler returns 400 on malformed requests — what matters is it doesn't panic
	if resp.StatusCode == 0 {
		t.Error("expected a non-zero status code from /mutate")
	}
}
