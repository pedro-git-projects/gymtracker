package web

import (
	"fmt"
	"html/template"
	"regexp"
	"strings"
)

var urlPattern = regexp.MustCompile(`https?://[^\s<>"']+`)

// linkify escapes free text and turns the http(s) URLs in it into links.
func linkify(text string) template.HTML {
	var b strings.Builder
	last := 0
	for _, m := range urlPattern.FindAllStringIndex(text, -1) {
		link := strings.TrimRight(text[m[0]:m[1]], ".,;:!?)")
		end := m[0] + len(link)
		b.WriteString(template.HTMLEscapeString(text[last:m[0]]))
		esc := template.HTMLEscapeString(link)
		fmt.Fprintf(&b, `<a href="%s" target="_blank" rel="noopener noreferrer">%s</a>`, esc, esc)
		last = end
	}
	b.WriteString(template.HTMLEscapeString(text[last:]))
	return template.HTML(b.String())
}

// formatLoad renders a load with its unit. For bodyweight exercises the
// stored load is the weight added on top of the lifter.
func formatLoad(bodyweight bool, load uint16) string {
	if !bodyweight {
		return fmt.Sprintf("%dkg", load)
	}
	if load == 0 {
		return "BW"
	}
	return fmt.Sprintf("BW+%dkg", load)
}

// formatChip is the compact form used in set lists such as "8@40".
func formatChip(bodyweight bool, load uint16) string {
	return strings.TrimSuffix(formatLoad(bodyweight, load), "kg")
}
