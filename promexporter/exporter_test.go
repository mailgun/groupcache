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
	group := newGroup(t, t.Name())
	defer groupcache.DeregisterGroup(group.Name())
	gcexporter := promexporter.NewExporter()
	registry := prometheus.NewRegistry()
	registry.MustRegister(gcexporter)

	// Setup metrics HTTP server.
	httpSrv, err := startMetricsServer(ctx, t, registry, &wg)
	require.NoError(t, err)
	defer func() {
		// Tear down.
		t.Log("HTTP server shutting down...")
		err = httpSrv.Shutdown(ctx)
		require.NoError(t, err)
		wg.Wait()
	}()

	// When
	metricsContent, err := getMetrics(ctx, httpSrv)
	require.NoError(t, err)

	// Then
	// Parse metrics and assert expected values are found.
	mfs, err := parseMetricsContent(metricsContent)
	require.NoError(t, err)
	expectedMetrics := []MetricInfo{
		{Name: "groupcache_cache_bytes", Labels: prometheus.Labels{"group": group.Name(), "type": "main"}},
		{Name: "groupcache_cache_bytes", Labels: prometheus.Labels{"group": group.Name(), "type": "hot"}},
		{Name: "groupcache_cache_evictions_nonexpired_total", Labels: prometheus.Labels{"group": group.Name(), "type": "main"}},
		{Name: "groupcache_cache_evictions_nonexpired_total", Labels: prometheus.Labels{"group": group.Name(), "type": "hot"}},
		{Name: "groupcache_cache_evictions_total", Labels: prometheus.Labels{"group": group.Name(), "type": "main"}},
		{Name: "groupcache_cache_evictions_total", Labels: prometheus.Labels{"group": group.Name(), "type": "hot"}},
		{Name: "groupcache_cache_gets_total", Labels: prometheus.Labels{"group": group.Name(), "type": "main"}},
		{Name: "groupcache_cache_gets_total", Labels: prometheus.Labels{"group": group.Name(), "type": "hot"}},
		{Name: "groupcache_cache_hits_total", Labels: prometheus.Labels{"group": group.Name(), "type": "main"}},
		{Name: "groupcache_cache_hits_total", Labels: prometheus.Labels{"group": group.Name(), "type": "hot"}},
		{Name: "groupcache_cache_items", Labels: prometheus.Labels{"group": group.Name(), "type": "main"}},
		{Name: "groupcache_cache_items", Labels: prometheus.Labels{"group": group.Name(), "type": "hot"}},
		{Name: "groupcache_get_from_peers_latency_lower", Labels: prometheus.Labels{"group": group.Name()}},
		{Name: "groupcache_gets_total", Labels: prometheus.Labels{"group": group.Name()}},
		{Name: "groupcache_hits_total", Labels: prometheus.Labels{"group": group.Name()}},
		{Name: "groupcache_loads_deduped_total", Labels: prometheus.Labels{"group": group.Name()}},
		{Name: "groupcache_loads_total", Labels: prometheus.Labels{"group": group.Name()}},
		{Name: "groupcache_local_load_errs_total", Labels: prometheus.Labels{"group": group.Name()}},
		{Name: "groupcache_local_loads_total", Labels: prometheus.Labels{"group": group.Name()}},
		{Name: "groupcache_peer_errors_total", Labels: prometheus.Labels{"group": group.Name()}},
		{Name: "groupcache_peer_loads_total", Labels: prometheus.Labels{"group": group.Name()}},
		{Name: "groupcache_server_requests_total", Labels: prometheus.Labels{"group": group.Name()}},
	}
	for _, expectedMetric := range expectedMetrics {
		testName := fmt.Sprintf("Metric exported %s{%s}", expectedMetric.Name, labelsToString(expectedMetric.Labels))
		t.Run(testName, func(t *testing.T) {
			assertContainsMetric(t, expectedMetric, mfs)
		})
	}

	t.Run("Lint", func(t *testing.T) {
		linter := promlint.New(bytes.NewReader(metricsContent))
		problems, err := linter.Lint()
		require.NoError(t, err)
		for _, problem := range problems {
			assert.Fail(t, fmt.Sprintf("%#v", problem))
		}
	})
}

func TestExporterWithNamespace(t *testing.T) {
	// Given
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup groupcache.
	group := newGroup(t, t.Name())
	defer groupcache.DeregisterGroup(group.Name())
	gcexporter := promexporter.NewExporter(promexporter.WithNamespace("mynamespace"))
	registry := prometheus.NewRegistry()
	registry.MustRegister(gcexporter)

	// Setup metrics HTTP server.
	httpSrv, err := startMetricsServer(ctx, t, registry, &wg)
	require.NoError(t, err)
	defer func() {
		// Tear down.
		t.Log("HTTP server shutting down...")
		err = httpSrv.Shutdown(ctx)
		require.NoError(t, err)
		wg.Wait()
	}()

	// When
	metricsContent, err := getMetrics(ctx, httpSrv)
	require.NoError(t, err)

	// Then
	// Parse metrics and assert expected values are found.
	mfs, err := parseMetricsContent(metricsContent)
	require.NoError(t, err)
	expectedMetrics := []MetricInfo{
		{Name: "mynamespace_groupcache_cache_bytes", Labels: prometheus.Labels{"group": group.Name(), "type": "main"}},
		{Name: "mynamespace_groupcache_cache_bytes", Labels: prometheus.Labels{"group": group.Name(), "type": "hot"}},
		{Name: "mynamespace_groupcache_gets_total", Labels: prometheus.Labels{"group": group.Name()}},
	}
	for _, expectedMetric := range expectedMetrics {
		testName := fmt.Sprintf("Metric exported %s{%s}", expectedMetric.Name, labelsToString(expectedMetric.Labels))
		t.Run(testName, func(t *testing.T) {
			assertContainsMetric(t, expectedMetric, mfs)
		})
	}
}

func TestExporterWithLabels(t *testing.T) {
	// Given
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup groupcache.
	group := newGroup(t, t.Name())
	defer groupcache.DeregisterGroup(group.Name())
	gcexporter := promexporter.NewExporter(promexporter.WithLabels(map[string]string{
		"accountid": "0001",
		"dc":        "west",
	}))
	registry := prometheus.NewRegistry()
	registry.MustRegister(gcexporter)

	// Setup metrics HTTP server.
	httpSrv, err := startMetricsServer(ctx, t, registry, &wg)
	require.NoError(t, err)
	defer func() {
		// Tear down.
		t.Log("HTTP server shutting down...")
		err = httpSrv.Shutdown(ctx)
		require.NoError(t, err)
		wg.Wait()
	}()

	// When
	metricsContent, err := getMetrics(ctx, httpSrv)
	require.NoError(t, err)

	// Then
	// Parse metrics and assert expected values are found.
	mfs, err := parseMetricsContent(metricsContent)
	require.NoError(t, err)
	expectedMetrics := []MetricInfo{
		{Name: "groupcache_cache_bytes", Labels: prometheus.Labels{"group": group.Name(), "type": "main", "accountid": "0001", "dc": "west"}},
		{Name: "groupcache_cache_bytes", Labels: prometheus.Labels{"group": group.Name(), "type": "hot", "accountid": "0001", "dc": "west"}},
		{Name: "groupcache_gets_total", Labels: prometheus.Labels{"group": group.Name(), "accountid": "0001", "dc": "west"}},
	}
	for _, expectedMetric := range expectedMetrics {
		assertContainsMetric(t, expectedMetric, mfs)
	}
}

func TestExporterWithGroups(t *testing.T) {
	// Given
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup groupcache.
	group1 := newGroup(t, t.Name()+"_1")
	group2 := newGroup(t, t.Name()+"_2")
	group3 := newGroup(t, t.Name()+"_3")
	defer func() {
		groupcache.DeregisterGroup(group1.Name())
		groupcache.DeregisterGroup(group2.Name())
		groupcache.DeregisterGroup(group3.Name())
	}()
	gcexporter := promexporter.NewExporter(promexporter.WithGroups(func() []*groupcache.Group {
		return []*groupcache.Group{group1, group2}
	}))
	registry := prometheus.NewRegistry()
	registry.MustRegister(gcexporter)

	// Setup metrics HTTP server.
	httpSrv, err := startMetricsServer(ctx, t, registry, &wg)
	require.NoError(t, err)
	defer func() {
		// Tear down.
		t.Log("HTTP server shutting down...")
		err = httpSrv.Shutdown(ctx)
		require.NoError(t, err)
		wg.Wait()
	}()

	// When
	metricsContent, err := getMetrics(ctx, httpSrv)
	require.NoError(t, err)

	// Then
	// Parse metrics and assert expected values are found.
	mfs, err := parseMetricsContent(metricsContent)
	require.NoError(t, err)
	expectedMetrics := []MetricInfo{
		{Name: "groupcache_cache_bytes", Labels: prometheus.Labels{"group": group1.Name(), "type": "main"}},
		{Name: "groupcache_cache_bytes", Labels: prometheus.Labels{"group": group1.Name(), "type": "hot"}},
		{Name: "groupcache_gets_total", Labels: prometheus.Labels{"group": group1.Name()}},
		{Name: "groupcache_cache_bytes", Labels: prometheus.Labels{"group": group2.Name(), "type": "main"}},
		{Name: "groupcache_cache_bytes", Labels: prometheus.Labels{"group": group2.Name(), "type": "hot"}},
		{Name: "groupcache_gets_total", Labels: prometheus.Labels{"group": group2.Name()}},
	}
	for _, expectedMetric := range expectedMetrics {
		assertContainsMetric(t, expectedMetric, mfs)
	}
	// Assert unexpected values are not found.
	unexpectedMetrics := []MetricInfo{
		{Name: "groupcache_cache_bytes", Labels: prometheus.Labels{"group": group3.Name(), "type": "main"}},
		{Name: "groupcache_cache_bytes", Labels: prometheus.Labels{"group": group3.Name(), "type": "hot"}},
		{Name: "groupcache_gets_total", Labels: prometheus.Labels{"group": group3.Name()}},
	}
	for _, unexpectedMetric := range unexpectedMetrics {
		assertNotContainsMetric(t, unexpectedMetric, mfs)
	}
}

// Create new test group.
func newGroup(t *testing.T, name string) *groupcache.Group {
	getter := func(_ context.Context, key string, dest groupcache.Sink) error {
		err := dest.SetString("foobar", time.Now().Add(ttl))
		require.NoError(t, err)
		return nil
	}
	return groupcache.NewGroup(name, 10_000, groupcache.GetterFunc(getter))
}

// Start an HTTP server on a dynamic port.
func startMetricsServer(ctx context.Context, t *testing.T, registry *prometheus.Registry, wg *sync.WaitGroup) (*http.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc(pingRoute, pingHandler)
	// mux.Handle(metricsRoute, promhttp.Handler())
	mux.Handle(metricsRoute, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
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
	t.Logf("HTTP server ready at %s", httpSrv.Addr)
	return httpSrv, nil
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

func pingHandler(writer http.ResponseWriter, rq *http.Request) {
	writer.WriteHeader(http.StatusOK)
}

func getURL(ctx context.Context, u string) (*http.Response, error) {
	rq, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(rq)
}

// Request metrics endpoint and return content.
func getMetrics(ctx context.Context, httpSrv *http.Server) ([]byte, error) {
	rs, err := getURL(ctx, fmt.Sprintf("http://%s%s", httpSrv.Addr, metricsRoute))
	if err != nil {
		return nil, err
	}
	content, err := io.ReadAll(rs.Body)
	rs.Body.Close()
	return content, err
}

// Parse metrics content into Prometheus metric structures.
func parseMetricsContent(content []byte) (map[string]*promdto.MetricFamily, error) {
	var tp expfmt.TextParser
	return tp.TextToMetricFamilies(bytes.NewReader(content))
}

func containsMetric(mi MetricInfo, mfs map[string]*promdto.MetricFamily) bool {
	mf, ok := mfs[mi.Name]
	if !ok {
		return false
	}

LM1:
	for _, metric := range mf.Metric {
	LM2:
		for key, value := range mi.Labels {
			for _, label := range metric.Label {
				if label.Name == nil || label.Value == nil {
					continue
				}
				if *label.Name == key && *label.Value == value {
					// Label match, go to next expected label.
					continue LM2
				}
			}
			// Expected label not found.
			continue LM1
		}
		// All labels match.
		return true
	}
	// No metrics match.
	return false
}

// Assert expected metric name and labels are present.
func assertContainsMetric(t *testing.T, expected MetricInfo, mfs map[string]*promdto.MetricFamily) {
	assert.True(t, containsMetric(expected, mfs), "Metric not found: %s", expected.String())
}

// Assert expected metric name and labels are not present.
func assertNotContainsMetric(t *testing.T, unexpected MetricInfo, mfs map[string]*promdto.MetricFamily) {
	assert.False(t, containsMetric(unexpected, mfs), "Metric unexpectedly found: %s", unexpected.String())
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

func (mi *MetricInfo) String() string {
	return fmt.Sprintf("%s{%s}", mi.Name, labelsToString(mi.Labels))
}
