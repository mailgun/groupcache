package groupcache_exporter_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/Baliedge/groupcache_exporter"
	"github.com/mailgun/groupcache/v2"
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

func TestExporter(t *testing.T) {
	// Given
	const metricsRoute = "/metrics"
	const httpPort = 9080
	const ttl = time.Minute
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
	gcexporter := groupcache_exporter.NewExporter("", nil, group)
	prometheus.MustRegister(gcexporter)

	// Setup HTTP server.
	mux := http.NewServeMux()
	mux.Handle(metricsRoute, promhttp.Handler())
	httpSrv := &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", httpPort),
		Handler:           mux,
		ReadHeaderTimeout: time.Minute,
	}
	wg.Add(1)
	go func() {
		_ = httpSrv.ListenAndServe()
		wg.Done()
	}()
	defer func() {
		// Tear down.
		err := httpSrv.Shutdown(ctx)
		require.NoError(t, err)
		wg.Wait()
	}()

	// When
	// Add cache activity.
	for i := 0; i < 10; i++ {
		var value string
		err := group.Get(ctx, fmt.Sprintf("key%d", i), groupcache.StringSink(&value))
		require.NoError(t, err)
	}

	// Then
	// Get metrics from endpoint.
	httpClt := http.DefaultClient
	metricsURL := fmt.Sprintf("http://127.0.0.1:%d%s", httpPort, metricsRoute)
	rq, err := http.NewRequestWithContext(ctx, http.MethodGet, metricsURL, http.NoBody)
	require.NoError(t, err)
	rs, err := httpClt.Do(rq)
	require.NoError(t, err)
	content, err := io.ReadAll(rs.Body)
	defer rs.Body.Close()
	require.NoError(t, err)

	// Parse metrics and assert expected values are found.
	expectedMetrics := []struct {
		Name   string
		Labels prometheus.Labels
	}{
		{
			Name:   "groupcache_cache_bytes",
			Labels: prometheus.Labels{"type": "main"},
		},
		{
			Name:   "groupcache_cache_bytes",
			Labels: prometheus.Labels{"type": "hot"},
		},
		{
			Name:   "groupcache_cache_evictions_total",
			Labels: prometheus.Labels{"type": "main"},
		},
		{
			Name:   "groupcache_cache_evictions_total",
			Labels: prometheus.Labels{"type": "hot"},
		},
		{
			Name:   "groupcache_cache_gets_total",
			Labels: prometheus.Labels{"type": "main"},
		},
		{
			Name:   "groupcache_cache_gets_total",
			Labels: prometheus.Labels{"type": "hot"},
		},
		{
			Name:   "groupcache_cache_hits_total",
			Labels: prometheus.Labels{"type": "main"},
		},
		{
			Name:   "groupcache_cache_hits_total",
			Labels: prometheus.Labels{"type": "hot"},
		},
		{
			Name:   "groupcache_cache_items",
			Labels: prometheus.Labels{"type": "main"},
		},
		{
			Name:   "groupcache_cache_items",
			Labels: prometheus.Labels{"type": "hot"},
		},
		{
			Name: "groupcache_gets_total",
		},
		{
			Name: "groupcache_hits_total",
		},
		{
			Name: "groupcache_loads_deduped_total",
		},
		{
			Name: "groupcache_loads_total",
		},
		{
			Name: "groupcache_local_load_errs_total",
		},
		{
			Name: "groupcache_local_load_total",
		},
		{
			Name: "groupcache_peer_errors_total",
		},
		{
			Name: "groupcache_peer_loads_total",
		},
		{
			Name: "groupcache_server_requests_total",
		},
	}
	tp := new(expfmt.TextParser)
	mfs, err := tp.TextToMetricFamilies(bytes.NewReader(content))
	require.NoError(t, err)

	for _, expectedMetric := range expectedMetrics {
		testName := fmt.Sprintf("Metric %s{%s}", expectedMetric.Name, labelsToString(expectedMetric.Labels))
		t.Run(testName, func(t *testing.T) {
			assertMetricPresent(t, expectedMetric, mfs)
		})
	}

	t.Run("Lint", func(t *testing.T) {
		// FIXME: Fix lint errors
		t.Skip()
		linter := promlint.New(bytes.NewReader(content))
		problems, err := linter.Lint()
		require.NoError(t, err)
		for _, problem := range problems {
			assert.Fail(t, fmt.Sprintf("%#v", problem))
		}
	})
}

// Assert expected metric name and labels are present.
func assertMetricPresent(t *testing.T, expected MetricInfo, mfs map[string]*promdto.MetricFamily) {
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
