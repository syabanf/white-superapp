package main

import (
	"github.com/syabanf/white-superapp/white-backend/internal/publishing"
	"github.com/syabanf/white-superapp/white-backend/pkg/app"
)

func main() {
	app.Main(publishing.Spec())
}
