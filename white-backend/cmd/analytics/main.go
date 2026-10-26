package main

import (
	"github.com/syabanf/white-superapp/white-backend/internal/analytics"
	"github.com/syabanf/white-superapp/white-backend/pkg/app"
)

func main() {
	app.Main(analytics.Spec())
}
