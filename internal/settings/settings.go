// Package settings keeps the user's preferences in a JSON file in the OS
// config folder and exposes them to the frontend.
//
// The notes folder is read before the app starts (see [Load]), so it is
// owned here rather than by the frontend; the theme and layout live here too
// so every setting survives clearing the webview's storage.
package settings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"dotmd/internal/notes"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Themes accepted by [Service.SetTheme].
const (
	ThemeSystem = "system"
	ThemeLight  = "light"
	ThemeDark   = "dark"
)

// Settings is what the settings file holds.
type Settings struct {
	// NotesFolder is the absolute path of the notes folder; empty means the
	// default (~/Documents/dotmd).
	NotesFolder string `json:"notesFolder"`
	// Theme is "system", "light" or "dark".
	Theme string `json:"theme"`
	// WideLayout lets note text use more of the window's width.
	WideLayout bool `json:"wideLayout"`
}

// Defaults are the settings used when there is no settings file.
func Defaults() Settings {
	return Settings{Theme: ThemeSystem}
}

func validTheme(t string) bool {
	return t == ThemeSystem || t == ThemeLight || t == ThemeDark
}

// DefaultPath is where the settings file lives: dotmd/settings.json in the
// OS config folder.
func DefaultPath() string {
	dir := application.Path(application.PathConfigHome)
	if dir == "" {
		dir, _ = os.UserConfigDir()
	}
	return filepath.Join(dir, "dotmd", "settings.json")
}

// Load reads the settings file at path. A missing or unreadable file, or an
// invalid value in it, gives the defaults for that value: bad settings must
// never stop the app from starting.
func Load(path string) Settings {
	s := Defaults()
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return Defaults()
	}
	if !validTheme(s.Theme) {
		s.Theme = ThemeSystem
	}
	return s
}

// save writes s to path, replacing the file atomically.
func save(path string, s Settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".settings-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Notes is the part of the notes service that settings changes.
type Notes interface {
	Root() string
	SetRoot(path string) error
}

// State is everything the settings dialog shows.
type State struct {
	// NotesFolder is the absolute path of the notes folder in use.
	NotesFolder string `json:"notesFolder"`
	Theme       string `json:"theme"`
	WideLayout  bool   `json:"wideLayout"`
	// DefaultFolder is true while notes are in the default folder
	// (~/Documents/dotmd), the only folder the Welcome note is added to.
	DefaultFolder bool `json:"defaultFolder"`
	// MissingFolder is the saved notes folder when it wasn't found at
	// startup (say, on an unplugged drive), so the default is in use.
	MissingFolder string `json:"missingFolder"`
	// CanChooseFolder is false on iOS and Android, where apps can't pick
	// an arbitrary folder.
	CanChooseFolder bool `json:"canChooseFolder"`
}

// Service is bound to the frontend. Each setter writes the file first and
// only then changes the setting, so a failed write leaves things as they were.
type Service struct {
	mu          sync.Mutex
	path        string
	cur         Settings
	notes       Notes
	defaultRoot string
}

// New returns a Service holding cur, saving to path, that switches notes
// folders through n.
func New(path string, cur Settings, n Notes) *Service {
	return &Service{path: path, cur: cur, notes: n, defaultRoot: notes.DefaultRoot()}
}

// Get returns the settings and the notes folder in use.
func (s *Service) Get() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state()
}

func (s *Service) state() State {
	root := s.notes.Root()
	st := State{
		NotesFolder:     root,
		Theme:           s.cur.Theme,
		WideLayout:      s.cur.WideLayout,
		DefaultFolder:   root == s.defaultRoot,
		CanChooseFolder: canChooseFolder(),
	}
	if s.cur.NotesFolder != "" && s.cur.NotesFolder != root {
		st.MissingFolder = s.cur.NotesFolder
	}
	return st
}

// SetTheme saves the theme: "system", "light" or "dark".
func (s *Service) SetTheme(theme string) error {
	if !validTheme(theme) {
		return errors.New("unknown theme: " + theme)
	}
	return s.update(func(c *Settings) { c.Theme = theme })
}

// SetWideLayout saves whether note text uses more of the window's width.
func (s *Service) SetWideLayout(on bool) error {
	return s.update(func(c *Settings) { c.WideLayout = on })
}

func canChooseFolder() bool {
	return runtime.GOOS != "ios" && runtime.GOOS != "android"
}

// ChooseNotesFolder asks the user for a new notes folder and switches to it.
// It returns the new state, or null if the user cancelled.
func (s *Service) ChooseNotesFolder() (*State, error) {
	if !canChooseFolder() {
		return nil, errors.New("the notes folder can't be changed on this device")
	}
	path, err := application.Get().Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		CanChooseDirectories: true,
		CanChooseFiles:       false,
		CanCreateDirectories: true,
		Title:                "Choose notes folder",
		ButtonText:           "Choose",
		Directory:            s.notes.Root(),
	}).PromptForSingleSelection()
	if err != nil || path == "" {
		return nil, err
	}
	return s.setNotesFolder(path)
}

// setNotesFolder switches to the folder at path (SetRoot checks it is one)
// and saves it, switching back if it can't be saved.
func (s *Service) setNotesFolder(path string) (*State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.notes.Root()
	if err := s.notes.SetRoot(path); err != nil {
		return nil, err
	}
	next := s.cur
	next.NotesFolder = s.notes.Root()
	if err := save(s.path, next); err != nil {
		_ = s.notes.SetRoot(old)
		return nil, err
	}
	s.cur = next
	st := s.state()
	return &st, nil
}

func (s *Service) update(change func(*Settings)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.cur
	change(&next)
	if err := save(s.path, next); err != nil {
		return err
	}
	s.cur = next
	return nil
}
