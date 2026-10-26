package main

import (
	"github.com/syabanf/white-superapp/white-backend/internal/reports"
	"github.com/syabanf/white-superapp/white-backend/pkg/app"
)

func main() {
	app.Main(reports.Spec())
}
