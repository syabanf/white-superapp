package main

import (
	"context"
	"fmt"
	"os"

	"github.com/syabanf/white-superapp/white-backend/internal/performance"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/seed"
	"github.com/syabanf/white-superapp/white-backend/pkg/app"
)

// serve | migrate | worker come from pkg/app; seed inserts the Mixist demo data.
func main() {
	if len(os.Args) > 1 && os.Args[1] == "seed" {
		if err := seed.Run(context.Background()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	app.Main(performance.Spec())
}
