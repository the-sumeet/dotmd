package main

import (
	"embed"
	"log"

	"dotmd/internal/lifecycle"
	"dotmd/internal/notes"
	"dotmd/internal/settings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// The built frontend (frontend/dist) is embedded into the binary.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	life := lifecycle.New()
	settingsPath := settings.DefaultPath()
	prefs := settings.Load(settingsPath)
	notesSvc := notes.New()
	if prefs.NotesFolder != "" {
		// A missing folder (say, on an unplugged drive) is not recreated; the
		// default is used and the setting kept, so the folder is used again
		// once it is back. The frontend says so (State.MissingFolder).
		if err := notesSvc.SetRoot(prefs.NotesFolder); err != nil {
			log.Printf("notes folder: %v; using the default", err)
		}
	}
	settingsSvc := settings.New(settingsPath, prefs, notesSvc)

	app := application.New(application.Options{
		Name:        "dotmd",
		Description: "Plain markdown notes",
		Services: []application.Service{
			application.NewService(notesSvc),
			application.NewService(life),
			application.NewService(settingsSvc),
		},
		// Quitting waits for the frontend to save pending edits.
		ShouldQuit: lifecycle.ShouldQuit(life),
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
			// Lets notes show images stored beside them.
			Middleware: notes.FileMiddleware(notesSvc),
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
