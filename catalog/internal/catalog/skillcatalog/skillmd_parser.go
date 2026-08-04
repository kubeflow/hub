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
	Author        string
	Compatibility string
	AllowedTools  []string
	Metadata      map[string]any
	Body          string
	BodyLineCount int
	// Warnings are non-fatal issues; a skill with warnings is still indexed but
	// its source is marked partially-available.
	Warnings []string
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

	// Decode into a generic map rather than a typed struct so that a single
	// field with the wrong type (e.g. metadata written as a string) is tolerated
	// — warned and ignored — instead of failing the whole frontmatter. Only YAML
	// that is malformed or not a mapping is fatal.
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(fmText), &raw); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFrontmatter, err)
	}

	skill := &ParsedSkill{
		Body:          body,
		BodyLineCount: countLines(body),
	}

	// description is required; missing, empty, or non-string skips the skill.
	skill.Description = skill.stringField(raw, "description")
	if strings.TrimSpace(skill.Description) == "" {
		return nil, ErrMissingDescription
	}

	skill.Name = skill.stringField(raw, "name")
	skill.License = skill.stringField(raw, "license")
	skill.Compatibility = skill.stringField(raw, "compatibility")
	skill.AllowedTools = parseAllowedTools(raw["allowed-tools"])
	skill.Metadata = skill.mapField(raw, "metadata")
	skill.Author = skill.resolveAuthor(raw)

	// Name (lenient: warn, never skip).
	switch {
	case skill.Name == "":
		skill.Name = expectedName
		skill.warnf("frontmatter is missing 'name'; using directory name %q", expectedName)
	case skill.Name != expectedName:
		skill.warnf("frontmatter name %q does not match directory %q", skill.Name, expectedName)
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

// resolveAuthor determines the skill's author. It prefers a top-level `author`
// frontmatter field (like license); when absent it falls back to the spec's
// canonical metadata.author. Either way `author` is removed from the metadata map
// so it surfaces only as the dedicated Author field, not also as a custom property.
func (s *ParsedSkill) resolveAuthor(raw map[string]any) string {
	author, _ := raw["author"].(string)
	if s.Metadata != nil {
		if mdAuthor, ok := s.Metadata["author"]; ok {
			if str, isStr := mdAuthor.(string); isStr && strings.TrimSpace(author) == "" {
				author = str
			}
			// The "author" metadata key is reserved for the dedicated Author field;
			// remove it (whatever its type) so it never also surfaces as a custom
			// property.
			delete(s.Metadata, "author")
		}
	}
	return author
}

// warnf appends a formatted non-fatal warning.
func (s *ParsedSkill) warnf(format string, args ...any) {
	s.Warnings = append(s.Warnings, fmt.Sprintf(format, args...))
}

// stringField reads a string frontmatter field. An absent field yields ""; a
// present field of the wrong type is warned and ignored (returns "") rather than
// failing the whole parse.
func (s *ParsedSkill) stringField(raw map[string]any, key string) string {
	v, ok := raw[key]
	if !ok || v == nil {
		return ""
	}
	str, ok := v.(string)
	if !ok {
		s.warnf("frontmatter field %q is not a string and was ignored", key)
		return ""
	}
	return str
}

// mapField reads a map frontmatter field. An absent field yields nil; a present
// field of the wrong type is warned and ignored (returns nil) rather than
// failing the whole parse.
func (s *ParsedSkill) mapField(raw map[string]any, key string) map[string]any {
	v, ok := raw[key]
	if !ok || v == nil {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		s.warnf("frontmatter field %q is not a map and was ignored", key)
		return nil
	}
	return m
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
