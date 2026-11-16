package gosanitize

import (
	"bytes"
	"html"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Policy defines permitted tags, attributes, and URL schemes for HTML sanitization.
type Policy struct {
	AllowedTags        map[string]bool
	AllowedAttributes  map[string]map[string]bool // tag -> attr -> allowed
	AllowedURLSchemes  map[string]bool
	DropEntireElements map[string]bool // tags whose contents must be discarded entirely (e.g. script, style)
}

// UGCPolicy permits basic formatting tags suitable for user comments, reviews, and blog posts.
var UGCPolicy = Policy{
	AllowedTags: map[string]bool{
		"b": true, "strong": true, "i": true, "em": true, "u": true,
		"p": true, "br": true, "ul": true, "ol": true, "li": true,
		"blockquote": true, "code": true, "pre": true, "a": true,
	},
	AllowedAttributes: map[string]map[string]bool{
		"a": {"href": true, "title": true, "rel": true, "target": true},
	},
	AllowedURLSchemes: map[string]bool{
		"http": true, "https": true, "mailto": true,
	},
	DropEntireElements: map[string]bool{
		"script": true, "style": true, "iframe": true, "object": true,
		"embed": true, "applet": true, "form": true, "svg": true,
	},
}

// StripHTML removes all HTML tags and unescapes entities, returning clean plain text.
func StripHTML(input string) string {
	if input == "" {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(input))

	inTag := false
	inScriptOrStyle := false
	var tagBuf strings.Builder

	runes := []rune(input)
	rLen := len(runes)

	for i := 0; i < rLen; i++ {
		r := runes[i]

		if r == '<' {
			inTag = true
			tagBuf.Reset()
			continue
		}

		if inTag {
			if r == '>' {
				inTag = false
				tagStr := strings.ToLower(strings.TrimSpace(tagBuf.String()))
				if strings.HasPrefix(tagStr, "script") || strings.HasPrefix(tagStr, "style") {
					inScriptOrStyle = true
				} else if strings.HasPrefix(tagStr, "/script") || strings.HasPrefix(tagStr, "/style") {
					inScriptOrStyle = false
				}
			} else {
				tagBuf.WriteRune(r)
			}
			continue
		}

		if !inScriptOrStyle {
			sb.WriteRune(r)
		}
	}

	return strings.TrimSpace(html.UnescapeString(sb.String()))
}

// SanitizeHTML strips unauthorized HTML tags, removes dangerous elements, and cleans URLs according to policy.
func SanitizeHTML(input string, policy Policy) string {
	if input == "" {
		return ""
	}

	var out bytes.Buffer
	rLen := len(input)
	i := 0

	for i < rLen {
		if input[i] == '<' {
			// Find closing '>'
			end := strings.IndexByte(input[i:], '>')
			if end == -1 {
				// Malformed unclosed tag: escape it safely
				out.WriteString(html.EscapeString(input[i:]))
				break
			}
			tagContent := input[i+1 : i+end]
			i += end + 1

			// Check if closing tag
			trimmed := strings.TrimSpace(tagContent)
			if trimmed == "" {
				continue
			}

			isClosing := false
			if trimmed[0] == '/' {
				isClosing = true
				trimmed = strings.TrimSpace(trimmed[1:])
			}

			parts := strings.Fields(trimmed)
			if len(parts) == 0 {
				continue
			}

			tagName := strings.ToLower(parts[0])

			// If tag is in DropEntireElements, skip until closing tag
			if policy.DropEntireElements[tagName] && !isClosing {
				closeSeq := "</" + tagName + ">"
				closeIdx := strings.Index(strings.ToLower(input[i:]), closeSeq)
				if closeIdx != -1 {
					i += closeIdx + len(closeSeq)
				}
				continue
			}

			// If tag not allowed, drop tag
			if !policy.AllowedTags[tagName] {
				continue
			}

			if isClosing {
				out.WriteString("</" + tagName + ">")
				continue
			}

			// Process attributes
			out.WriteString("<" + tagName)
			if allowedAttrs, hasAttrs := policy.AllowedAttributes[tagName]; hasAttrs && len(parts) > 1 {
				cleanAttrs := parseAndSanitizeAttributes(parts[1:], allowedAttrs, policy.AllowedURLSchemes)
				if cleanAttrs != "" {
					out.WriteString(" " + cleanAttrs)
				}
			}
			out.WriteString(">")
		} else {
			out.WriteByte(input[i])
			i++
		}
	}

	return out.String()
}

func parseAndSanitizeAttributes(parts []string, allowed map[string]bool, allowedSchemes map[string]bool) string {
	fullAttrStr := strings.Join(parts, " ")
	var validAttrs []string

	// Simple key=value tokenizer
	attrPairs := strings.Split(fullAttrStr, " ")
	for _, pair := range attrPairs {
		eqIdx := strings.Index(pair, "=")
		if eqIdx == -1 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(pair[:eqIdx]))
		val := strings.Trim(pair[eqIdx+1:], `"'`)

		if !allowed[key] {
			continue
		}

		// Security: Disallow javascript/data/vbscript URLs on href/src
		if key == "href" || key == "src" {
			u, err := url.Parse(val)
			if err != nil || (u.Scheme != "" && !allowedSchemes[strings.ToLower(u.Scheme)]) {
				continue
			}
		}

		validAttrs = append(validAttrs, key+`="`+html.EscapeString(val)+`"`)
	}

	return strings.Join(validAttrs, " ")
}

// EscapeSQLWildcards escapes '%' and '_' in user inputs to prevent accidental full-table scans in LIKE queries.
func EscapeSQLWildcards(input string) string {
	input = strings.ReplaceAll(input, `\`, `\\`)
	input = strings.ReplaceAll(input, `%`, `\%`)
	input = strings.ReplaceAll(input, `_`, `\_`)
	return input
}

// CleanUnicode removes invisible zero-width spaces, bidi override markers, and invalid UTF-8 bytes.
func CleanUnicode(input string) string {
	if !utf8.ValidString(input) {
		input = strings.ToValidUTF8(input, "")
	}

	return strings.Map(func(r rune) rune {
		// Zero-width spaces, soft hyphens, BOMs, and directional override characters
		switch r {
		case '\u200B', '\u200C', '\u200D', '\uFEFF', '\u00AD', '\u202A', '\u202B', '\u202C', '\u202D', '\u202E':
			return -1 // Drop rune
		}
		// Drop unprintable control runes except \n, \r, \t
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, input)
}
