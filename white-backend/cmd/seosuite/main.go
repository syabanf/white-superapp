package main

import (
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite"
	"github.com/syabanf/white-superapp/white-backend/pkg/app"
)

func main() {
	app.Main(seosuite.Spec())
}
