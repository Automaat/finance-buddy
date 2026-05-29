package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStatusText(t *testing.T) {
	cases := map[int]string{
		100: "1xx", 200: "2xx", 201: "2xx", 302: "3xx",
		404: "4xx", 422: "4xx", 500: "5xx", 503: "5xx",
	}
	for code, want := range cases {
		if got := statusText(code); got != want {
			t.Errorf("statusText(%d) = %q, want %q", code, got, want)
		}
	}
}

func scrape(t *testing.T) string {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), "GET", "/metrics", nil)
	Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("/metrics status = %d", rec.Code)
	}
	return rec.Body.String()
}

func TestObserveRequestExposesSeries(t *testing.T) {
	ObserveRequest("GET", "/api/test", 200, 5*time.Millisecond)
	body := scrape(t)
	if !strings.Contains(body, "fb_http_requests_total") {
		t.Fatalf("expected fb_http_requests_total in /metrics output")
	}
	if !strings.Contains(body, `route="/api/test"`) || !strings.Contains(body, `status="2xx"`) {
		t.Fatalf("expected the observed labels in output:\n%s", body)
	}
	if !strings.Contains(body, "fb_http_request_duration_seconds") {
		t.Fatalf("expected the duration histogram in output")
	}
}

func TestSchedulerRunExposesSeries(t *testing.T) {
	SchedulerRun("cpi", "success")
	body := scrape(t)
	if !strings.Contains(body, "fb_scheduler_runs_total") {
		t.Fatalf("expected fb_scheduler_runs_total in /metrics output")
	}
	if !strings.Contains(body, `scheduler="cpi"`) || !strings.Contains(body, `result="success"`) {
		t.Fatalf("expected scheduler labels in output:\n%s", body)
	}
}

func TestObserveRequestEmptyRouteFallsBack(t *testing.T) {
	ObserveRequest("GET", "", 404, time.Millisecond)
	if !strings.Contains(scrape(t), `route="unmatched"`) {
		t.Fatalf("empty route should be recorded as unmatched")
	}
}

func TestPoolCollectorExposesStats(t *testing.T) {
	RegisterPoolCollector(func() PoolStats {
		return PoolStats{Total: 4, Idle: 3, Acquired: 1, Max: 10, AcquireCount: 42, EmptyAcquireCount: 7}
	})
	body := scrape(t)
	for _, want := range []string{
		"fb_db_pool_max_conns 10",
		"fb_db_pool_total_conns 4",
		"fb_db_pool_acquired_conns 1",
		"fb_db_pool_empty_acquire_total 7",
		"fb_db_pool_acquire_total 42",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in /metrics output:\n%s", want, body)
		}
	}
}
