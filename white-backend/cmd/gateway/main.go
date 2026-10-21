package main

import (
	"github.com/syabanf/white-superapp/white-backend/internal/gateway"
	"github.com/syabanf/white-superapp/white-backend/pkg/app"
)

func main() {
	app.Main(gateway.Spec())
}
