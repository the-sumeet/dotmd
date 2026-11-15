//go:build windows && !386 && !arm

package notes

// shFileOpStruct mirrors SHFILEOPSTRUCTW on 64-bit Windows, where it has the
// natural alignment Go also uses.
type shFileOpStruct struct {
	hwnd                  uintptr
	wFunc                 uint32
	pFrom                 *uint16
	pTo                   *uint16
	fFlags                uint16
	fAnyOperationsAborted int32
	hNameMappings         uintptr
	lpszProgressTitle     *uint16
}

func newSHFileOp(fn uint32, from *uint16, flags uint16) *shFileOpStruct {
	return &shFileOpStruct{wFunc: fn, pFrom: from, fFlags: flags}
}

func (op *shFileOpStruct) aborted() bool { return op.fAnyOperationsAborted != 0 }
