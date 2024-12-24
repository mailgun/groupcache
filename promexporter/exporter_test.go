package promexporter_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/mailgun/groupcache/v2"
	"github.com/mailgun/groupcache/v2/promexporter"
	"github.com/mailgun/holster/v4/retry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil/promlint"
	promdto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MetricInfo struct {
	Name   string
	Labels prometheus.Labels
}

const (
	pingRoute    = "/ping"
	metricsRoute = "/metrics"
	ttl          = time.Minute
)

func TestExporter(t *testing.T) {
	// Given
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup groupcache.
	getter := func(_ context.Context, key string, dest groupcache.Sink) error {
		err := dest.SetString("foobar", time.Now().Add(ttl))
		require.NoError(t, err)
		return nil
	}
	group := groupcache.NewGroup("test", 10_000, groupcache.GetterFunc(getter))
	gcexporter := promexporter.NewExporter()
	prometheus.MustRegister(gcexporter)

	// Setup HTTP server.
	mux := http.NewServeMux()
	mux.HandleFunc(pingRoute, pingHandler)
	mux.Handle(metricsRoute, promhttp.Handler())
	httpSrv, err := startHTTPServer(ctx, mux, &wg)
	require.NoError(t, err)
	t.Logf("HTTP server ready at %s", httpSrv.Addr)
	defer func() {
		// Tear down.
		t.Log("HTTP server shutting down...")
		err = httpSrv.Shutdown(ctx)
		require.NoError(t, err)
		wg.Wait()
	}()

	// When
	// Add cache activity.
	for i := 0; i < 10; i++ {
		var value string
		err = group.Get(ctx, fmt.Sprintf("key%d", i), groupcache.StringSink(&value))
		require.NoError(t, err)
	}

	// Then
	// Get metrics from endpoint.
	rs, err := getURL(ctx, fmt.Sprintf("http://%s%s", httpSrv.Addr, metricsRoute))
	require.NoError(t, err)
	content, err := io.ReadAll(rs.Body)
	defer rs.Body.Close()
	require.NoError(t, err)

	// Parse metrics and assert expected values are found.
	expectedMetrics := []struct {
		Name   string
		Labels prometheus.Labels
	}{
		{Name: "groupcache_cache_bytes", Labels: prometheus.Labels{"type": "main"}},
		{Name: "groupcache_cache_bytes", Labels: prometheus.Labels{"type": "hot"}},
		{Name: "groupcache_cache_evictions_nonexpired_total", Labels: prometheus.Labels{"type": "main"}},
		{Name: "groupcache_cache_evictions_nonexpired_total", Labels: prometheus.Labels{"type": "hot"}},
		{Name: "groupcache_cache_evictions_total", Labels: prometheus.Labels{"type": "main"}},
		{Name: "groupcache_cache_evictions_total", Labels: prometheus.Labels{"type": "hot"}},
		{Name: "groupcache_cache_gets_total", Labels: prometheus.Labels{"type": "main"}},
		{Name: "groupcache_cache_gets_total", Labels: prometheus.Labels{"type": "hot"}},
		{Name: "groupcache_cache_hits_total", Labels: prometheus.Labels{"type": "main"}},
		{Name: "groupcache_cache_hits_total", Labels: prometheus.Labels{"type": "hot"}},
		{Name: "groupcache_cache_items", Labels: prometheus.Labels{"type": "main"}},
		{Name: "groupcache_cache_items", Labels: prometheus.Labels{"type": "hot"}},
		{Name: "groupcache_get_from_peers_latency_lower"},
		{Name: "groupcache_gets_total"},
		{Name: "groupcache_hits_total"},
		{Name: "groupcache_loads_deduped_total"},
		{Name: "groupcache_loads_total"},
		{Name: "groupcache_local_load_errs_total"},
		{Name: "groupcache_local_loads_total"},
		{Name: "groupcache_peer_errors_total"},
		{Name: "groupcache_peer_loads_total"},
		{Name: "groupcache_server_requests_total"},
	}
	tp := new(expfmt.TextParser)
	mfs, err := tp.TextToMetricFamilies(bytes.NewReader(content))
	require.NoError(t, err)

	for _, expectedMetric := range expectedMetrics {
		testName := fmt.Sprintf("Metric %s{%s}", expectedMetric.Name, labelsToString(expectedMetric.Labels))
		t.Run(testName, func(t *testing.T) {
			assertContainsMetric(t, expectedMetric, mfs)
		})
	}

	t.Run("Lint", func(t *testing.T) {
		linter := promlint.New(bytes.NewReader(content))
		problems, err := linter.Lint()
		require.NoError(t, err)
		for _, problem := range problems {
			assert.Fail(t, fmt.Sprintf("%#v", problem))
		}
	})
}

// Start an HTTP server on a dynamic port.
func startHTTPServer(ctx context.Context, mux *http.ServeMux, wg *sync.WaitGroup) (*http.Server, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	httpSrv := &http.Server{
		Addr:              listener.Addr().(*net.TCPAddr).AddrPort().String(),
		Handler:           mux,
		ReadHeaderTimeout: time.Minute,
	}
	wg.Add(1)
	go func() {
		_ = httpSrv.Serve(listener)
		wg.Done()
	}()
	err = waitForReady(ctx, httpSrv)
	return httpSrv, err
}

func waitForReady(ctx context.Context, httpSrv *http.Server) error {
	httpClt := http.DefaultClient
	return retry.Until(ctx, retry.Interval(20*time.Millisecond), func(ctx context.Context, _ int) error {
		pingURL := fmt.Sprintf("http://%s%s", httpSrv.Addr, pingRoute)
		ctx2, cancel2 := context.WithTimeout(ctx, 10*time.Second)
		defer cancel2()
		rq, err := http.NewRequestWithContext(ctx2, http.MethodGet, pingURL, http.NoBody)
		if err != nil {
			return err
		}
		rs, err := httpClt.Do(rq)
		if err != nil {
			return err
		}
		rs.Body.Close()
		return nil
	})
}

func getURL(ctx context.Context, u string) (*http.Response, error) {
	rq, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(rq)
}

// Assert expected metric name and labels are present.
func assertContainsMetric(t *testing.T, expected MetricInfo, mfs map[string]*promdto.MetricFamily) {
	mf, ok := mfs[expected.Name]
	if !assert.True(t, ok, "Metric not found") {
		return
	}

	matchFlag := false
	for _, metric := range mf.Metric {
	LM1:
		for key, value := range expected.Labels {
			for _, label := range metric.Label {
				if label.Name == nil || label.Value == nil {
					continue
				}
				if *label.Name == key && *label.Value == value {
					// Label match, go to next expected label.
					continue LM1
				}
			}
			break LM1
		}
		// All labels match.
		matchFlag = true
		break
	}
	assert.True(t, matchFlag, "Labels mismatch")
}

func pingHandler(writer http.ResponseWriter, rq *http.Request) {
	writer.WriteHeader(http.StatusOK)
}

func labelsToString(labels prometheus.Labels) string {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	for i, key := range keys {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(fmt.Sprintf("%s=%q", key, labels[key]))
	}
	return buf.String()
}
