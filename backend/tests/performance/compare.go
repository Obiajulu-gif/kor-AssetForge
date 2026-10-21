// Package performance contains load and performance tests for the kor-AssetForge API.
//
// This file provides the baseline / regression-detection utilities used by the
// nightly performance pipeline (.github/workflows/performance.yml). The load
// tests log a metrics block per scenario; ParseLoadOutput turns that output
// into structured metrics, LoadBaselines reads the recorded baselines, and
// Compare classifies regressions against a Policy.
package performance

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// BaselineEntry is the set of metrics recorded for a single scenario.
type BaselineEntry struct {
	AvgMs       float64 `json:"avg_ms"`
	P50Ms       float64 `json:"p50_ms"`
	P95Ms       float64 `json:"p95_ms"`
	P99Ms       float64 `json:"p99_ms"`
	SuccessRate float64 `json:"success_rate"`
	Total       int64   `json:"total,omitempty"`
}

// BaselineFile is the on-disk format of baselines.json.
type BaselineFile struct {
	Version   int                      `json:"version"`
	UpdatedAt string                   `json:"updated_at"`
	Note      string                   `json:"note,omitempty"`
	Baselines map[string]BaselineEntry `json:"baselines"`
}

// Policy controls when a measurement is considered a regression.
type Policy struct {
	// P95RegressionFactor is the multiplier applied to the baseline P95; a
	// current P95 above baseline*factor is a regression. 1.2 == +20%.
	P95RegressionFactor float64 `json:"p95_regression_factor"`
	// AvgRegressionFactor is the same idea for average latency.
	AvgRegressionFactor float64 `json:"avg_regression_factor"`
	// MinSuccessRate is the absolute floor; anything below is a regression.
	MinSuccessRate float64 `json:"min_success_rate"`
	// AllowedSuccessRateDrop is the number of percentage points a scenario may
	// drop below its baseline success rate before being flagged.
	AllowedSuccessRateDrop float64 `json:"allowed_success_rate_drop"`
}

// DefaultPolicy returns the policy used by the nightly pipeline.
func DefaultPolicy() Policy {
	return Policy{
		P95RegressionFactor:    1.20,
		AvgRegressionFactor:    1.25,
		MinSuccessRate:         90.0,
		AllowedSuccessRateDrop: 1.0,
	}
}

// Comparison is the result of comparing one scenario to its baseline.
type Comparison struct {
	Scenario   string        `json:"scenario"`
	Baseline   BaselineEntry `json:"baseline"`
	Current    BaselineEntry `json:"current"`
	Regression bool          `json:"regression"`
	Reasons    []string      `json:"reasons,omitempty"`
}

// Report aggregates every comparison plus bookkeeping about missing data.
type Report struct {
	GeneratedAt      string       `json:"generated_at"`
	Policy           Policy       `json:"policy"`
	Comparisons      []Comparison `json:"comparisons"`
	MissingBaselines []string     `json:"missing_baselines,omitempty"`
	MissingResults   []string     `json:"missing_results,omitempty"`
	Regressions      int          `json:"regressions"`
}

// HasRegression reports whether any scenario regressed.
func (r Report) HasRegression() bool { return r.Regressions > 0 }

// LoadBaselines reads and decodes a baselines.json file.
func LoadBaselines(path string) (*BaselineFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read baselines: %w", err)
	}
	var bf BaselineFile
	if err := json.Unmarshal(data, &bf); err != nil {
		return nil, fmt.Errorf("decode baselines: %w", err)
	}
	if bf.Baselines == nil {
		bf.Baselines = map[string]BaselineEntry{}
	}
	return &bf, nil
}

var (
	reScenario = regexp.MustCompile(`===\s*(.+?)\s*===`)
	reMetric   = regexp.MustCompile(`(Success rate|Avg latency|P50 latency|P95 latency|P99 latency|Total requests)\s*:\s*([0-9.]+)`)
)

// ParseLoadOutput parses the verbose `go test` output produced by the load
// tests into per-scenario metrics. Unknown lines are ignored, so it is safe to
// feed the entire test log.
func ParseLoadOutput(r io.Reader) (map[string]BaselineEntry, error) {
	results := map[string]BaselineEntry{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	current := ""
	for scanner.Scan() {
		line := scanner.Text()

		if m := reScenario.FindStringSubmatch(line); m != nil {
			current = strings.TrimSpace(m[1])
			if _, ok := results[current]; !ok {
				results[current] = BaselineEntry{}
			}
			continue
		}
		if current == "" {
			continue
		}

		m := reMetric.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		value, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			continue
		}
		entry := results[current]
		switch m[1] {
		case "Success rate":
			entry.SuccessRate = value
		case "Avg latency":
			entry.AvgMs = value
		case "P50 latency":
			entry.P50Ms = value
		case "P95 latency":
			entry.P95Ms = value
		case "P99 latency":
			entry.P99Ms = value
		case "Total requests":
			entry.Total = int64(value)
		}
		results[current] = entry
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan output: %w", err)
	}
	// Drop scenario headers that never accumulated any metrics.
	for name, entry := range results {
		if entry == (BaselineEntry{}) {
			delete(results, name)
		}
	}
	return results, nil
}

// Compare classifies each current measurement against the baselines.
func Compare(bf *BaselineFile, current map[string]BaselineEntry, policy Policy) Report {
	report := Report{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Policy:      policy,
	}

	names := make([]string, 0, len(bf.Baselines))
	for name := range bf.Baselines {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		base := bf.Baselines[name]
		cur, ok := current[name]
		if !ok {
			report.MissingResults = append(report.MissingResults, name)
			continue
		}
		cmp := Comparison{Scenario: name, Baseline: base, Current: cur}
		cmp.Reasons = regressionReasons(base, cur, policy)
		cmp.Regression = len(cmp.Reasons) > 0
		if cmp.Regression {
			report.Regressions++
		}
		report.Comparisons = append(report.Comparisons, cmp)
	}

	for name := range current {
		if _, ok := bf.Baselines[name]; !ok {
			report.MissingBaselines = append(report.MissingBaselines, name)
		}
	}
	sort.Strings(report.MissingBaselines)

	return report
}

func regressionReasons(base, cur BaselineEntry, policy Policy) []string {
	var reasons []string

	if policy.P95RegressionFactor > 0 && base.P95Ms > 0 {
		limit := base.P95Ms * policy.P95RegressionFactor
		if cur.P95Ms > limit {
			reasons = append(reasons, fmt.Sprintf(
				"P95 latency %.1f ms exceeds %.1f ms (baseline %.1f ms x %.2f)",
				cur.P95Ms, limit, base.P95Ms, policy.P95RegressionFactor))
		}
	}
	if policy.AvgRegressionFactor > 0 && base.AvgMs > 0 {
		limit := base.AvgMs * policy.AvgRegressionFactor
		if cur.AvgMs > limit {
			reasons = append(reasons, fmt.Sprintf(
				"avg latency %.1f ms exceeds %.1f ms (baseline %.1f ms x %.2f)",
				cur.AvgMs, limit, base.AvgMs, policy.AvgRegressionFactor))
		}
	}
	if policy.MinSuccessRate > 0 && cur.SuccessRate < policy.MinSuccessRate {
		reasons = append(reasons, fmt.Sprintf(
			"success rate %.2f%% below absolute minimum %.2f%%",
			cur.SuccessRate, policy.MinSuccessRate))
	}
	if base.SuccessRate > 0 && policy.AllowedSuccessRateDrop >= 0 {
		floor := base.SuccessRate - policy.AllowedSuccessRateDrop
		if cur.SuccessRate < floor {
			reasons = append(reasons, fmt.Sprintf(
				"success rate %.2f%% dropped more than %.2f points below baseline %.2f%%",
				cur.SuccessRate, policy.AllowedSuccessRateDrop, base.SuccessRate))
		}
	}
	return reasons
}

// CompareFiles loads a baselines file and a captured test output file, then
// returns the comparison report.
func CompareFiles(baselinesPath, outputPath string, policy Policy) (Report, error) {
	bf, err := LoadBaselines(baselinesPath)
	if err != nil {
		return Report{}, err
	}
	f, err := os.Open(outputPath)
	if err != nil {
		return Report{}, fmt.Errorf("open output: %w", err)
	}
	defer f.Close()

	current, err := ParseLoadOutput(f)
	if err != nil {
		return Report{}, err
	}
	return Compare(bf, current, policy), nil
}

// Markdown renders the report as a GitHub-flavoured Markdown summary suitable
// for the CI dashboard ($GITHUB_STEP_SUMMARY).
func (r Report) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Performance Regression Report\n\n")
	fmt.Fprintf(&b, "_Generated %s_\n\n", r.GeneratedAt)

	if r.HasRegression() {
		fmt.Fprintf(&b, "> **%d regression(s) detected.**\n\n", r.Regressions)
	} else {
		fmt.Fprintf(&b, "> No regressions detected. :white_check_mark:\n\n")
	}

	fmt.Fprintf(&b, "Policy: P95 x%.2f, avg x%.2f, min success %.2f%%, max success drop %.2f pts\n\n",
		r.Policy.P95RegressionFactor, r.Policy.AvgRegressionFactor,
		r.Policy.MinSuccessRate, r.Policy.AllowedSuccessRateDrop)

	b.WriteString("| Scenario | P95 (base → cur) | Avg (base → cur) | Success (base → cur) | Status |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, c := range r.Comparisons {
		status := "OK"
		if c.Regression {
			status = "**REGRESSION**"
		}
		fmt.Fprintf(&b, "| %s | %.1f → %.1f | %.1f → %.1f | %.2f%% → %.2f%% | %s |\n",
			c.Scenario, c.Baseline.P95Ms, c.Current.P95Ms,
			c.Baseline.AvgMs, c.Current.AvgMs,
			c.Baseline.SuccessRate, c.Current.SuccessRate, status)
	}

	if len(r.Comparisons) > 0 {
		regressed := false
		for _, c := range r.Comparisons {
			if c.Regression {
				if !regressed {
					b.WriteString("\n### Regression details\n\n")
					regressed = true
				}
				fmt.Fprintf(&b, "- **%s**: %s\n", c.Scenario, strings.Join(c.Reasons, "; "))
			}
		}
	}
	if len(r.MissingResults) > 0 {
		fmt.Fprintf(&b, "\n### Scenarios with no result this run\n\n%s\n", strings.Join(r.MissingResults, ", "))
	}
	if len(r.MissingBaselines) > 0 {
		fmt.Fprintf(&b, "\n### Scenarios without a baseline (update baselines.json)\n\n%s\n", strings.Join(r.MissingBaselines, ", "))
	}
	return b.String()
}
