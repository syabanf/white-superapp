package domain

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// lib/sales-funnel/lead-import.ts and lead-spreadsheet.ts.

// LeadImportMaxRows is MAX_ROWS: data rows per import.
const LeadImportMaxRows = 500

// LeadImportMaxBytes caps the uploaded spreadsheet.
const LeadImportMaxBytes = 5 << 20

var leadHeaderAliases = map[string]string{
	"org_name": "nama_instansi", "instansi": "nama_instansi", "company": "nama_instansi", "perusahaan": "nama_instansi", "nama_perusahaan": "nama_instansi",
	"org_type": "jenis_instansi", "jenis": "jenis_instansi",
	"pic_name": "nama_pic", "pic": "nama_pic", "contact_person": "nama_pic",
	"pic_title": "jabatan_pic", "jabatan": "jabatan_pic",
	"pic_phone": "wa_pic", "phone": "wa_pic", "no_wa": "wa_pic", "telepon": "wa_pic",
	"pic_email": "email_pic", "email": "email_pic",
	"city": "kota", "source": "sumber", "temperature": "suhu",
	"notes": "catatan", "keterangan": "catatan",
}

// NormalizeLeadHeader is normalizeLeadSpreadsheetHeader: "Nama PIC" becomes
// "nama_pic", then aliases map it to the canonical key.
func NormalizeLeadHeader(header string) string {
	var b strings.Builder
	inSpace := false
	for _, r := range strings.ToLower(JSTrim(header)) {
		if unicode.IsSpace(r) {
			if !inSpace {
				b.WriteByte('_')
			}
			inSpace = true
			continue
		}
		inSpace = false
		b.WriteRune(r)
	}
	key := b.String()
	if alias, ok := leadHeaderAliases[key]; ok {
		return alias
	}
	return key
}

var importEmailRe = regexp.MustCompile(`^\S+@\S+\.\S+$`)

// LeadImportRow is the lead one spreadsheet row inserts.
type LeadImportRow struct {
	OrgName, OrgType, PicName, PicPhone, Source, Temperature string
	PicTitle, PicEmail, City, Notes                          *string
}

// LeadImportKind classifies a row.
type LeadImportKind string

const (
	LeadRowEmpty   LeadImportKind = "empty"
	LeadRowInvalid LeadImportKind = "invalid"
	LeadRowOK      LeadImportKind = "ok"
)

// LeadImportRowResult is LeadImportRowResult.
type LeadImportRowResult struct {
	Kind     LeadImportKind
	Message  string
	Lead     LeadImportRow
	RawPhone string
	Warning  string
}

// LeadDedupKey is leadDedupKey: organisation (case-insensitive) plus
// WhatsApp number.
func LeadDedupKey(phone, orgName string) string {
	return phone + "|" + strings.ToLower(JSTrim(orgName))
}

// MapLeadImportRow is mapLeadImportRow: one row under normalized headers.
// An invalid email does not fail the row: it is dropped with a warning.
func MapLeadImportRow(headers, cells []string) LeadImportRowResult {
	row := map[string]string{}
	empty := true
	for i, h := range headers {
		if i < len(cells) {
			row[h] = cells[i]
			if JSTrim(cells[i]) != "" {
				empty = false
			}
		}
	}
	if empty {
		return LeadImportRowResult{Kind: LeadRowEmpty}
	}
	orgName, picName, rawPhone := JSTrim(row["nama_instansi"]), JSTrim(row["nama_pic"]), JSTrim(row["wa_pic"])
	if orgName == "" || picName == "" || rawPhone == "" {
		return LeadImportRowResult{Kind: LeadRowInvalid, Message: "Field wajib kosong: nama_instansi / nama_pic / wa_pic"}
	}
	phone := NormalizePhone(rawPhone)
	if !IsValidNormalizedPhone(phone) {
		return LeadImportRowResult{Kind: LeadRowInvalid, Message: "No. WA tidak valid: " + rawPhone}
	}
	res := LeadImportRowResult{Kind: LeadRowOK, RawPhone: rawPhone}
	emailRaw := JSTrim(row["email_pic"])
	var email *string
	if emailRaw != "" && importEmailRe.MatchString(emailRaw) {
		e := SliceUTF16(emailRaw, 150)
		email = &e
	} else if emailRaw != "" {
		res.Warning = "Email diabaikan (tidak valid): " + emailRaw
	}
	res.Lead = LeadImportRow{
		OrgName:     orgName,
		OrgType:     pickEnum(row["jenis_instansi"], LeadOrgTypes, "corporate"),
		PicName:     picName,
		PicTitle:    TrimmedOrNil(row["jabatan_pic"]),
		PicPhone:    phone,
		PicEmail:    email,
		City:        TrimmedOrNil(row["kota"]),
		Source:      pickEnum(row["sumber"], LeadSources, "lainnya"),
		Temperature: pickEnum(row["suhu"], LeadTemperatures, "hangat"),
		Notes:       TrimmedOrNil(row["catatan"]),
	}
	return res
}

func pickEnum(value string, allowed []string, fallback string) string {
	raw := strings.ToLower(JSTrim(value))
	if raw != "" && Contains(allowed, raw) {
		return raw
	}
	return fallback
}

// ParseCSVMatrix is parseCsvMatrix: split on "\n", drop blank lines,
// commas separate cells unless inside quotes, quotes are dropped, cells
// are trimmed.
func ParseCSVMatrix(text string) [][]string {
	var out [][]string
	text = strings.TrimPrefix(text, "\ufeff")
	for _, line := range strings.Split(text, "\n") {
		if JSTrim(line) == "" {
			continue
		}
		var row []string
		var cur strings.Builder
		inQuotes := false
		for _, ch := range line {
			switch {
			case ch == '"':
				inQuotes = !inQuotes
			case ch == ',' && !inQuotes:
				row = append(row, JSTrim(cur.String()))
				cur.Reset()
			default:
				cur.WriteRune(ch)
			}
		}
		out = append(out, append(row, JSTrim(cur.String())))
	}
	return out
}

var ErrSpreadsheet = errors.New("File tidak dapat dibaca. Gunakan format .xlsx atau .csv")

// SpreadsheetMatrix is parseSpreadsheetMatrix: a .csv name is read as text,
// anything else as the first sheet of an .xlsx workbook.
func SpreadsheetMatrix(data []byte, fileName string) ([][]string, error) {
	if strings.HasSuffix(strings.ToLower(fileName), ".csv") {
		return ParseCSVMatrix(strings.ToValidUTF8(string(data), "�")), nil
	}
	return parseXLSX(data)
}

// parseXLSX reads the first worksheet of a workbook with shared strings,
// inline strings, numbers and booleans. Formulas give their cached value.
func parseXLSX(data []byte) ([][]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, ErrSpreadsheet
	}
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	read := func(name string) ([]byte, error) {
		f, ok := files[name]
		if !ok {
			return nil, nil
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(io.LimitReader(rc, 64<<20))
	}
	var shared []string
	if raw, err := read("xl/sharedStrings.xml"); err != nil {
		return nil, ErrSpreadsheet
	} else if raw != nil {
		var sst struct {
			Items []struct {
				T string `xml:"t"`
				R []struct {
					T string `xml:"t"`
				} `xml:"r"`
			} `xml:"si"`
		}
		if err := xml.Unmarshal(raw, &sst); err != nil {
			return nil, ErrSpreadsheet
		}
		for _, si := range sst.Items {
			s := si.T
			for _, r := range si.R {
				s += r.T
			}
			shared = append(shared, s)
		}
	}
	sheetPath := firstSheetPath(read)
	raw, err := read(sheetPath)
	if err != nil || raw == nil {
		return nil, ErrSpreadsheet
	}
	var ws struct {
		Rows []struct {
			Cells []struct {
				Ref    string `xml:"r,attr"`
				Type   string `xml:"t,attr"`
				Value  string `xml:"v"`
				Inline struct {
					T string `xml:"t"`
				} `xml:"is"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
	if err := xml.Unmarshal(raw, &ws); err != nil {
		return nil, ErrSpreadsheet
	}
	var out [][]string
	width := 0
	for _, row := range ws.Rows {
		var cells []string
		for i, c := range row.Cells {
			col := i
			if c.Ref != "" {
				col = columnIndex(c.Ref)
			}
			for len(cells) <= col {
				cells = append(cells, "")
			}
			v := c.Value
			switch c.Type {
			case "s":
				if n, err := strconv.Atoi(v); err == nil && n >= 0 && n < len(shared) {
					v = shared[n]
				}
			case "inlineStr":
				v = c.Inline.T
			case "b":
				v = map[string]string{"1": "true", "0": "false"}[v]
			}
			cells[col] = JSTrim(v)
		}
		width = max(width, len(cells))
		out = append(out, cells)
	}
	for i := range out {
		for len(out[i]) < width {
			out[i] = append(out[i], "")
		}
	}
	return out, nil
}

// firstSheetPath resolves the first sheet through workbook.xml and its
// relationships, falling back to sheet1.xml.
func firstSheetPath(read func(string) ([]byte, error)) string {
	const fallback = "xl/worksheets/sheet1.xml"
	wb, _ := read("xl/workbook.xml")
	rels, _ := read("xl/_rels/workbook.xml.rels")
	if wb == nil || rels == nil {
		return fallback
	}
	var book struct {
		Sheets []struct {
			RID string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"sheets>sheet"`
	}
	var rel struct {
		Items []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	if xml.Unmarshal(wb, &book) != nil || xml.Unmarshal(rels, &rel) != nil || len(book.Sheets) == 0 {
		return fallback
	}
	for _, r := range rel.Items {
		if r.ID == book.Sheets[0].RID {
			if strings.HasPrefix(r.Target, "/") {
				return strings.TrimPrefix(r.Target, "/")
			}
			return path.Join("xl", r.Target)
		}
	}
	return fallback
}

// columnIndex turns "C7" into 2.
func columnIndex(ref string) int {
	n := 0
	for _, c := range ref {
		if c < 'A' || c > 'Z' {
			break
		}
		n = n*26 + int(c-'A'+1)
	}
	return max(0, n-1)
}
