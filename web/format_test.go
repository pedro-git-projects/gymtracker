package web

import "testing"

func TestLinkify(t *testing.T) {
	link := func(u string) string {
		return `<a href="` + u + `" target="_blank" rel="noopener noreferrer">` + u + `</a>`
	}
	cases := map[string]string{
		"elbows in":                         "elbows in",
		"<b>bold</b> & co":                  "&lt;b&gt;bold&lt;/b&gt; &amp; co",
		"https://youtu.be/abc":              link("https://youtu.be/abc"),
		"see https://youtu.be/abc.":         "see " + link("https://youtu.be/abc") + ".",
		"(https://a.b/c) then http://d.e/f": "(" + link("https://a.b/c") + ") then " + link("http://d.e/f"),
		"https://y.t/watch?v=1&t=2s":        link("https://y.t/watch?v=1&amp;t=2s"),
		"javascript:alert(1)":               "javascript:alert(1)",
	}
	for in, want := range cases {
		if got := string(linkify(in)); got != want {
			t.Errorf("linkify(%q)\n got %s\nwant %s", in, got, want)
		}
	}
}

func TestFormatLoad(t *testing.T) {
	cases := []struct {
		bw         bool
		load       uint16
		full, chip string
	}{
		{false, 40, "40kg", "40"},
		{false, 0, "0kg", "0"},
		{true, 0, "BW", "BW"},
		{true, 10, "BW+10kg", "BW+10"},
	}
	for _, c := range cases {
		if got := formatLoad(c.bw, c.load); got != c.full {
			t.Errorf("formatLoad(%v, %d) = %q", c.bw, c.load, got)
		}
		if got := formatChip(c.bw, c.load); got != c.chip {
			t.Errorf("formatChip(%v, %d) = %q", c.bw, c.load, got)
		}
	}
}
