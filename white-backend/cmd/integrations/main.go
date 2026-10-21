package main

import (
	"github.com/syabanf/white-superapp/white-backend/internal/integrations"
	"github.com/syabanf/white-superapp/white-backend/pkg/app"
)

func main() {
	app.Main(integrations.Spec())
}
