package runtime

import (
	"fmt"
	"sort"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
)

func splitLines(data string) []string {
	data = strings.ReplaceAll(data, "\r\n", "\n")
	data = strings.ReplaceAll(data, "\r", "\n")
	return strings.Split(data, "\n")
}
func buildLineWindow(lines []string, index int) lineWindow {
	beforeStart := index - 3
	if beforeStart < 0 {
		beforeStart = 0
	}

	for beforeStart < index {
		trimmed := strings.TrimSpace(stripLineComment(lines[beforeStart]))
		if trimmed == "" || isStatementBoundary(trimmed) {
			beforeStart++
			continue
		}

		break
	}

	afterEnd := index + 1
	for afterEnd < len(lines) && afterEnd < index+7 {
		afterEnd++

		trimmed := strings.TrimSpace(stripLineComment(lines[afterEnd-1]))
		if isStatementBoundary(trimmed) {
			break
		}
	}

	before := append([]string(nil), lines[beforeStart:index]...)
	after := append([]string(nil), lines[index+1:afterEnd]...)

	return lineWindow{
		Before: before,
		Line:   lines[index],
		After:  after,
	}
}
func isStatementBoundary(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return true
	}

	return strings.HasSuffix(line, ";") ||
		strings.HasSuffix(line, "};") ||
		strings.HasSuffix(line, "});") ||
		strings.HasSuffix(line, "})") ||
		line == "}"
}
func rawWindowText(lines []string, index int, before int, after int) string {
	start := index - before
	if start < 0 {
		start = 0
	}

	end := index + after + 1
	if end > len(lines) {
		end = len(lines)
	}

	var b strings.Builder
	for _, line := range lines[start:end] {
		b.WriteString(stripLineComment(line))
		b.WriteByte('\n')
	}

	return b.String()
}
func evidenceSnippet(lines []string, index int, before int, after int) string {
	start := index - before
	if start < 0 {
		start = 0
	}

	end := index + after + 1
	if end > len(lines) {
		end = len(lines)
	}

	var b strings.Builder
	for i := start; i < end; i++ {
		prefix := "  "
		if i == index {
			prefix = "> "
		}

		fmt.Fprintf(&b, "%s%4d | %s\n", prefix, i+1, strings.TrimRight(lines[i], " \t"))
	}

	return strings.TrimRight(b.String(), "\n")
}
func addEvidence(builder *findingBuilder, evidence model.Evidence) {
	if builder == nil {
		return
	}

	builder.finding.Evidence = append(builder.finding.Evidence, evidence)
}
func stripLineComment(line string) string {
	index := strings.Index(line, "//")
	if index < 0 {
		return line
	}

	return line[:index]
}
func sortEvidence(items []model.Evidence) {
	sort.SliceStable(items, func(i int, j int) bool {
		if items[i].File != items[j].File {
			return items[i].File < items[j].File
		}

		if items[i].LineStart != items[j].LineStart {
			return items[i].LineStart < items[j].LineStart
		}

		return items[i].Message < items[j].Message
	})
}
