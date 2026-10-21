package performance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleLoadOutput = `
=== RUN   TestLoad_Assets_List
    assets_load_test.go:316: === Assets/List 1000-concurrent 30s ===
    assets_load_test.go:317:   Total requests : 12345
    assets_load_test.go:318:   Success rate   : 99.9%
    assets_load_test.go:319:   Avg latency    : 12.3 ms
    assets_load_test.go:320:   P50 latency    : 9.0 ms
    assets_load_test.go:321:   P95 latency    : 25.0 ms
    assets_load_test.go:322:   P99 latency    : 40.0 ms
--- PASS: TestLoad_Assets_List (30.00s)
=== RUN   TestLoad_Health_Liveness
    health_search_load_test.go:41: === Health/Liveness 2000-concurrent 30s ===
    health_search_load_test.go:42:   Total requests : 999999
    health_search_load_test.go:43:   Success rate   : 100.0%
    health_search_load_test.go:44:   Avg latency    : 1.0 ms
    health_search_load_test.go:45:   P50 latency    : 1.0 ms
    health_search_load_test.go:46:   P95 latency    : 2.0 ms
    health_search_load_test.go:47:   P99 latency    : 3.0 ms
`

func TestParseLoadOutput(t *testing.T) {
	results, err := ParseLoadOutput(strings.NewReader(sampleLoadOutput))
	if err != nil {
		t.Fatalf("ParseLoadOutput: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d scenarios, want 2: %v", len(results), results)
	}

	list, ok := results["Assets/List 1000-concurrent 30s"]
	if !ok {
		t.Fatalf("missing Assets/List scenario, got %v", results)
	}
	if list.Total != 12345 || list.SuccessRate != 99.9 || list.P95Ms != 25.0 || list.P99Ms != 40.0 || list.AvgMs != 12.3 {
		t.Errorf("unexpected parsed entry: %+v", list)
	}

	health := results["Health/Liveness 2000-concurrent 30s"]
	if health.P95Ms != 2.0 || health.SuccessRate != 100.0 {
		t.Errorf("unexpected parsed health entry: %+v", health)
	}
}

func TestCompareDetectsRegression(t *testing.T) {
	bf := &BaselineFile{
		Baselines: map[string]BaselineEntry{
			"Assets/List": {AvgMs: 70, P95Ms: 100, SuccessRate: 99.0},
			"Health/Live": {AvgMs: 30, P95Ms: 50, SuccessRate: 99.9},
		},
	}

	current := map[string]BaselineEntry{
		// p95 150 > 100*1.2 and avg 100 > 70*1.25 -> regression
		"Assets/List": {AvgMs: 100, P95Ms: 150, SuccessRate: 99.5},
		// within bounds -> OK
		"Health/Live": {AvgMs: 32, P95Ms: 55, SuccessRate: 99.9},
	}

	report := Compare(bf, current, DefaultPolicy())
	if !report.HasRegression() {
		t.Fatalf("expected a regression, got none: %+v", report)
	}
	if report.Regressions != 1 {
		t.Fatalf("expected 1 regression, got %d", report.Regressions)
	}
	for _, c := range report.Comparisons {
		if c.Scenario == "Assets/List" && !c.Regression {
			t.Errorf("Assets/List should be flagged")
		}
		if c.Scenario == "Health/Live" && c.Regression {
			t.Errorf("Health/Live should not be flagged: %v", c.Reasons)
		}
	}
}

func TestCompareDetectsSuccessRateDrop(t *testing.T) {
	bf := &BaselineFile{
		Baselines: map[string]BaselineEntry{
			"KYC/Submit": {AvgMs: 700, P95Ms: 1000, SuccessRate: 95.0},
		},
	}
	current := map[string]BaselineEntry{
		"KYC/Submit": {AvgMs: 700, P95Ms: 1000, SuccessRate: 80.0},
	}
	report := Compare(bf, current, DefaultPolicy())
	if !report.HasRegression() {
		t.Fatalf("expected success-rate regression")
	}
}

func TestDefaultBaselinesValid(t *testing.T) {
	bf, err := LoadBaselines(filepath.Join(".", "baselines.json"))
	if err != nil {
		t.Fatalf("load baselines: %v", err)
	}
	if len(bf.Baselines) == 0 {
		t.Fatal("baselines.json is empty")
	}
	for name, e := range bf.Baselines {
		if e.P95Ms <= 0 {
			t.Errorf("%s: p95_ms must be > 0, got %v", name, e.P95Ms)
		}
		if e.SuccessRate < 0 || e.SuccessRate > 100 {
			t.Errorf("%s: success_rate %v out of range", name, e.SuccessRate)
		}
		if e.P50Ms > e.P95Ms || e.P95Ms > e.P99Ms {
			t.Errorf("%s: latency percentiles not monotonic (p50 %v, p95 %v, p99 %v)", name, e.P50Ms, e.P95Ms, e.P99Ms)
		}
	}
}

// TestCompareToBaselines is the regression gate invoked by the nightly
// performance pipeline. It is a no-op unless PERF_OUTPUT_FILE points at a
// captured load-test log, so it never affects a normal `go test` run.
func TestCompareToBaselines(t *testing.T) {
	output := os.Getenv("PERF_OUTPUT_FILE")
	if output == "" {
		t.Skip("PERF_OUTPUT_FILE not set; skipping regression comparison")
	}
	baselines := os.Getenv("PERF_BASELINES_FILE")
	if baselines == "" {
		baselines = filepath.Join(".", "baselines.json")
	}

	report, err := CompareFiles(baselines, output, DefaultPolicy())
	if err != nil {
		t.Fatalf("compare: %v", err)
	}

	if path := os.Getenv("PERF_REPORT_FILE"); path != "" {
		if err := os.WriteFile(path, []byte(report.Markdown()), 0o644); err != nil {
			t.Fatalf("write report: %v", err)
		}
	}
	t.Logf("\n%s", report.Markdown())

	if report.HasRegression() {
		t.Errorf("performance regressions detected: %d", report.Regressions)
	}
}
