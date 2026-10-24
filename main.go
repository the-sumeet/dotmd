package main

import (
	"embed"
	"log"
	"net/http"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

// main function serves as the application's entry point. It initializes the application, creates a window,
// then runs the application and logs any error that might occur.
func main() {

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	files, err := NewFileService()
	if err != nil {
		log.Fatal(err)
	}

	app := application.New(application.Options{
		Name:        "dotmd",
		Description: "A Markdown notes app",
		Services: []application.Service{
			application.NewService(files),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
			// Serves the data directory, so notes can show the images saved next to them.
			Middleware: func(next http.Handler) http.Handler {
				images := files.handler()
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.HasPrefix(r.URL.Path, FilesPrefix) {
						images.ServeHTTP(w, r)
						return
					}
					next.ServeHTTP(w, r)
				})
			},
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Cmd/Ctrl+W closes the open tab rather than the window; the frontend closes
	// the window itself when no tab is open. The default Close items are role
	// based, so their click handlers can't be overridden: replace them.
	menu := application.DefaultApplicationMenu()
	for _, label := range []string{"Close", "Close Window"} {
		for item := menu.FindByLabel(label); item != nil; item = menu.FindByLabel(label) {
			menu.RemoveMenuItem(item)
		}
	}
	file := menu.FindByLabel("File")
	if file == nil {
		log.Println("default menu has no File menu; adding one for Close Tab")
		menu.AddSubmenu("File")
		file = menu.FindByLabel("File")
	}
	file.GetSubmenu().Add("Close Tab").
		SetAccelerator("CmdOrCtrl+w").
		OnClick(func(*application.Context) {
			app.Event.Emit("tab:close")
		})
	app.Menu.SetApplicationMenu(menu)

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "dotmd",
		// Window sized to the golden ratio (1000 / 618 ≈ 1.618).
		Width:  1000,
		Height: 618,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(6, 7, 15),
		URL:              "/",
	})

	// Run the application. This blocks until the application has been exited.
	err = app.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
