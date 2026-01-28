package metadata

import (
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractDocumentLeadingH1(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "..")
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}

	filename = "testdata/header.md"

	markdown, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	actual := ExtractDocumentLeadingH1(markdown)

	assert.Equal(t, "a", actual)
}

func TestSetTitleFromFilename(t *testing.T) {
	t.Run("set title from filename", func(t *testing.T) {
		meta := &Meta{Title: ""}
		setTitleFromFilename(meta, "/path/to/test.md")
		assert.Equal(t, "Test", meta.Title)
	})

	t.Run("replace underscores with spaces", func(t *testing.T) {
		meta := &Meta{Title: ""}
		setTitleFromFilename(meta, "/path/to/test_with_underscores.md")
		assert.Equal(t, "Test With Underscores", meta.Title)
	})

	t.Run("replace dashes with spaces", func(t *testing.T) {
		meta := &Meta{Title: ""}
		setTitleFromFilename(meta, "/path/to/test-with-dashes.md")
		assert.Equal(t, "Test With Dashes", meta.Title)
	})

	t.Run("mixed underscores and dashes", func(t *testing.T) {
		meta := &Meta{Title: ""}
		setTitleFromFilename(meta, "/path/to/test_with-mixed_separators.md")
		assert.Equal(t, "Test With Mixed Separators", meta.Title)
	})

	t.Run("already title cased", func(t *testing.T) {
		meta := &Meta{Title: ""}
		setTitleFromFilename(meta, "/path/to/Already-Title-Cased.md")
		assert.Equal(t, "Already Title Cased", meta.Title)
	})
}

func TestExtractMeta_YAML_Complex(t *testing.T) {
	data := []byte(`---
Confluence:
  Space: VICTRIAL
  Label:
    - Label1
    - Label2
  Attachment: single.png
  Parent: [Parent1, Parent2]
Other: ignore
---
Content`)

	meta, content, err := ExtractMeta(data, "", false, false, "", nil, false)
	assert.NoError(t, err)
	assert.NotNil(t, meta)
	assert.Equal(t, "VICTRIAL", meta.Space)
	assert.Equal(t, []string{"Label1", "Label2"}, meta.Labels)
	assert.Equal(t, []string{"single.png"}, meta.Attachments)
	assert.Equal(t, []string{"Parent1", "Parent2"}, meta.Parents)
	assert.Equal(t, []byte("Content"), content)
}

func TestExtractMeta_HTML_Complex(t *testing.T) {
	data := []byte(`<!-- Space: VICTRIAL -->
<!-- Label: Label1 -->
<!-- Label: Label2 -->
<!-- Attachment: single.png -->
<!-- Parent: Parent1 -->
<!-- Parent: Parent2 -->
Content`)

	meta, content, err := ExtractMeta(data, "", false, false, "", nil, false)
	assert.NoError(t, err)
	assert.NotNil(t, meta)
	assert.Equal(t, "VICTRIAL", meta.Space)
	assert.Equal(t, []string{"Label1", "Label2"}, meta.Labels)
	assert.Equal(t, []string{"single.png"}, meta.Attachments)
	assert.Equal(t, []string{"Parent1", "Parent2"}, meta.Parents)
	assert.Equal(t, []byte("Content"), content)
}

func TestExtractMeta_Macro(t *testing.T) {
	data := []byte(`<!-- Space: VICTRIAL -->
<!-- Macro: test -->
Content`)

	meta, content, err := ExtractMeta(data, "", false, false, "", nil, false)
	assert.NoError(t, err)
	assert.NotNil(t, meta)
	assert.Equal(t, "VICTRIAL", meta.Space)
	// Macro should be part of the content
	assert.Equal(t, []byte("<!-- Macro: test -->\nContent"), content)
}

func TestExtractMeta_ContentAppearance(t *testing.T) {
	data := []byte(`---
Confluence:
  content-appearance: fixed
---
Content`)

	meta, _, err := ExtractMeta(data, "", false, false, "", nil, false)
	assert.NoError(t, err)
	assert.NotNil(t, meta)
	assert.Equal(t, FixedContentAppearance, meta.ContentAppearance)
}
