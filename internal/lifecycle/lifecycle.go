// Package lifecycle holds back quitting the app until the frontend has saved
// any pending edits, so text typed just before closing is not lost.
//
// Closing the window or quitting is cancelled; the frontend is sent
// FlushEvent, calls [Service.Flushing] to say it is saving, and then calls
// [Service.Saved], which quits for real, or [Service.Unsaved] if the save
// failed, which keeps the app open so the error can be seen. Once a later save
// succeeds, the frontend calls [Service.Recovered].
//
// Quitting must never get stuck: if the frontend does not answer within
// answerTimeout, or its save takes longer than saveTimeout, the app quits
// anyway, and once a save on quit has failed, the next failure quits too
// (the user has been told and chose to quit again), unless the edits have
// been saved since: then a new failure is a new problem and warns again.
package lifecycle

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// FlushEvent asks the frontend to save pending edits and report back.
const FlushEvent = "dotmd:flush"

const (
	answerTimeout = 3 * time.Second  // frontend never picked up FlushEvent
	saveTimeout   = 30 * time.Second // frontend is saving but never finished
)

// Service is bound to the frontend; its methods are Flushing, Saved, Unsaved
// and Recovered.
type Service struct {
	app       *application.App
	quitting  atomic.Bool // set once quitting is allowed
	requested atomic.Bool // set while the frontend is being asked to flush
	failed    atomic.Bool // a save on quit failed and nothing has been saved since

	mu    sync.Mutex
	timer *time.Timer // quits if the frontend stops answering
}

// New returns a Service. Pass ShouldQuit(s) in the application options and
// call Attach once the app and its window exist.
func New() *Service {
	return &Service{}
}

// Flushing tells the app the frontend got FlushEvent and is saving; a slow
// save then gets saveTimeout instead of answerTimeout before the app gives up.
func (s *Service) Flushing() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.timer != nil {
		s.timer.Reset(saveTimeout)
	}
}

// Saved tells the app that pending edits are on disk; the app then quits.
func (s *Service) Saved() {
	s.quit()
}

// Unsaved tells the app that pending edits could not be written. The first
// time, the quit is called off so the user sees the error and keeps the
// edits; if it happens again on a later quit, with no successful save in
// between (see Recovered), the app quits anyway.
func (s *Service) Unsaved() {
	if s.failed.Swap(true) {
		s.quit()
		return
	}
	s.mu.Lock()
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	s.mu.Unlock()
	s.requested.Store(false)
}

// Recovered tells the app that edits are being saved again after a save on
// quit failed, so the next failure on quit warns again instead of quitting.
func (s *Service) Recovered() {
	s.failed.Store(false)
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
	s.mu.Lock()
	s.timer = time.AfterFunc(answerTimeout, s.quit)
	s.mu.Unlock()
	s.app.Event.Emit(FlushEvent)
}

func (s *Service) quit() {
	if s.quitting.Swap(true) || s.app == nil {
		return
	}
	s.app.Quit()
}
