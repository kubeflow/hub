package skillcatalog

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"k8s.io/apimachinery/pkg/util/yaml"
)

// Length limits from the Agent Skills specification
// (https://agentskills.io/specification). Violations warn rather than skip, per
// the lenient validation policy.
const (
	maxSkillNameLength          = 64
	maxSkillDescriptionLength   = 1024
	maxSkillCompatibilityLength = 500
	maxSkillBodyLines           = 500
)

// utf8BOM is the UTF-8 byte-order mark, stripped from the start of a file if present.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Skip errors: a SKILL.md that returns one of these is logged and not indexed.
var (
	// ErrNoFrontmatter indicates the file has no `---`-delimited YAML frontmatter.
	ErrNoFrontmatter = errors.New("SKILL.md has no YAML frontmatter")
	// ErrInvalidFrontmatter indicates the frontmatter block is not valid YAML.
	ErrInvalidFrontmatter = errors.New("SKILL.md frontmatter is not valid YAML")
	// ErrMissingDescription indicates the required `description` field is absent.
	ErrMissingDescription = errors.New("SKILL.md frontmatter is missing required 'description'")
)

// ParsedSkill is the result of parsing a SKILL.md file per the Agent Skills
// specification. Metadata is surfaced downstream as customProperties.
type ParsedSkill struct {
	Name          string
	Description   string
	License       string
	Compatibility string
	AllowedTools  []string
	Metadata      map[string]any
	Body          string
	BodyLineCount int
	// Warnings are non-fatal issues; a skill with warnings is still indexed but
	// its source is marked partially-available.
	Warnings []string
}

// skillFrontmatter is the YAML frontmatter schema. Parsing is lenient: unknown
// keys are ignored (not strict) so authors can carry extra fields.
type skillFrontmatter struct {
	Name          string `json:"name,omitempty"`
	Description   string `json:"description,omitempty"`
	License       string `json:"license,omitempty"`
	Compatibility string `json:"compatibility,omitempty"`
	// AllowedTools is a space-separated string per the spec, e.g.
	// "Bash(git:*) Read". A YAML list is also accepted for leniency, so this is
	// decoded as `any` and normalized by parseAllowedTools.
	AllowedTools any            `json:"allowed-tools,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// ParseSkillMD parses SKILL.md content leniently per the Agent Skills spec.
// expectedName is the skill's directory name, used for the name-match warning.
//
// It returns a non-nil error only for the skip cases (no/invalid frontmatter,
// missing description); a successful parse may still carry Warnings.
func ParseSkillMD(content []byte, expectedName string) (*ParsedSkill, error) {
	fmText, body, err := splitFrontmatter(content)
	if err != nil {
		return nil, err
	}

	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(fmText), &fm); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFrontmatter, err)
	}

	if strings.TrimSpace(fm.Description) == "" {
		return nil, ErrMissingDescription
	}

	skill := &ParsedSkill{
		Name:          fm.Name,
		Description:   fm.Description,
		License:       fm.License,
		Compatibility: fm.Compatibility,
		AllowedTools:  parseAllowedTools(fm.AllowedTools),
		Metadata:      fm.Metadata,
		Body:          body,
		BodyLineCount: countLines(body),
	}

	// Name (lenient: warn, never skip).
	switch {
	case fm.Name == "":
		skill.Name = expectedName
		skill.warnf("frontmatter is missing 'name'; using directory name %q", expectedName)
	case fm.Name != expectedName:
		skill.warnf("frontmatter name %q does not match directory %q", fm.Name, expectedName)
	}

	// Length limits from the spec (lenient: warn, never skip).
	if n := utf8.RuneCountInString(skill.Name); n > maxSkillNameLength {
		skill.warnf("name is %d characters, which exceeds the maximum of %d", n, maxSkillNameLength)
	}
	if n := utf8.RuneCountInString(skill.Description); n > maxSkillDescriptionLength {
		skill.warnf("description is %d characters, which exceeds the maximum of %d", n, maxSkillDescriptionLength)
	}
	if n := utf8.RuneCountInString(skill.Compatibility); n > maxSkillCompatibilityLength {
		skill.warnf("compatibility is %d characters, which exceeds the maximum of %d", n, maxSkillCompatibilityLength)
	}
	if skill.BodyLineCount > maxSkillBodyLines {
		skill.warnf("body is %d lines, which exceeds the recommended maximum of %d", skill.BodyLineCount, maxSkillBodyLines)
	}

	return skill, nil
}

// warnf appends a formatted non-fatal warning.
func (s *ParsedSkill) warnf(format string, args ...any) {
	s.Warnings = append(s.Warnings, fmt.Sprintf(format, args...))
}

// parseAllowedTools normalizes the frontmatter allowed-tools value into a slice.
// The spec defines it as a space-separated string (e.g. "Bash(git:*) Read"); a
// YAML list of strings is also accepted for leniency. Anything else yields nil.
func parseAllowedTools(v any) []string {
	switch t := v.(type) {
	case string:
		return strings.Fields(t)
	case []any:
		tools := make([]string, 0, len(t))
		for _, e := range t {
			if s, ok := e.(string); ok && strings.TrimSpace(s) != "" {
				tools = append(tools, s)
			}
		}
		if len(tools) == 0 {
			return nil
		}
		return tools
	default:
		return nil
	}
}

// splitFrontmatter separates the `---`-delimited YAML frontmatter from the body.
// Line endings are normalized to LF and a leading UTF-8 BOM is stripped.
func splitFrontmatter(content []byte) (frontmatter string, body string, err error) {
	s := strings.ReplaceAll(string(bytes.TrimPrefix(content, utf8BOM)), "\r\n", "\n")

	lines := strings.Split(s, "\n")
	if len(lines) == 0 || !isFrontmatterFence(lines[0]) {
		return "", "", ErrNoFrontmatter
	}

	closing := -1
	for i := 1; i < len(lines); i++ {
		if isFrontmatterFence(lines[i]) {
			closing = i
			break
		}
	}
	if closing == -1 {
		return "", "", fmt.Errorf("%w: frontmatter opened with '---' but has no closing delimiter", ErrInvalidFrontmatter)
	}

	frontmatter = strings.Join(lines[1:closing], "\n")
	body = strings.Trim(strings.Join(lines[closing+1:], "\n"), "\n")
	return frontmatter, body, nil
}

// isFrontmatterFence reports whether a line is a `---` frontmatter delimiter.
// The dashes must start at column 0: leading whitespace is significant, so an
// indented `---` inside a YAML block scalar is not mistaken for the fence.
// Trailing whitespace (and a stray CR) is tolerated.
func isFrontmatterFence(line string) bool {
	return strings.TrimRight(line, " \t\r") == "---"
}

// countLines returns the number of lines in the body, ignoring surrounding blank
// lines. An empty body counts as zero.
func countLines(body string) int {
	trimmed := strings.Trim(body, "\n")
	if trimmed == "" {
		return 0
	}
	return strings.Count(trimmed, "\n") + 1
}
