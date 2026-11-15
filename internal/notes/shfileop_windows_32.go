//go:build windows && (386 || arm)

package notes

import "encoding/binary"

// shFileOpStruct mirrors SHFILEOPSTRUCTW on 32-bit Windows, where shellapi.h
// packs it to 1 byte: the BOOL after the WORD flags sits at offset 18, which
// Go would pad to 20. Byte arrays (alignment 1) keep the C offsets.
type shFileOpStruct struct {
	hwnd                  uintptr
	wFunc                 uint32
	pFrom                 *uint16
	pTo                   *uint16
	fFlags                uint16
	fAnyOperationsAborted [4]byte
	hNameMappings         [4]byte
	lpszProgressTitle     [4]byte
}

func newSHFileOp(fn uint32, from *uint16, flags uint16) *shFileOpStruct {
	return &shFileOpStruct{wFunc: fn, pFrom: from, fFlags: flags}
}

func (op *shFileOpStruct) aborted() bool {
	return binary.LittleEndian.Uint32(op.fAnyOperationsAborted[:]) != 0
}
