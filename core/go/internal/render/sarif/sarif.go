package sarif

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	contracts "github.com/orurh/patchcourt/internal/diff/contract"
	depdiff "github.com/orurh/patchcourt/internal/diff/dep"
	findingdiff "github.com/orurh/patchcourt/internal/diff/finding"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/files"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

const (
	sarifVersion = "2.1.0"
	sarifSchema  = "https://json.schemastore.org/sarif-2.1.0.json"

	toolName = "PatchCourt"
)

type Log struct {
	Version string `json:"version"`
	Schema  string `json:"$schema,omitempty"`
	Runs    []Run  `json:"runs"`
}

type Run struct {
	Tool    Tool     `json:"tool"`
	Results []Result `json:"results,omitempty"`
}

type Tool struct {
	Driver Driver `json:"driver"`
}

type Driver struct {
	Name           string `json:"name"`
	InformationURI string `json:"informationUri,omitempty"`
	Rules          []Rule `json:"rules,omitempty"`
}

type Rule struct {
	ID               string         `json:"id"`
	Name             string         `json:"name,omitempty"`
	ShortDescription Message        `json:"shortDescription,omitempty"`
	FullDescription  Message        `json:"fullDescription,omitempty"`
	Help             Message        `json:"help,omitempty"`
	Properties       map[string]any `json:"properties,omitempty"`
}

type Result struct {
	RuleID     string         `json:"ruleId"`
	Level      string         `json:"level"`
	Message    Message        `json:"message"`
	Locations  []Location     `json:"locations,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
}

type Message struct {
	Text string `json:"text"`
}

type Location struct {
	PhysicalLocation PhysicalLocation `json:"physicalLocation"`
}

type PhysicalLocation struct {
	ArtifactLocation ArtifactLocation `json:"artifactLocation"`
	Region           *Region          `json:"region,omitempty"`
}

type ArtifactLocation struct {
	URI string `json:"uri"`
}

type Region struct {
	StartLine int      `json:"startLine,omitempty"`
	EndLine   int      `json:"endLine,omitempty"`
	Snippet   *Snippet `json:"snippet,omitempty"`
}

type Snippet struct {
	Text string `json:"text"`
}

func WriteReviewSARIFFile(path string, result reportmodel.ReviewResult) error {
	var buf bytes.Buffer

	if err := WriteReviewSARIF(&buf, result); err != nil {
		return err
	}

	if err := files.WriteFileAtomic(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write SARIF %s: %w", path, err)
	}

	return nil
}

func WriteReviewSARIF(w io.Writer, result reportmodel.ReviewResult) error {
	log := BuildReviewSARIF(result)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(log); err != nil {
		return fmt.Errorf("encode SARIF: %w", err)
	}

	return nil
}

func BuildReviewSARIF(result reportmodel.ReviewResult) Log {
	builder := newBuilder()

	builder.addFindingChanges(result.FindingChanges)
	builder.addContractChanges(result.ContractChanges)
	builder.addImpactItems(result.Impact.Worse)

	return Log{
		Version: sarifVersion,
		Schema:  sarifSchema,
		Runs: []Run{
			{
				Tool: Tool{
					Driver: Driver{
						Name:  toolName,
						Rules: builder.rules(),
					},
				},
				Results: builder.results,
			},
		},
	}
}

type builder struct {
	ruleByID map[string]Rule
	results  []Result
}

func newBuilder() *builder {
	return &builder{
		ruleByID: make(map[string]Rule),
		results:  make([]Result, 0),
	}
}

func (b *builder) addFindingChanges(changes []findingdiff.FindingChange) {
	for _, change := range changes {
		if change.Kind != findingdiff.FindingChangeKindAdded || change.After == nil {
			continue
		}

		finding := *change.After
		ruleID := findingRuleID(finding)
		message := finding.Title
		if message == "" {
			message = ruleID
		}

		b.addRule(Rule{
			ID:               ruleID,
			Name:             string(finding.Kind),
			ShortDescription: Message{Text: message},
			FullDescription:  Message{Text: nonEmpty(finding.Risk, message)},
			Help:             Message{Text: finding.Suggestion},
			Properties: map[string]any{
				"patchcourt.kind":       string(finding.Kind),
				"patchcourt.severity":   string(finding.Severity),
				"patchcourt.confidence": string(finding.Confidence),
				"patchcourt.source":     "finding_change",
			},
		})

		b.results = append(b.results, Result{
			RuleID:    ruleID,
			Level:     levelForSeverity(finding.Severity),
			Message:   Message{Text: findingMessage(finding)},
			Locations: evidenceLocations(finding.Evidence),
			Properties: compactProperties(map[string]any{
				"patchcourt.id":         finding.ID,
				"patchcourt.kind":       string(finding.Kind),
				"patchcourt.severity":   string(finding.Severity),
				"patchcourt.confidence": string(finding.Confidence),
				"patchcourt.impact":     "worse",
				"patchcourt.risk":       finding.Risk,
				"patchcourt.suggestion": finding.Suggestion,
			}),
		})
	}
}

func (b *builder) addContractChanges(changes []contracts.SymbolChange) {
	for _, change := range changes {
		if !isContractAlert(change) {
			continue
		}

		ruleID := "patchcourt.contract." + strings.ReplaceAll(string(change.Kind), "_", "-")
		title := contractTitle(change)

		b.addRule(Rule{
			ID:               ruleID,
			Name:             "contract change",
			ShortDescription: Message{Text: title},
			FullDescription:  Message{Text: "Patch changed a public contract symbol."},
			Help:             Message{Text: "Verify callers, implementations, compatibility, and related tests."},
			Properties: map[string]any{
				"patchcourt.kind":   "contract_change",
				"patchcourt.source": "contract_diff",
			},
		})

		b.results = append(b.results, Result{
			RuleID:    ruleID,
			Level:     levelForContractChange(change),
			Message:   Message{Text: title},
			Locations: contractLocations(change),
			Properties: compactProperties(map[string]any{
				"patchcourt.kind":       "contract_change",
				"patchcourt.change":     string(change.Kind),
				"patchcourt.symbol_key": change.SymbolKey,
				"patchcourt.impact":     "worse",
			}),
		})
	}
}

func (b *builder) addImpactItems(items []reportmodel.ReviewImpactItem) {
	for _, item := range items {
		if len(item.Evidence) == 0 {
			continue
		}

		if strings.HasPrefix(item.Kind, "contract_") {
			continue
		}

		ruleID := "patchcourt.impact." + sanitizeID(nonEmpty(item.Kind, "worse"))
		title := nonEmpty(item.Title, item.Kind)

		b.addRule(Rule{
			ID:               ruleID,
			Name:             item.Kind,
			ShortDescription: Message{Text: title},
			FullDescription:  Message{Text: nonEmpty(item.Risk, item.Detail, title)},
			Help:             Message{Text: item.Suggestion},
			Properties: map[string]any{
				"patchcourt.kind":   item.Kind,
				"patchcourt.source": "architecture_impact",
			},
		})

		b.results = append(b.results, Result{
			RuleID:    ruleID,
			Level:     levelForSeverity(model.Severity(item.Severity)),
			Message:   Message{Text: impactMessage(item)},
			Locations: evidenceLocations(item.Evidence),
			Properties: compactProperties(map[string]any{
				"patchcourt.id":         item.ID,
				"patchcourt.kind":       item.Kind,
				"patchcourt.severity":   item.Severity,
				"patchcourt.impact":     "worse",
				"patchcourt.detail":     item.Detail,
				"patchcourt.risk":       item.Risk,
				"patchcourt.suggestion": item.Suggestion,
			}),
		})
	}
}

func (b *builder) addRule(rule Rule) {
	if rule.ID == "" {
		return
	}

	if _, ok := b.ruleByID[rule.ID]; ok {
		return
	}

	b.ruleByID[rule.ID] = rule
}

func (b *builder) rules() []Rule {
	ids := make([]string, 0, len(b.ruleByID))
	for id := range b.ruleByID {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	rules := make([]Rule, 0, len(ids))
	for _, id := range ids {
		rules = append(rules, b.ruleByID[id])
	}

	return rules
}

func isContractAlert(change contracts.SymbolChange) bool {
	switch change.Kind {
	case contracts.ChangeKindRemoved, contracts.ChangeKindSignatureChanged, contracts.ChangeKindModifiersChanged:
		return true
	default:
		return false
	}
}

func findingRuleID(finding model.Finding) string {
	if finding.ID != "" {
		return finding.ID
	}

	if finding.Kind != "" {
		return "patchcourt.finding." + sanitizeID(string(finding.Kind))
	}

	return "patchcourt.finding"
}

func findingMessage(finding model.Finding) string {
	parts := make([]string, 0, 3)

	if finding.Title != "" {
		parts = append(parts, finding.Title)
	}
	if finding.Risk != "" {
		parts = append(parts, finding.Risk)
	}
	if finding.Suggestion != "" {
		parts = append(parts, "Suggestion: "+finding.Suggestion)
	}

	if len(parts) == 0 {
		return nonEmpty(finding.ID, "PatchCourt finding")
	}

	return strings.Join(parts, " ")
}

func contractTitle(change contracts.SymbolChange) string {
	switch change.Kind {
	case contracts.ChangeKindRemoved:
		return "Public contract removed: " + change.SymbolKey
	case contracts.ChangeKindSignatureChanged:
		return "Public contract signature changed: " + change.SymbolKey
	case contracts.ChangeKindModifiersChanged:
		return "Public contract modifiers changed: " + change.SymbolKey
	default:
		return "Public contract changed: " + change.SymbolKey
	}
}

func impactMessage(item reportmodel.ReviewImpactItem) string {
	parts := make([]string, 0, 4)

	if item.Title != "" {
		parts = append(parts, item.Title)
	}
	if item.Detail != "" {
		parts = append(parts, item.Detail)
	}
	if item.Risk != "" {
		parts = append(parts, item.Risk)
	}
	if item.Suggestion != "" {
		parts = append(parts, "Suggestion: "+item.Suggestion)
	}

	if len(parts) == 0 {
		return nonEmpty(item.ID, item.Kind, "PatchCourt architecture impact")
	}

	return strings.Join(parts, " ")
}

func evidenceLocations(evidence []model.Evidence) []Location {
	locations := make([]Location, 0, len(evidence))

	for _, item := range evidence {
		location := evidenceLocation(item)
		if location == nil {
			continue
		}

		locations = append(locations, *location)
	}

	return locations
}

func evidenceLocation(evidence model.Evidence) *Location {
	uri := nonEmpty(evidence.File, evidence.FromFile)
	if uri == "" {
		return nil
	}

	physical := PhysicalLocation{
		ArtifactLocation: ArtifactLocation{URI: normalizeURI(uri)},
	}

	if evidence.LineStart > 0 || evidence.LineEnd > 0 || evidence.Snippet != "" {
		region := &Region{
			StartLine: evidence.LineStart,
			EndLine:   evidence.LineEnd,
		}

		if evidence.Snippet != "" {
			region.Snippet = &Snippet{Text: evidence.Snippet}
		}

		physical.Region = region
	}

	return &Location{PhysicalLocation: physical}
}

func contractLocations(change contracts.SymbolChange) []Location {
	locations := make([]Location, 0, 2)

	if change.Before != nil {
		if location := symbolLocation(*change.Before); location != nil {
			locations = append(locations, *location)
		}
	}

	if change.After != nil {
		if location := symbolLocation(*change.After); location != nil {
			locations = append(locations, *location)
		}
	}

	return dedupeLocations(locations)
}

func symbolLocation(symbol model.SymbolModel) *Location {
	if symbol.File == "" {
		return nil
	}

	physical := PhysicalLocation{
		ArtifactLocation: ArtifactLocation{URI: normalizeURI(symbol.File)},
	}

	if symbol.Line > 0 {
		physical.Region = &Region{StartLine: symbol.Line}
	}

	return &Location{PhysicalLocation: physical}
}

func dedupeLocations(locations []Location) []Location {
	seen := make(map[string]struct{}, len(locations))
	result := make([]Location, 0, len(locations))

	for _, location := range locations {
		key := locationKey(location)
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		result = append(result, location)
	}

	return result
}

func locationKey(location Location) string {
	physical := location.PhysicalLocation
	line := 0
	if physical.Region != nil {
		line = physical.Region.StartLine
	}

	return fmt.Sprintf("%s:%d", physical.ArtifactLocation.URI, line)
}

func levelForSeverity(severity model.Severity) string {
	switch severity {
	case model.SeverityCritical, model.SeverityHigh:
		return "error"
	case model.SeverityMedium:
		return "warning"
	case model.SeverityLow:
		return "note"
	default:
		return "warning"
	}
}

func levelForContractChange(change contracts.SymbolChange) string {
	switch change.Kind {
	case contracts.ChangeKindRemoved, contracts.ChangeKindSignatureChanged:
		return "error"
	default:
		return "warning"
	}
}

func normalizeURI(uri string) string {
	return strings.ReplaceAll(uri, "\\", "/")
}

func sanitizeID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, ":", "-")

	var b strings.Builder
	lastDash := false

	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-':
			if !lastDash {
				b.WriteRune(r)
				lastDash = true
			}
		}
	}

	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "unknown"
	}

	return result
}

func compactProperties(properties map[string]any) map[string]any {
	result := make(map[string]any)

	for key, value := range properties {
		switch typed := value.(type) {
		case string:
			if typed == "" {
				continue
			}
		case nil:
			continue
		}

		result[key] = value
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func nonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}

var _ = depdiff.DependencyChange{}
