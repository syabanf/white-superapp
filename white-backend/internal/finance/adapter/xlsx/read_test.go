package xlsx

import (
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/xuri/excelize/v2"
)

func TestReadCoaSheetPrefersCoa(t *testing.T) {
	f := excelize.NewFile()
	_ = f.SetCellValue("Sheet1", "A1", "ignored")
	if _, err := f.NewSheet("COA"); err != nil {
		t.Fatal(err)
	}
	rows := [][]any{{"Kode Akun", "Nama Akun"}, {"1 1 01 000", "Kas"}, {"1 1 01 001", "Kas Kecil"}}
	for i, r := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		if err := f.SetSheetRow("COA", cell, &r); err != nil {
			t.Fatal(err)
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	matrix, err := ReadCoaSheet(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	parsed, issues := domain.ParseCoaMatrix(matrix)
	if len(issues) != 0 || len(parsed) != 2 || parsed[1].ParentCode != "1101000" || !parsed[1].IsCashBank {
		t.Fatalf("%+v %+v", parsed, issues)
	}
	if _, err := ReadCoaSheet([]byte("not a workbook")); err == nil {
		t.Fatal("garbage must fail")
	}
}
