package main

import (
	"embed"
	"log"

	"dotmd/internal/lifecycle"
	"dotmd/internal/notes"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// The built frontend (frontend/dist) is embedded into the binary.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	life := lifecycle.New()

	app := application.New(application.Options{
		Name:        "dotmd",
		Description: "Plain markdown notes",
		Services: []application.Service{
			application.NewService(notes.New()),
			application.NewService(life),
		},
		// Quitting waits for the frontend to save pending edits.
		ShouldQuit: lifecycle.ShouldQuit(life),
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "dotmd",
		Width:  1000,
		Height: 700,
		Mac: application.MacWindow{
			// Window dragging is handled in the frontend via `--wails-draggable`
			// on the header, so interactive controls there stay clickable.
			Backdrop: application.MacBackdropTranslucent,
			TitleBar: application.MacTitleBarHiddenInset,
		},
		URL: "/",
	})
	lifecycle.Attach(life, app, window)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
