package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/jsvm"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/hook"
)

func main() {
	app := pocketbase.New()

	// ---------------------------------------------------------------
	// Optional plugin flags:
	// ---------------------------------------------------------------

	var hooksDir string
	app.RootCmd.PersistentFlags().StringVar(&hooksDir, "hooksDir", "./pb_hooks", "the directory with the JS app hooks")

	var hooksWatch bool
	app.RootCmd.PersistentFlags().BoolVar(&hooksWatch, "hooksWatch", true, "auto restart the app on pb_hooks file change")

	var hooksPool int
	app.RootCmd.PersistentFlags().IntVar(&hooksPool, "hooksPool", 15, "the total prewarm goja.Runtime instances for the JS app hooks execution")

	var migrationsDir string
	app.RootCmd.PersistentFlags().StringVar(&migrationsDir, "migrationsDir", "./pb_migrations", "the directory with the user defined migrations")

	var automigrate bool
	app.RootCmd.PersistentFlags().BoolVar(&automigrate, "automigrate", true, "enable/disable auto migrations")

	var publicDir string
	app.RootCmd.PersistentFlags().StringVar(&publicDir, "publicDir", defaultPublicDir(), "the directory to serve static files")

	var indexFallback bool
	app.RootCmd.PersistentFlags().BoolVar(&indexFallback, "indexFallback", true, "fallback the request to index.html on missing static path (eg. when pretty urls are used with SPA)")

	app.RootCmd.ParseFlags(os.Args[1:])

	// ---------------------------------------------------------------
	// Plugins and hooks:
	// ---------------------------------------------------------------

	// load jsvm (pb_hooks and pb_migrations)
	jsvm.MustRegister(app, jsvm.Config{
		MigrationsDir: migrationsDir,
		HooksDir:      hooksDir,
		HooksWatch:    hooksWatch,
		HooksPoolSize: hooksPool,
	})

	// migrate command (with js templates)
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		TemplateLang: migratecmd.TemplateLangJS,
		Automigrate:  automigrate,
		Dir:          migrationsDir,
	})

	// static route to serves files from the provided public dir
	// (if publicDir exists and the route path is not already defined)
	app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(e *core.ServeEvent) error {
			if !e.Router.HasRoute(http.MethodGet, "/{path...}") {
				e.Router.GET("/{path...}", apis.Static(os.DirFS(publicDir), indexFallback))
			}

			return e.Next()
		},
		Priority: 999, // execute as latest as possible to allow users to provide their own route
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/api/mmd-hook", func(e *core.RequestEvent) error {
			MmdHookSecret, ok := os.LookupEnv("MMD_HOOK_SECRET")
			if !ok {
				return e.InternalServerError("MMD_HOOK_SECRET not set", nil)
			}

			if e.Request.Header.Get("MMD-Signature") != MmdHookSecret {
				return e.BadRequestError("Invalid signature", nil)
			}

			info, err := e.RequestInfo()
			if err != nil {
				return e.BadRequestError("Failed to parse request", err)
			}

			eventType, ok := info.Body["eventType"].(string)
			if !ok {
				return e.BadRequestError("Failed to parse request body", err)
			}

			if eventType != "donation_completed" {
				return e.NoContent(http.StatusOK)
			}

			data := struct {
				Id        string `json:"id"`
				EventType string `json:"eventType"`
				LiveMode  bool   `json:"liveMode"`
				Data      struct {
					Donation struct {
						Id         string `json:"id"`
						Amount     int64  `json:"amount"`
						TipAmount  int64  `json:"tipAmount"`
						Currency   string `json:"currency"`
						LiveMode   bool   `json:"liveMode"`
						Dedication string `json:"dedication"`
						Message    string `json:"message"`
						Anonymous  bool   `json:"anonymous"`
						CreatedAt  string `json:"createdAt"`
					} `json:"donation"`
				} `json:"data"`
				CreatedAt string `json:"createdAt"`
			}{}

			if err := e.BindBody(&data); err != nil {
				return e.BadRequestError("Failed to read request data", err)
			}

			collection, err := app.FindCollectionByNameOrId("donations")
			if err != nil {
				return e.InternalServerError("Failed to find collection", err)
			}

			record := core.NewRecord(collection)
			record.Set("username", data.Data.Donation.Dedication)
			record.Set("message", data.Data.Donation.Message)
			record.Set("amount", (data.Data.Donation.Amount+data.Data.Donation.TipAmount)/100)
			record.Set("status", "pending_review")
			err = app.Save(record)
			if err != nil {
				return e.InternalServerError("Failed to save record", err)
			}

			return e.NoContent(http.StatusOK)
		})

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// the default pb_public dir location is relative to the executable
func defaultPublicDir() string {
	if strings.HasPrefix(os.Args[0], os.TempDir()) {
		// most likely ran with go run
		return "./pb_public"
	}

	return filepath.Join(os.Args[0], "../pb_public")
}
