package domain

import (
	"regexp"
	"strings"
)

// Markdown-lite as the PDF summary page reads it: #/##/### headings,
// "-", "*" and "•" bullets, "1." or "1)" lists and **bold** inline.

type BlockKind int

const (
	BlockParagraph BlockKind = iota
	BlockHeading
	BlockBullet
	BlockNumbered
)

type Span struct {
	Text string
	Bold bool
}

type Block struct {
	Kind BlockKind
	// Number is the list marker of a numbered item ("3").
	Number string
	Spans  []Span
}

var (
	mdHeading  = regexp.MustCompile(`^(#{1,3})\s+(.*)$`)
	mdBullet   = regexp.MustCompile(`^\s*[-*•]\s+(.*)$`)
	mdNumbered = regexp.MustCompile(`^\s*(\d+)[.)]\s+(.*)$`)
	mdBold     = regexp.MustCompile(`\*\*[^*]+\*\*`)
)

// ParseMarkdownLite skips blank lines; every other line is one block.
func ParseMarkdownLite(text string) []Block {
	var out []Block
	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line := strings.TrimRight(raw, " \t\r\v\f")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if m := mdHeading.FindStringSubmatch(line); m != nil {
			out = append(out, Block{Kind: BlockHeading, Spans: InlineBold(m[2])})
		} else if m := mdBullet.FindStringSubmatch(line); m != nil {
			out = append(out, Block{Kind: BlockBullet, Spans: InlineBold(m[1])})
		} else if m := mdNumbered.FindStringSubmatch(line); m != nil {
			out = append(out, Block{Kind: BlockNumbered, Number: m[1], Spans: InlineBold(m[2])})
		} else {
			out = append(out, Block{Kind: BlockParagraph, Spans: InlineBold(line)})
		}
	}
	return out
}

// InlineBold splits "a **b** c" into plain and bold spans.
func InlineBold(s string) []Span {
	var out []Span
	last := 0
	for _, loc := range mdBold.FindAllStringIndex(s, -1) {
		if loc[0] > last {
			out = append(out, Span{Text: s[last:loc[0]]})
		}
		out = append(out, Span{Text: s[loc[0]+2 : loc[1]-2], Bold: true})
		last = loc[1]
	}
	if last < len(s) {
		out = append(out, Span{Text: s[last:]})
	}
	return out
}
