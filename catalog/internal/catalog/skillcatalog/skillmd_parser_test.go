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
	assert.Equal(t, "Example Org", skill.Metadata["author"])
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

func TestParseSkillMD_MissingNameUsesDirectory(t *testing.T) {
	content := `---
description: A skill with no name.
---
Body.
`
	skill, err := ParseSkillMD([]byte(content), "fallback-name")
	require.NoError(t, err)
	require.NotNil(t, skill)
	assert.Equal(t, "fallback-name", skill.Name)
	assert.True(t, hasWarning(skill.Warnings, "name"), "expected a missing-name warning, got %v", skill.Warnings)
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
	assert.Equal(t, "Jane", skill.Metadata["author"])
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

func hasWarning(warnings []string, substr string) bool {
	for _, w := range warnings {
		if strings.Contains(w, substr) {
			return true
		}
	}
	return false
}
