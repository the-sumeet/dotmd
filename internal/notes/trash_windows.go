package notes

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

var procSHFileOperationW = syscall.NewLazyDLL("shell32.dll").NewProc("SHFileOperationW")

const (
	foDelete          = 0x0003
	fofSilent         = 0x0004
	fofNoConfirmation = 0x0010
	fofAllowUndo      = 0x0040
	fofNoErrorUI      = 0x0400
	// Ask before deleting outright where there is no Recycle Bin (network
	// shares, some removable drives): otherwise FOF_ALLOWUNDO quietly
	// deletes for good there.
	fofWantNukeWarning = 0x4000
)

// trash moves abs to the Recycle Bin.
func trash(abs string) error {
	from, err := syscall.UTF16FromString(abs)
	if err != nil {
		return err
	}
	from = append(from, 0) // the list of paths ends with a second NUL
	op := newSHFileOp(foDelete, &from[0],
		fofAllowUndo|fofNoConfirmation|fofSilent|fofNoErrorUI|fofWantNukeWarning)
	ret, _, _ := procSHFileOperationW.Call(uintptr(unsafe.Pointer(op)))
	if op.aborted() {
		return errors.New("not deleted: it couldn't go to the Recycle Bin")
	}
	if ret != 0 {
		return fmt.Errorf("couldn't move to the Recycle Bin (error 0x%x)", ret)
	}
	return nil
}
