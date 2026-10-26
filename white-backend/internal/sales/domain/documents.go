package domain

import (
	"regexp"
	"sort"
	"strings"
	"time"
)

// Contract templates are Markdown with {{placeholders}}. The renderer
// fills them from the deal, account and quotation; the PDF adapter then
// lays out a markdown-lite subset (headings, paragraphs, lists, tables,
// bold).

// ContractTypes are the template kinds.
var ContractTypes = []string{"retainer", "project", "nda"}

// Contract states: draft -> sent -> signed -> active -> ended | cancelled.
const (
	ContractDraft     = "draft"
	ContractSent      = "sent"
	ContractSigned    = "signed"
	ContractActive    = "active"
	ContractEnded     = "ended"
	ContractCancelled = "cancelled"
)

// ContractTransitionAllowed is the contract state machine.
func ContractTransitionAllowed(from, to string) bool {
	switch to {
	case ContractSent:
		return from == ContractDraft || from == ContractSent
	case ContractSigned:
		return from == ContractSent
	case ContractActive:
		return from == ContractSigned
	case ContractEnded:
		return from == ContractActive
	case ContractCancelled:
		return from == ContractDraft || from == ContractSent || from == ContractSigned || from == ContractActive
	}
	return false
}

// ContractPlaceholders documents every placeholder the renderer knows (the
// template editor shows this list).
var ContractPlaceholders = map[string]string{
	"agency_name":     "Nama agensi (WHITE)",
	"contract_number": "Nomor kontrak (KTR-YYMM-####)",
	"today":           "Tanggal hari ini",
	"client_name":     "Nama klien (akun atau instansi lead)",
	"client_address":  "Alamat klien",
	"pic_name":        "Nama PIC klien",
	"pic_title":       "Jabatan PIC klien",
	"pic_email":       "Email PIC",
	"pic_phone":       "Telepon PIC",
	"deal_title":      "Judul deal",
	"quote_number":    "Nomor quotation",
	"services_table":  "Tabel layanan dari quotation",
	"services_list":   "Daftar layanan (bullet)",
	"fees":            "Total nilai kontrak (termasuk PPN)",
	"subtotal":        "Subtotal sebelum diskon",
	"discount":        "Nominal diskon",
	"ppn":             "Nominal PPN",
	"payment_terms":   "Daftar termin pembayaran",
	"terms":           "Catatan dan ketentuan tambahan dari quotation",
	"start_date":      "Tanggal mulai",
	"end_date":        "Tanggal selesai",
	"duration_months": "Durasi dalam bulan",
}

// ContractContext is what a contract is rendered from.
type ContractContext struct {
	AgencyName     string
	ContractNumber string
	Today          time.Time
	ClientName     string
	ClientAddress  string
	PicName        string
	PicTitle       string
	PicEmail       string
	PicPhone       string
	DealTitle      string
	Quotation      *Quotation
	StartDate      *time.Time
	EndDate        *time.Time
}

var contractPlaceholder = regexp.MustCompile(`\{\{\s*([a-z_]+)\s*\}\}`)

// ContractValues builds the placeholder map.
func ContractValues(c ContractContext) map[string]string {
	v := map[string]string{
		"agency_name":     c.AgencyName,
		"contract_number": c.ContractNumber,
		"today":           FormatDateID(c.Today.In(Jakarta)),
		"client_name":     c.ClientName,
		"client_address":  orDash(c.ClientAddress),
		"pic_name":        orDash(c.PicName),
		"pic_title":       orDash(c.PicTitle),
		"pic_email":       orDash(c.PicEmail),
		"pic_phone":       orDash(c.PicPhone),
		"deal_title":      c.DealTitle,
		"start_date":      "-",
		"end_date":        "-",
		"duration_months": "-",
	}
	if c.StartDate != nil {
		v["start_date"] = FormatDateID(*c.StartDate)
	}
	if c.EndDate != nil {
		v["end_date"] = FormatDateID(*c.EndDate)
	}
	if c.StartDate != nil && c.EndDate != nil && !c.EndDate.Before(*c.StartDate) {
		months := (c.EndDate.Year()-c.StartDate.Year())*12 + int(c.EndDate.Month()-c.StartDate.Month())
		if c.EndDate.Day() >= c.StartDate.Day() {
			months++
		}
		v["duration_months"] = FormatNumber(float64(max(1, months)), 0)
	}
	q := c.Quotation
	if q == nil {
		for _, k := range []string{"quote_number", "services_table", "services_list", "fees", "subtotal", "discount", "ppn", "payment_terms", "terms"} {
			v[k] = "-"
		}
		return v
	}
	v["quote_number"] = q.QuoteNumber
	v["fees"] = q.Total.Rupiah()
	v["subtotal"] = q.Subtotal.Rupiah()
	v["discount"] = q.DiscountNominal.Rupiah()
	v["ppn"] = q.PpnNominal.Rupiah()
	var table, list strings.Builder
	table.WriteString("| No | Layanan | Qty | Harga Satuan | Jumlah |\n|---|---|---|---|---|\n")
	for i, it := range q.Items {
		desc := it.Description
		if it.Scope != "" {
			desc += ": " + strings.ReplaceAll(it.Scope, "\n", " ")
		}
		table.WriteString("| " + FormatNumber(float64(i+1), 0) + " | " + strings.ReplaceAll(desc, "|", "/") + " | " +
			FormatNumber(it.Qty, 3) + " " + it.Unit + " | " + it.UnitPrice.Rupiah() + " | " + it.LineTotal.Rupiah() + " |\n")
		list.WriteString("- **" + it.Description + "**")
		if it.Scope != "" {
			list.WriteString(": " + strings.ReplaceAll(it.Scope, "\n", " "))
		}
		list.WriteString("\n")
	}
	v["services_table"] = strings.TrimRight(table.String(), "\n")
	v["services_list"] = strings.TrimRight(list.String(), "\n")
	if len(q.Terms) == 0 {
		v["payment_terms"] = "- Pembayaran penuh: " + q.Total.Rupiah()
	} else {
		percents := make([]float64, len(q.Terms))
		for i, t := range q.Terms {
			percents[i] = t.Percent
		}
		amounts := AllocateTermAmounts(q.Total, percents)
		var terms strings.Builder
		for i, t := range q.Terms {
			terms.WriteString("- " + t.Label + " (" + FormatNumber(t.Percent, 2) + "%): " + amounts[i].Rupiah())
			if t.DueDate != nil {
				terms.WriteString(", jatuh tempo " + FormatDateID(*t.DueDate))
			}
			terms.WriteString("\n")
		}
		v["payment_terms"] = strings.TrimRight(terms.String(), "\n")
	}
	if q.Notes != nil && JSTrim(*q.Notes) != "" {
		v["terms"] = *q.Notes
	} else {
		v["terms"] = "-"
	}
	return v
}

func orDash(s string) string {
	if JSTrim(s) == "" {
		return "-"
	}
	return s
}

// RenderContract fills the placeholders; unknown ones are left visible so
// the salesperson notices them in the draft.
func RenderContract(body string, values map[string]string) string {
	return contractPlaceholder.ReplaceAllStringFunc(body, func(m string) string {
		key := contractPlaceholder.FindStringSubmatch(m)[1]
		if v, ok := values[key]; ok {
			return v
		}
		return m
	})
}

// UnknownPlaceholders lists placeholders a template uses that the
// renderer does not fill.
func UnknownPlaceholders(body string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range contractPlaceholder.FindAllStringSubmatch(body, -1) {
		if _, ok := ContractPlaceholders[m[1]]; !ok && !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	sort.Strings(out)
	return out
}

// Markdown-lite blocks.
const (
	BlockHeading   = "heading"
	BlockParagraph = "paragraph"
	BlockBullets   = "bullets"
	BlockNumbered  = "numbered"
	BlockTable     = "table"
	BlockRule      = "rule"
)

// Block is one laid-out unit of a markdown-lite document.
type Block struct {
	Kind  string
	Level int        // heading level 1-3
	Text  string     // heading or paragraph text (inline **bold** kept)
	Items []string   // list items
	Rows  [][]string // table rows, first row is the header
}

var (
	headingRe  = regexp.MustCompile(`^(#{1,3})\s+(.*)$`)
	bulletRe   = regexp.MustCompile(`^\s*[-*]\s+(.*)$`)
	numberedRe = regexp.MustCompile(`^\s*\d+[.)]\s+(.*)$`)
	tableSepRe = regexp.MustCompile(`^\|?\s*:?-{2,}:?\s*(\|\s*:?-{2,}:?\s*)*\|?$`)
)

// ParseMarkdown splits markdown-lite text into blocks.
func ParseMarkdown(src string) []Block {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	var blocks []Block
	var para []string
	flush := func() {
		if len(para) > 0 {
			blocks = append(blocks, Block{Kind: BlockParagraph, Text: strings.Join(para, " ")})
			para = nil
		}
	}
	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], " \t")
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			flush()
		case trimmed == "---" || trimmed == "***":
			flush()
			blocks = append(blocks, Block{Kind: BlockRule})
		case headingRe.MatchString(trimmed):
			flush()
			m := headingRe.FindStringSubmatch(trimmed)
			blocks = append(blocks, Block{Kind: BlockHeading, Level: len(m[1]), Text: strings.TrimSpace(m[2])})
		case strings.HasPrefix(trimmed, "|"):
			flush()
			var rows [][]string
			for ; i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|"); i++ {
				row := strings.TrimSpace(lines[i])
				if tableSepRe.MatchString(row) {
					continue
				}
				rows = append(rows, tableCells(row))
			}
			i--
			blocks = append(blocks, Block{Kind: BlockTable, Rows: rows})
		case bulletRe.MatchString(line), numberedRe.MatchString(line):
			flush()
			kind, re := BlockBullets, bulletRe
			if !bulletRe.MatchString(line) {
				kind, re = BlockNumbered, numberedRe
			}
			var items []string
			for ; i < len(lines) && re.MatchString(lines[i]); i++ {
				items = append(items, strings.TrimSpace(re.FindStringSubmatch(lines[i])[1]))
			}
			i--
			blocks = append(blocks, Block{Kind: kind, Items: items})
		default:
			para = append(para, trimmed)
		}
	}
	flush()
	return blocks
}

func tableCells(row string) []string {
	row = strings.TrimSuffix(strings.TrimPrefix(row, "|"), "|")
	parts := strings.Split(row, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// Span is a run of inline text.
type Span struct {
	Text string
	Bold bool
}

// InlineSpans splits **bold** runs out of a line.
func InlineSpans(s string) []Span {
	var out []Span
	bold := false
	for {
		i := strings.Index(s, "**")
		if i < 0 {
			if s != "" {
				out = append(out, Span{Text: s, Bold: bold})
			}
			return out
		}
		if i > 0 {
			out = append(out, Span{Text: s[:i], Bold: bold})
		}
		bold = !bold
		s = s[i+2:]
	}
}

// PlainText strips the bold markers.
func PlainText(s string) string { return strings.ReplaceAll(s, "**", "") }
