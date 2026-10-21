// Package version exposes API versioning metadata and migration guidance.
//
// The v1 API is deprecated and will be removed on SunsetDate. The
// MigrationGuide endpoint returns a machine-readable description of every
// breaking change between v1 and v2 so clients (and SDK generators) can
// migrate without reading the prose guide. The same information is mirrored in
// docs/API_MIGRATION_V1_TO_V2.md.
package version

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	// CurrentVersion is the latest, fully supported API version.
	CurrentVersion = "v2"

	// DeprecatedVersion is the legacy API version, still served with
	// Deprecation/Sunset headers until it is removed.
	DeprecatedVersion = "v1"

	// SunsetDate is the RFC1123 date on which v1 will be removed. It matches
	// the Sunset header emitted by middleware.DeprecationWarning.
	SunsetDate = "Sat, 31 Dec 2026 23:59:59 GMT"

	// DocsPath is the repository-relative location of the full migration guide.
	DocsPath = "docs/API_MIGRATION_V1_TO_V2.md"

	// SupportEmail is the contact address for API migration questions.
	SupportEmail = "api-support@assetforge.example"

	// IssuesURL is where API issues and migration blockers should be reported.
	IssuesURL = "https://github.com/parkerwinner/kor-AssetForge/issues"
)

// TimelinePhase describes a single milestone in the v1 deprecation schedule.
type TimelinePhase struct {
	Date      string `json:"date"`
	Milestone string `json:"milestone"`
}

// EndpointMapping maps a v1 endpoint to its v2 replacement.
type EndpointMapping struct {
	V1    string `json:"v1"`
	V2    string `json:"v2"`
	Notes string `json:"notes,omitempty"`
}

// SupportInfo lists the channels available to migrating clients.
type SupportInfo struct {
	Email     string `json:"email"`
	IssuesURL string `json:"issues_url"`
	DocsPath  string `json:"docs_path"`
}

// MigrationResponse is the payload returned by the migration guide endpoint.
type MigrationResponse struct {
	CurrentVersion         string            `json:"current_version"`
	DeprecatedVersion      string            `json:"deprecated_version"`
	SunsetDate             string            `json:"sunset_date"`
	Timeline               []TimelinePhase   `json:"timeline"`
	BreakingChanges        []string          `json:"breaking_changes"`
	EndpointMappings       []EndpointMapping `json:"endpoint_mapping"`
	RequestResponseChanges []string          `json:"request_response_changes"`
	MigrationSteps         []string          `json:"migration_steps"`
	CodeExamples           map[string]string `json:"code_examples"`
	Support                SupportInfo       `json:"support"`
}

// BuildMigrationResponse assembles the static migration description. It is
// exported so tests and other documentation tooling can reuse it.
func BuildMigrationResponse() MigrationResponse {
	return MigrationResponse{
		CurrentVersion:    CurrentVersion,
		DeprecatedVersion: DeprecatedVersion,
		SunsetDate:        SunsetDate,
		Timeline: []TimelinePhase{
			{Date: "2025-01-01", Milestone: "v2 GA; v1 marked deprecated with Deprecation/Sunset headers."},
			{Date: "2026-06-30", Milestone: "Final v1 feature freeze; only critical security fixes accepted."},
			{Date: "2026-12-31", Milestone: "v1 endpoints removed (sunset). Requests return 410 Gone."},
		},
		BreakingChanges: []string{
			"Base paths moved from /api/v1 to /api/v2.",
			"List responses now use a consistent envelope with data and api_version metadata.",
			"List pagination fields are standardised: page, page_size, total_count (v1 used limit and nested links).",
			"Single-resource responses are wrapped in a data object with api_version.",
			"Errors use a consistent { \"error\": string } shape across all v2 endpoints.",
			"v1 responses carry Deprecation and Sunset headers and will be removed at sunset.",
		},
		EndpointMappings: []EndpointMapping{
			{V1: "GET /api/v1/assets", V2: "GET /api/v2/assets", Notes: "limit -> page_size; response wrapped in data[]. "},
			{V1: "GET /api/v1/assets/:id", V2: "GET /api/v2/assets/:id", Notes: "Resource wrapped in a data object."},
			{V1: "GET /api/v1/health", V2: "GET /health", Notes: "Health endpoints are versionless."},
			{V1: "GET /api/v1/compliance/report", V2: "GET /api/v1/compliance/report", Notes: "No v2 replacement yet; endpoint remains on v1 until migrated."},
		},
		RequestResponseChanges: []string{
			"Query parameter limit is renamed page_size on v2 list endpoints (max 100, enforced globally).",
			"The X-API-Version header may be sent as v2; a conflicting path/header version returns 400.",
			"Use Accept: application/vnd.assetforge.v2+json to negotiate v2 during transition.",
			"Pagination links move from a links object (v1) to top-level page/page_size/total_count fields (v2).",
		},
		MigrationSteps: []string{
			"Replace /api/v1 path prefixes with /api/v2.",
			"Rename limit query parameters to page_size and read page_size/total_count from the response.",
			"Parse list results from the data field instead of the top-level array/links object.",
			"Send X-API-Version: v2 (or negotiate via Accept) during the transition window.",
			"Remove any reliance on resource fields returned bare in v1; v2 always wraps them in data.",
		},
		CodeExamples: map[string]string{
			"before_v1": "GET /api/v1/assets?page=1&limit=10\nAuthorization: Bearer <token>",
			"after_v2":  "GET /api/v2/assets?page=1&page_size=10\nAuthorization: Bearer <token>\nAccept: application/vnd.assetforge.v2+json",
			"v1_response": `{"limit":10,"page":1,"total":128,"data":[...]}`,
			"v2_response": `{"data":[...],"page":1,"page_size":10,"total_count":128,"api_version":"v2"}`,
		},
		Support: SupportInfo{
			Email:     SupportEmail,
			IssuesURL: IssuesURL,
			DocsPath:  DocsPath,
		},
	}
}

// MigrationGuide handles GET /api/version/migration-guide and returns the
// enhanced, machine-readable v1 -> v2 migration document.
//
// @Summary API v1 to v2 migration guide
// @Description Returns breaking changes, endpoint mappings, timeline and examples for migrating from API v1 to v2.
// @Tags versioning
// @Produce json
// @Success 200 {object} MigrationResponse
// @Router /api/version/migration-guide [get]
func MigrationGuide(c *gin.Context) {
	c.JSON(http.StatusOK, BuildMigrationResponse())
}
