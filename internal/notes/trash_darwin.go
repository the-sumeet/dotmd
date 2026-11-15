package notes

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation
#import <Foundation/Foundation.h>
#include <stdlib.h>

// dotmdTrash moves path to the Trash. It returns NULL on success, or an error
// message the caller must free.
static char *dotmdTrash(const char *path) {
	@autoreleasepool {
		NSURL *url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:path]];
		NSError *err = nil;
		if ([[NSFileManager defaultManager] trashItemAtURL:url resultingItemURL:nil error:&err]) {
			return NULL;
		}
		NSString *msg = err ? err.localizedDescription : @"couldn't move to Trash";
		return strdup(msg.UTF8String);
	}
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

// trash moves abs to the Trash with NSFileManager, which needs no permission
// to control Finder.
func trash(abs string) error {
	p := C.CString(abs)
	defer C.free(unsafe.Pointer(p))
	if msg := C.dotmdTrash(p); msg != nil {
		defer C.free(unsafe.Pointer(msg))
		return errors.New(C.GoString(msg))
	}
	return nil
}
