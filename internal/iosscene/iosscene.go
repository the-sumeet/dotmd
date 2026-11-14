//go:build ios

package iosscene

// Importing "C" makes cgo compile scene_delegate.m into the Go archive that
// the iOS app links (-force_load / -ObjC), so UIKit can find
// DotmdSceneDelegate by the class name given in Info.plist.

// #cgo LDFLAGS: -framework UIKit
import "C"
