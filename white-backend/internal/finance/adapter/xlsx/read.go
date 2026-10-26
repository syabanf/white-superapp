// Package xlsx reads the chart-of-accounts workbook.
package xlsx

import (
	"bytes"
	"strings"

	"github.com/xuri/excelize/v2"
)

const maxRows = 5000

// ReadCoaSheet returns the rows of the sheet named "coa" (any case) or the
// first sheet, padded to the widest row.
func ReadCoaSheet(data []byte) ([][]string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return [][]string{}, nil
	}
	sheet := sheets[0]
	for _, name := range sheets {
		if strings.EqualFold(name, "coa") {
			sheet = name
			break
		}
	}
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	if len(rows) > maxRows {
		rows = rows[:maxRows]
	}
	width := 0
	for _, r := range rows {
		width = max(width, len(r))
	}
	out := make([][]string, len(rows))
	for i, r := range rows {
		out[i] = make([]string, width)
		copy(out[i], r)
	}
	return out, nil
}
