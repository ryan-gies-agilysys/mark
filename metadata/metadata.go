package metadata

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/reconquest/pkg/log"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

const (
	HeaderParent      = `Parent`
	HeaderSpace       = `Space`
	HeaderType        = `Type`
	HeaderTitle       = `Title`
	HeaderLayout      = `Layout`
	HeaderEmoji       = `Emoji`
	HeaderAttachment  = `Attachment`
	HeaderLabel       = `Label`
	HeaderInclude     = `Include`
	HeaderSidebar     = `Sidebar`
	ContentAppearance = `Content-Appearance`
)

type Meta struct {
	Parents           []string
	Space             string
	Type              string
	Title             string
	Layout            string
	Sidebar           string
	Emoji             string
	Attachments       []string
	Labels            []string
	ContentAppearance string
}

const (
	FullWidthContentAppearance = "full-width"
	FixedContentAppearance     = "fixed"
)

var (
	reHeaderPatternV2    = regexp.MustCompile(`<!--\s*([^:]+):\s*(.*)\s*-->`)
	reHeaderPatternMacro = regexp.MustCompile(`<!-- Macro: .*`)
	reYAMLFrontmatter    = regexp.MustCompile(`(?s)^---\r?\n(.*?)\r?\n---\r?\n?`)
)

func (meta *Meta) setHeader(key string, value interface{}) {
	if values, ok := value.([]interface{}); ok {
		for _, v := range values {
			meta.setHeader(key, v)
		}
		return
	}

	header := cases.Title(language.English).String(key)
	sValue := strings.TrimSpace(fmt.Sprint(value))

	switch header {
	case HeaderParent:
		meta.Parents = append(meta.Parents, sValue)

	case HeaderSpace:
		meta.Space = sValue

	case HeaderType:
		meta.Type = sValue

	case HeaderTitle:
		meta.Title = sValue

	case HeaderLayout:
		meta.Layout = sValue

	case HeaderSidebar:
		meta.Layout = "article"
		meta.Sidebar = sValue

	case HeaderEmoji:
		meta.Emoji = sValue

	case HeaderAttachment:
		meta.Attachments = append(meta.Attachments, sValue)

	case HeaderLabel:
		meta.Labels = append(meta.Labels, sValue)

	case HeaderInclude:
		// Includes are parsed by a different func

	case ContentAppearance:
		if sValue == FixedContentAppearance {
			meta.ContentAppearance = FixedContentAppearance
		} else {
			meta.ContentAppearance = FullWidthContentAppearance
		}

	default:
		log.Errorf(
			nil,
			`encountered unknown header %q`,
			header,
		)
	}
}

func ExtractMeta(data []byte, spaceFromCli string, titleFromH1 bool, titleFromFilename bool, filename string, parents []string, titleAppendGeneratedHash bool) (*Meta, []byte, error) {
	var (
		meta   *Meta
		offset int
	)

	if match := reYAMLFrontmatter.FindSubmatchIndex(data); match != nil {
		yamlData := data[match[2]:match[3]]
		offset = match[1]

		var raw map[string]interface{}
		err := yaml.Unmarshal(yamlData, &raw)
		if err != nil {
			return nil, nil, err
		}

		if meta == nil {
			meta = &Meta{}
			meta.Type = "page"                                  // Default if not specified
			meta.ContentAppearance = FullWidthContentAppearance // Default to full-width for backwards compatibility
		}

		// YAML map keys are not guaranteed to be in order, but for these headers it doesn't matter much
		// except for Parents/Labels/Attachments which we append.
		topLevel := "Confluence"
		for k, v := range raw {
			if strings.EqualFold(k, topLevel) {
				if confluence, ok := v.(map[string]interface{}); ok {
					for ck, cv := range confluence {
						meta.setHeader(ck, cv)
					}
				}
				break
			}
		}
	} else {
		scanner := bufio.NewScanner(bytes.NewBuffer(data))
		for scanner.Scan() {
			line := scanner.Text()

			if err := scanner.Err(); err != nil {
				return nil, nil, err
			}

			if reHeaderPatternMacro.MatchString(line) {
				break
			}

			matches := reHeaderPatternV2.FindStringSubmatch(line)
			if matches == nil {
				break
			}

			offset += len(line) + 1

			if meta == nil {
				meta = &Meta{}
				meta.Type = "page"                                  // Default if not specified
				meta.ContentAppearance = FullWidthContentAppearance // Default to full-width for backwards compatibility
			}

			meta.setHeader(matches[1], matches[2])
		}
	}

	if titleFromH1 || titleFromFilename || spaceFromCli != "" {
		if meta == nil {
			meta = &Meta{}
		}

		if meta.Type == "" {
			meta.Type = "page"
		}

		if meta.ContentAppearance == "" {
			meta.ContentAppearance = FullWidthContentAppearance // Default to full-width for backwards compatibility
		}

		if titleFromH1 && meta.Title == "" {
			meta.Title = ExtractDocumentLeadingH1(data)
		}
		if titleFromFilename && meta.Title == "" && filename != "" {
			setTitleFromFilename(meta, filename)
		}
		if spaceFromCli != "" && meta.Space == "" {
			meta.Space = spaceFromCli
		}
	}

	if meta == nil {
		return nil, data, nil
	}

	// Prepend parent pages that are defined via the cli flag
	if len(parents) > 0 && parents[0] != "" {
		meta.Parents = append(parents, meta.Parents...)
	}

	// deterministically generate a hash from the page's parents, space, and title
	if titleAppendGeneratedHash {
		path := strings.Join(append(meta.Parents, meta.Space, meta.Title), "/")
		pathHash := sha256.Sum256([]byte(path))
		// postfix is an 8-character hexadecimal string representation of the first 4 out of 32 bytes of the hash
		meta.Title = fmt.Sprintf("%s - %x", meta.Title, pathHash[0:4])
		log.Debugf(
			nil,
			"appended hash to page title: %s",
			meta.Title,
		)
	}

	// Remove trailing spaces from title
	meta.Title = strings.Trim(meta.Title, " ")
	meta.Space = strings.Trim(meta.Space, " ")
	return meta, data[offset:], nil
}

func setTitleFromFilename(meta *Meta, filename string) {
	base := filepath.Base(filename)
	title := strings.TrimSuffix(base, filepath.Ext(base))
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.ReplaceAll(title, "-", " ")
	meta.Title = cases.Title(language.English).String(title)
}

// ExtractDocumentLeadingH1 will extract leading H1 heading
func ExtractDocumentLeadingH1(markdown []byte) string {
	h1 := regexp.MustCompile(`#[^#]\s*(.*)\s*\n`)
	groups := h1.FindSubmatch(markdown)
	if groups == nil {
		return ""
	} else {
		return string(groups[1])
	}
}
