package domain

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Chart-of-accounts import from a spreadsheet matrix (NüHabit
// accounting/domain/coaimport.go): the SULU layout (code in column A) or
// the standard layout with a header row; type and cash-flow heuristics;
// the create/update/skip/error preview.

// CoaRow is one parsed row.
type CoaRow struct {
	Code             string
	Name             string
	ParentCode       string
	AccountTypeCode  string
	Level            int
	IsContra         bool
	IsCashBank       bool
	CashFlowCategory string
	Description      string
	SourceRow        int
}

// CoaIssue is a parse problem.
type CoaIssue struct {
	Row     int
	Code    string
	Message string
}

// CashFlowCategories are the allowed cash flow categories.
var CashFlowCategories = []string{"OPERATING", "INVESTING", "FINANCING", "NON_CASH"}

// ParseCoaMatrix reads the sheet already chosen by the reader.
func ParseCoaMatrix(matrix [][]string) ([]CoaRow, []CoaIssue) {
	if len(matrix) == 0 {
		return nil, []CoaIssue{{Row: 0, Message: "Workbook tidak punya sheet"}}
	}
	if isSuluCoaLayout(matrix) {
		return parseSuluCoaSheet(matrix)
	}
	return parseStandardCoaSheet(matrix)
}

func cellAt(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return strings.TrimFunc(row[i], isSpace)
}

var codeHeader = regexp.MustCompile(`(?i)^(code$|kode)`)

func isSuluCoaLayout(rows [][]string) bool {
	if len(rows) < 2 {
		return false
	}
	first := cellAt(rows[0], 0)
	if codeHeader.MatchString(first) {
		return false
	}
	return NormalizeAccountCode(first) != ""
}

var roundingGain = regexp.MustCompile(`(?i)rounding gain`)

func parseSuluCoaSheet(rows [][]string) ([]CoaRow, []CoaIssue) {
	issues := []CoaIssue{}
	type staged struct {
		code, name string
		sourceRow  int
	}
	var stage []staged
	seen := map[string]int{}
	for idx, row := range rows {
		sourceRow := idx + 1
		rawCode := cellAt(row, 0)
		if rawCode == "" {
			continue
		}
		code := NormalizeAccountCode(rawCode)
		if code == "" {
			issues = append(issues, CoaIssue{Row: sourceRow, Message: "Kode tidak valid: " + rawCode})
			continue
		}
		name := ""
		for c := 1; c <= 6 && name == ""; c++ {
			name = cellAt(row, c)
		}
		if name == "" {
			issues = append(issues, CoaIssue{Row: sourceRow, Code: code, Message: "Nama kosong, baris dilewati"})
			continue
		}
		// The SULU workbook lists Rounding Gain under Rounding Loss's code.
		if code == "8301001" && roundingGain.MatchString(name) {
			code = "8201003"
		}
		if first, dup := seen[code]; dup {
			issues = append(issues, CoaIssue{Row: sourceRow, Code: code, Message: "Kode duplikat (sudah di baris " + strconv.Itoa(first) + ")"})
			continue
		}
		seen[code] = sourceRow
		stage = append(stage, staged{code, name, sourceRow})
	}
	out := make([]CoaRow, len(stage))
	for i, r := range stage {
		level := InferAccountLevel(r.code)
		out[i] = CoaRow{
			Code: r.code, Name: r.name, ParentCode: NearestParentCode(r.code, seen),
			AccountTypeCode: InferAccountTypeCode(r.code), Level: level,
			IsContra: InferIsContra(r.name), IsCashBank: InferIsCashBank(r.name, r.code, level),
			CashFlowCategory: InferCashFlowCategory(r.code, r.name, level), SourceRow: r.sourceRow,
		}
	}
	return out, issues
}

var coaHeaderAliases = map[string]string{
	"account_code": "code", "kode": "code", "kode_akun": "code",
	"account_name": "name", "nama": "name", "nama_akun": "name",
	"parent": "parent_code", "parent_account": "parent_code", "parent_kode": "parent_code",
	"account_type": "account_type_code", "type": "account_type_code", "tipe": "account_type_code",
	"contra": "is_contra", "is_contra_account": "is_contra",
	"cash_bank": "is_cash_bank", "is_cash": "is_cash_bank", "is_bank": "is_cash_bank", "kas_bank": "is_cash_bank",
	"cashflow": "cash_flow_category", "cash_flow": "cash_flow_category",
	"deskripsi": "description",
}

func normalizeCoaHeader(h string) string {
	key := strings.Join(strings.FieldsFunc(strings.ToLower(h), isSpace), "_")
	if alias, ok := coaHeaderAliases[key]; ok {
		return alias
	}
	return key
}

func parseBool(v string, fallback bool) bool {
	switch strings.ToLower(v) {
	case "1", "true", "yes", "y", "ya":
		return true
	case "0", "false", "no", "n", "tidak":
		return false
	}
	return fallback
}

func parseStandardCoaSheet(matrix [][]string) ([]CoaRow, []CoaIssue) {
	issues := []CoaIssue{}
	if len(matrix) < 2 {
		return nil, []CoaIssue{{Row: 1, Message: "File kosong / tanpa data"}}
	}
	headers := make([]string, len(matrix[0]))
	for i := range headers {
		headers[i] = normalizeCoaHeader(cellAt(matrix[0], i))
	}
	col := func(key string) int { return slices.Index(headers, key) }
	codeIdx, nameIdx := col("code"), col("name")
	if codeIdx < 0 || nameIdx < 0 {
		return nil, []CoaIssue{{Row: 1, Message: "Header wajib: code, name"}}
	}
	parentIdx, typeIdx, contraIdx := col("parent_code"), col("account_type_code"), col("is_contra")
	cashBankIdx, cfIdx, descIdx := col("is_cash_bank"), col("cash_flow_category"), col("description")

	var out []CoaRow
	seen := map[string]int{}
	for i := 1; i < len(matrix); i++ {
		row := matrix[i]
		sourceRow := i + 1
		rawCode, name := cellAt(row, codeIdx), cellAt(row, nameIdx)
		if rawCode == "" && name == "" {
			continue
		}
		code := NormalizeAccountCode(rawCode)
		if code == "" {
			issues = append(issues, CoaIssue{Row: sourceRow, Message: "Kode tidak valid: " + rawCode})
			continue
		}
		if name == "" {
			issues = append(issues, CoaIssue{Row: sourceRow, Code: code, Message: "Nama wajib diisi"})
			continue
		}
		if first, dup := seen[code]; dup {
			issues = append(issues, CoaIssue{Row: sourceRow, Code: code, Message: "Kode duplikat (baris " + strconv.Itoa(first) + ")"})
			continue
		}
		seen[code] = sourceRow
		level := InferAccountLevel(code)
		accountType := strings.ToUpper(cellAt(row, typeIdx))
		if accountType == "" {
			accountType = InferAccountTypeCode(code)
		}
		cashFlow := strings.ToUpper(cellAt(row, cfIdx))
		if cashFlow != "" {
			if !slices.Contains(CashFlowCategories, cashFlow) {
				issues = append(issues, CoaIssue{Row: sourceRow, Code: code, Message: "cash_flow_category tidak valid: " + cashFlow})
				continue
			}
		} else {
			cashFlow = InferCashFlowCategory(code, name, level)
		}
		parent := ""
		if raw := cellAt(row, parentIdx); raw != "" {
			parent = NormalizeAccountCode(raw)
		}
		isContra := InferIsContra(name)
		if contraIdx >= 0 {
			isContra = parseBool(cellAt(row, contraIdx), isContra)
		}
		isCashBank := InferIsCashBank(name, code, level)
		if cashBankIdx >= 0 {
			isCashBank = parseBool(cellAt(row, cashBankIdx), isCashBank)
		}
		out = append(out, CoaRow{
			Code: code, Name: name, ParentCode: parent, AccountTypeCode: accountType, Level: level,
			IsContra: isContra, IsCashBank: isCashBank, CashFlowCategory: cashFlow, Description: cellAt(row, descIdx), SourceRow: sourceRow,
		})
	}
	// Rows without a parent take the nearest ancestor among the kept rows.
	kept := make(map[string]int, len(out))
	for _, r := range out {
		kept[r.Code] = r.SourceRow
	}
	for i := range out {
		if out[i].ParentCode == "" {
			out[i].ParentCode = NearestParentCode(out[i].Code, kept)
		}
	}
	return out, issues
}

// idealParentCode is the code one level up, "" for a level-1 code.
func idealParentCode(code string) string {
	p, ok := parseAccountCode(code)
	if !ok {
		return ""
	}
	switch InferAccountLevel(code) {
	case 2:
		return p.class + "000000"
	case 3:
		return p.class + p.group + "00000"
	case 4:
		return p.class + p.group + p.sub + "000"
	}
	return ""
}

// NearestParentCode walks up the ideal parents until one is in codes.
func NearestParentCode(code string, codes map[string]int) string {
	for parent := idealParentCode(code); parent != ""; parent = idealParentCode(parent) {
		if _, ok := codes[parent]; ok {
			return parent
		}
	}
	return ""
}

// InferAccountTypeCode maps the class digit to an account type; class 8
// splits on the second digit (82 is other income).
func InferAccountTypeCode(code string) string {
	if code == "" {
		return TypeExpense
	}
	switch code[0] {
	case '1':
		return TypeAsset
	case '2':
		return TypeLiability
	case '3':
		return TypeEquity
	case '4':
		return TypeRevenue
	case '5':
		return TypeCOGS
	case '8':
		if len(code) >= 2 && code[1] == '2' {
			return TypeOtherIncome
		}
		return TypeOtherExpense
	}
	return TypeExpense
}

var (
	depreciationAny = regexp.MustCompile(`(?i)DEPRECIATION|PENYUSUTAN`)
	nonCashName     = regexp.MustCompile(`DEPRECIATION|PENYUSUTAN|\bDE\b`)
)

// InferCashFlowCategory: headers (level < 4) get none unless depreciation.
func InferCashFlowCategory(code, name string, level int) string {
	if level < 4 && !depreciationAny.MatchString(name) {
		return ""
	}
	has := func(prefixes ...string) bool {
		for _, p := range prefixes {
			if strings.HasPrefix(code, p) {
				return true
			}
		}
		return false
	}
	switch {
	case nonCashName.MatchString(strings.ToUpper(name)) || has("81"):
		return "NON_CASH"
	case has("11", "12", "13", "21"):
		return "OPERATING"
	case has("16", "17"):
		return "INVESTING"
	case has("22", "3"):
		return "FINANCING"
	case has("4", "5", "6", "82", "83"):
		return "OPERATING"
	}
	return ""
}

var contraName = regexp.MustCompile(`(?i)ACCUMULAT|AKUMULASI|ALLOWANCE FOR|CADANGAN|CONTRA`)

// InferIsContra flags accumulated depreciation, allowances and contra accounts.
func InferIsContra(name string) bool { return contraName.MatchString(name) }

var (
	cashBankCode    = regexp.MustCompile(`^110[12]`)
	notCashBankName = regexp.MustCompile(`(?i)LOAN|INTEREST|CHARGE|MDR|RECEIVABLE|\bAR\b|TAX|PPH|COMMISION|COMMISSION|PIUTANG`)
	cashBankName    = regexp.MustCompile(`(?i)\bCASH\b|\bBANK\b|PETTY\s*CASH|\bGIRO\b|\bREKENING\b|\bKAS\b`)
)

// InferIsCashBank: leaf 1101xxx and 1102xxx accounts, else a cash or bank
// name that is not a loan, fee or tax.
func InferIsCashBank(name, code string, level int) bool {
	if cashBankCode.MatchString(code) {
		return level == 4
	}
	if notCashBankName.MatchString(name) {
		return false
	}
	return cashBankName.MatchString(name)
}

// CoaPreviewRow is one classified import row.
type CoaPreviewRow struct {
	SourceRow       int
	Code            string
	Name            string
	ParentCode      string
	AccountTypeCode string
	Action          string
	Message         string
}

// ExistingAccount is a live account, by code.
type ExistingAccount struct{ ID, Name string }

// BuildCoaImportPreview classifies each row against the live chart.
// typeIDs is keyed by the upper-cased account type code.
func BuildCoaImportPreview(parsed []CoaRow, typeIDs map[string]string, existing map[string]ExistingAccount) []CoaPreviewRow {
	importCodes := make(map[string]bool, len(parsed))
	for _, r := range parsed {
		importCodes[r.Code] = true
	}
	out := make([]CoaPreviewRow, len(parsed))
	for i, r := range parsed {
		p := CoaPreviewRow{SourceRow: r.SourceRow, Code: r.Code, Name: r.Name, ParentCode: r.ParentCode, AccountTypeCode: r.AccountTypeCode}
		ex, found := existing[r.Code]
		_, parentExists := existing[r.ParentCode]
		switch {
		case typeIDs[r.AccountTypeCode] == "":
			p.Action, p.Message = "error", "Account type "+r.AccountTypeCode+" tidak ditemukan"
		case r.ParentCode != "" && !importCodes[r.ParentCode] && !parentExists:
			p.Action, p.Message = "error", "Parent "+r.ParentCode+" tidak ditemukan"
		case !found:
			p.Action = "create"
		case ex.Name == r.Name:
			p.Action, p.Message = "skip", "Tidak berubah"
		default:
			p.Action = "update"
		}
		out[i] = p
	}
	return out
}

// CoaImportSummary counts the preview actions; parse issues count as errors.
type CoaImportSummary struct{ Create, Update, Skip, Error int }

func SummarizeCoaImport(preview []CoaPreviewRow, issues []CoaIssue) CoaImportSummary {
	s := CoaImportSummary{Error: len(issues)}
	for _, p := range preview {
		switch p.Action {
		case "create":
			s.Create++
		case "update":
			s.Update++
		case "skip":
			s.Skip++
		case "error":
			s.Error++
		}
	}
	return s
}
