// Package lifecycle holds back quitting the app until the frontend has saved
// any pending edits, so text typed just before closing is not lost.
//
// Closing the window or quitting is cancelled once; the frontend is sent
// FlushEvent, saves, and calls [Service.Saved], which quits for real. If the
// frontend does not answer within flushTimeout the app quits anyway.
package lifecycle

import (
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// FlushEvent asks the frontend to save pending edits and then call Saved.
const FlushEvent = "dotmd:flush"

const flushTimeout = 3 * time.Second

// Service is bound to the frontend; its only method is Saved.
type Service struct {
	app       *application.App
	quitting  atomic.Bool // set once quitting is allowed
	requested atomic.Bool // set once the frontend has been asked to flush
}

// New returns a Service. Pass ShouldQuit(s) in the application options and
// call Attach once the app and its window exist.
func New() *Service {
	return &Service{}
}

// Saved tells the app that pending edits are on disk; the app then quits.
func (s *Service) Saved() {
	s.quit()
}

// ShouldQuit is for application.Options.ShouldQuit: it defers the first quit
// request (e.g. Cmd+Q) until the frontend has saved.
func ShouldQuit(s *Service) func() bool {
	return func() bool {
		if s.quitting.Load() {
			return true
		}
		s.requestFlush()
		return false
	}
}

// Attach defers closing the window the same way.
func Attach(s *Service, app *application.App, window *application.WebviewWindow) {
	s.app = app
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if s.quitting.Load() {
			return
		}
		e.Cancel()
		s.requestFlush()
	})
}

func (s *Service) requestFlush() {
	if s.app == nil {
		// Not attached yet: nothing in the frontend can be unsaved.
		s.quitting.Store(true)
		return
	}
	if !s.requested.CompareAndSwap(false, true) {
		return
	}
	s.app.Event.Emit(FlushEvent)
	time.AfterFunc(flushTimeout, s.quit)
}

func (s *Service) quit() {
	if s.quitting.Swap(true) || s.app == nil {
		return
	}
	s.app.Quit()
}
