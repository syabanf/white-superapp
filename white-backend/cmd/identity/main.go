package main

import (
	"github.com/syabanf/white-superapp/white-backend/internal/identity"
	"github.com/syabanf/white-superapp/white-backend/pkg/app"
)

func main() {
	app.Main(identity.Spec())
}
