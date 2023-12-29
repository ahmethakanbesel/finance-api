package main

import (
	"log"
	"os"
	"strings"

	_ "github.com/ahmethakanbesel/finance-api/migrations"
	"github.com/ahmethakanbesel/finance-api/tefas"
	"github.com/ahmethakanbesel/finance-api/yahoo"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

func main() {
	app := pocketbase.New()

	// loosely check if it was executed using "go run"
	isGoRun := strings.HasPrefix(os.Args[0], os.TempDir())

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// enable auto creation of migration files when making collection changes in the Admin UI
		// (the isGoRun check is to enable it only during development)
		Automigrate: isGoRun,
	})

	// Setup tefas service
	tefasScraper := tefas.NewScraper(
		tefas.WithWorkers(5),
	)
	tefasService := tefas.NewService(app, tefasScraper)
	tefasApi := tefas.NewApi(tefasService, app)

	// Setup yahoo finance service
	yahooScraper := yahoo.NewScraper(
		yahoo.WithWorkers(5),
	)
	yahooService := yahoo.NewService(app, yahooScraper)
	yahooApi := yahoo.NewApi(yahooService, app)

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		tefasApi.SetupRoutes(e.Router.Group("/api/v1"))
		yahooApi.SetupRoutes(e.Router.Group("/api/v1"))
		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
