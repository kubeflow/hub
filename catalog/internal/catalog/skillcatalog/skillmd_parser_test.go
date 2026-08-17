package skillcatalog

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validSkillMD = `---
name: deploy
description: Deploy an application to the cluster.
license: apache-2.0
compatibility: claude-code >= 1.0
allowed-tools:
  - Bash
  - Read
metadata:
  author: Example Org
  tier: internal
---
# Deploy

This skill deploys an application.

## Usage

Run the deploy command.
`

func TestParseSkillMD_Valid(t *testing.T) {
	skill, err := ParseSkillMD([]byte(validSkillMD), "deploy")
	require.NoError(t, err)
	require.NotNil(t, skill)

	assert.Equal(t, "deploy", skill.Name)
	assert.Equal(t, "Deploy an application to the cluster.", skill.Description)
	assert.Equal(t, "apache-2.0", skill.License)
	assert.Equal(t, "claude-code >= 1.0", skill.Compatibility)
	assert.Equal(t, []string{"Bash", "Read"}, skill.AllowedTools)
	assert.Equal(t, "Example Org", skill.Author, "metadata.author is promoted to the Author field")
	assert.NotContains(t, skill.Metadata, "author")
	assert.Equal(t, "internal", skill.Metadata["tier"])
	assert.Empty(t, skill.Warnings)

	assert.Contains(t, skill.Body, "# Deploy")
	assert.Contains(t, skill.Body, "Run the deploy command.")
	assert.NotContains(t, skill.Body, "name: deploy", "body must not include frontmatter")
	assert.Positive(t, skill.BodyLineCount)
}

func TestParseSkillMD_NameMismatchWarns(t *testing.T) {
	skill, err := ParseSkillMD([]byte(validSkillMD), "deployment")
	require.NoError(t, err)
	require.NotNil(t, skill)
	assert.Equal(t, "deploy", skill.Name, "frontmatter name is kept")
	assert.True(t, hasWarning(skill.Warnings, "does not match"),
		"expected a name-mismatch warning, got %v", skill.Warnings)
}

func TestParseSkillMD_MissingNameSkipped(t *testing.T) {
	content := `---
description: A skill with no name.
---
Body.
`
	skill, err := ParseSkillMD([]byte(content), "fallback-name")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMissingName)
	assert.Nil(t, skill)
}

func TestParseSkillMD_NameTooLongWarns(t *testing.T) {
	longName := strings.Repeat("a", 65)
	content := "---\nname: " + longName + "\ndescription: x\n---\nbody\n"
	skill, err := ParseSkillMD([]byte(content), longName)
	require.NoError(t, err)
	require.NotNil(t, skill)
	assert.True(t, hasWarning(skill.Warnings, "exceeds"), "expected a length warning, got %v", skill.Warnings)
}

func TestParseSkillMD_MissingDescriptionSkipped(t *testing.T) {
	content := `---
name: deploy
license: apache-2.0
---
Body without a description.
`
	skill, err := ParseSkillMD([]byte(content), "deploy")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMissingDescription)
	assert.Nil(t, skill)
}

func TestParseSkillMD_MalformedYAMLSkipped(t *testing.T) {
	content := `---
name: deploy
description: "unterminated
  : : bad
---
Body.
`
	skill, err := ParseSkillMD([]byte(content), "deploy")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidFrontmatter)
	assert.Nil(t, skill)
}

func TestParseSkillMD_NoFrontmatterSkipped(t *testing.T) {
	content := "# Just a markdown file\n\nNo frontmatter here.\n"
	skill, err := ParseSkillMD([]byte(content), "deploy")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoFrontmatter)
	assert.Nil(t, skill)
}

func TestParseSkillMD_LongBodyWarns(t *testing.T) {
	var b strings.Builder
	b.WriteString("---\nname: big\ndescription: A big skill.\n---\n")
	for i := 0; i < 600; i++ {
		b.WriteString("line\n")
	}
	skill, err := ParseSkillMD([]byte(b.String()), "big")
	require.NoError(t, err)
	require.NotNil(t, skill)
	assert.Equal(t, 600, skill.BodyLineCount)
	assert.True(t, hasWarning(skill.Warnings, "500"), "expected a body-length warning, got %v", skill.Warnings)
}

func TestParseSkillMD_TrimsSingleLineFields(t *testing.T) {
	// A description authored as a literal block scalar keeps a trailing newline in
	// YAML; single-line fields must not carry that (or stray whitespace) through.
	content := "---\n" +
		"name: deploy\n" +
		"description: |\n  Deploy the app.\n" +
		"license: apache-2.0   \n" +
		"author: |\n  Jane Doe\n" +
		"compatibility: claude-code\n" +
		"---\nbody\n"
	skill, err := ParseSkillMD([]byte(content), "deploy")
	require.NoError(t, err)
	require.NotNil(t, skill)

	assert.Equal(t, "Deploy the app.", skill.Description, "block-scalar description has no trailing newline")
	assert.Equal(t, "apache-2.0", skill.License, "trailing whitespace trimmed")
	assert.Equal(t, "Jane Doe", skill.Author, "block-scalar author trimmed")
	assert.Equal(t, "deploy", skill.Name)
	assert.Equal(t, "claude-code", skill.Compatibility)
}

func TestParseSkillMD_TopLevelAuthorField(t *testing.T) {
	// author at the frontmatter top level populates the Author field (like license).
	content := "---\nname: m\ndescription: d\nauthor: Jane Doe\n---\nbody\n"
	skill, err := ParseSkillMD([]byte(content), "m")
	require.NoError(t, err)
	require.NotNil(t, skill)
	assert.Equal(t, "Jane Doe", skill.Author)
}

func TestParseSkillMD_AuthorFromMetadataFallback(t *testing.T) {
	// The spec's canonical location is metadata.author; it populates Author when
	// there is no top-level author, and is removed from the metadata map so it is
	// not also surfaced as a custom property.
	content := "---\nname: m\ndescription: d\nmetadata:\n  author: Canonical\n  team: platform\n---\nbody\n"
	skill, err := ParseSkillMD([]byte(content), "m")
	require.NoError(t, err)
	require.NotNil(t, skill)
	assert.Equal(t, "Canonical", skill.Author)
	assert.NotContains(t, skill.Metadata, "author", "author is promoted to the field, not left in metadata")
	assert.Equal(t, "platform", skill.Metadata["team"], "other metadata is preserved")
}

func TestParseSkillMD_TopLevelAuthorWinsOverMetadata(t *testing.T) {
	content := "---\nname: m\ndescription: d\nauthor: Top Level\nmetadata:\n  author: Canonical\n---\nbody\n"
	skill, err := ParseSkillMD([]byte(content), "m")
	require.NoError(t, err)
	require.NotNil(t, skill)
	assert.Equal(t, "Top Level", skill.Author)
	assert.NotContains(t, skill.Metadata, "author")
}

func TestParseSkillMD_MetadataMap(t *testing.T) {
	content := `---
name: m
description: d
metadata:
  author: Jane
  count: 3
  nested:
    a: b
---
body
`
	skill, err := ParseSkillMD([]byte(content), "m")
	require.NoError(t, err)
	require.NotNil(t, skill)
	assert.Equal(t, "Jane", skill.Author, "metadata.author is promoted to the Author field")
	assert.NotContains(t, skill.Metadata, "author")
	assert.EqualValues(t, 3, skill.Metadata["count"])
	assert.NotNil(t, skill.Metadata["nested"])
}

func TestParseSkillMD_Unicode(t *testing.T) {
	content := `---
name: 日本語スキル
description: スキルの説明 — with émojis 🚀
---
# 見出し

本文です。
`
	skill, err := ParseSkillMD([]byte(content), "日本語スキル")
	require.NoError(t, err)
	require.NotNil(t, skill)
	assert.Equal(t, "日本語スキル", skill.Name)
	assert.Contains(t, skill.Description, "🚀")
	assert.Contains(t, skill.Body, "本文です。")
}

func TestParseSkillMD_CRLFLineEndings(t *testing.T) {
	content := "---\r\nname: crlf\r\ndescription: Windows line endings.\r\n---\r\n# Body\r\n\r\nText.\r\n"
	skill, err := ParseSkillMD([]byte(content), "crlf")
	require.NoError(t, err)
	require.NotNil(t, skill)
	assert.Equal(t, "crlf", skill.Name)
	assert.Equal(t, "Windows line endings.", skill.Description)
	assert.Contains(t, skill.Body, "# Body")
}

func TestParseSkillMD_NoClosingDelimiterSkipped(t *testing.T) {
	content := "---\nname: x\ndescription: y\nbody without closing fence\n"
	_, err := ParseSkillMD([]byte(content), "x")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidFrontmatter, "an opened-but-unclosed frontmatter is invalid, not absent")
}

func TestParseSkillMD_IndentedDashesInsideBlockScalarNotAFence(t *testing.T) {
	// The `---` inside the `description` block scalar is indented, so it must not
	// be treated as the closing fence — only the column-0 `---` closes frontmatter.
	content := `---
name: x
foo: bar
description: |
   blah
   ---
   blah
baz: qux
---
Body
`
	skill, err := ParseSkillMD([]byte(content), "x")
	require.NoError(t, err)
	require.NotNil(t, skill)
	assert.Contains(t, skill.Description, "---", "the indented --- belongs to the description")
	assert.Contains(t, skill.Description, "blah")
	assert.Equal(t, "Body", strings.TrimSpace(skill.Body), "frontmatter must not leak into the body")
}

func TestParseSkillMD_StripsLeadingBOM(t *testing.T) {
	// Some editors save UTF-8 files with a leading BOM; it must not defeat
	// frontmatter detection (Go's TrimSpace does not treat U+FEFF as whitespace).
	content := append([]byte{0xEF, 0xBB, 0xBF}, []byte(validSkillMD)...)
	skill, err := ParseSkillMD(content, "deploy")
	require.NoError(t, err)
	require.NotNil(t, skill)
	assert.Equal(t, "deploy", skill.Name)
	assert.Equal(t, "Deploy an application to the cluster.", skill.Description)
}

func TestParseSkillMD_AllowedToolsSpaceSeparatedString(t *testing.T) {
	// The spec defines allowed-tools as a space-separated string.
	content := "---\nname: t\ndescription: d\nallowed-tools: Bash(git:*) Bash(jq:*) Read\n---\nbody\n"
	skill, err := ParseSkillMD([]byte(content), "t")
	require.NoError(t, err)
	require.NotNil(t, skill)
	assert.Equal(t, []string{"Bash(git:*)", "Bash(jq:*)", "Read"}, skill.AllowedTools)
}

func TestParseSkillMD_DescriptionTooLongWarns(t *testing.T) {
	content := "---\nname: d\ndescription: " + strings.Repeat("x", 1025) + "\n---\nbody\n"
	skill, err := ParseSkillMD([]byte(content), "d")
	require.NoError(t, err)
	require.NotNil(t, skill)
	assert.True(t, hasWarning(skill.Warnings, "description"), "expected a description-length warning, got %v", skill.Warnings)
}

func TestParseSkillMD_CompatibilityTooLongWarns(t *testing.T) {
	content := "---\nname: c\ndescription: d\ncompatibility: " + strings.Repeat("y", 501) + "\n---\nbody\n"
	skill, err := ParseSkillMD([]byte(content), "c")
	require.NoError(t, err)
	require.NotNil(t, skill)
	assert.True(t, hasWarning(skill.Warnings, "compatibility"), "expected a compatibility-length warning, got %v", skill.Warnings)
}

func TestParseSkillMD_WrongTypeMetadataDoesNotSkipSkill(t *testing.T) {
	// metadata written as a string (not a map) must be ignored with a warning —
	// the rest of the frontmatter still parses. (The whole skill must not be lost.)
	content := `---
name: deploy
description: Deploy the app.
metadata: not-a-map
---
Body.
`
	skill, err := ParseSkillMD([]byte(content), "deploy")
	require.NoError(t, err)
	require.NotNil(t, skill)
	assert.Equal(t, "deploy", skill.Name)
	assert.Equal(t, "Deploy the app.", skill.Description)
	assert.Nil(t, skill.Metadata)
	assert.True(t, hasWarning(skill.Warnings, "metadata"), "expected a metadata type warning, got %v", skill.Warnings)
}

func TestParseSkillMD_NonStringNameSkipped(t *testing.T) {
	// name written as a number is not a string; stringField returns "" and the
	// skill is skipped as if name were absent.
	content := "---\nname: 123\ndescription: Valid description.\nlicense: apache-2.0\n---\nbody\n"
	skill, err := ParseSkillMD([]byte(content), "expected-dir")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMissingName)
	assert.Nil(t, skill)
}

func hasWarning(warnings []string, substr string) bool {
	for _, w := range warnings {
		if strings.Contains(w, substr) {
			return true
		}
	}
	return false
}
