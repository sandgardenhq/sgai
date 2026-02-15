//go:build darwin

package notify

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework UserNotifications

#include <stdlib.h>

extern void SendNativeNotification(const char *title, const char *message);
*/
import "C"

import "unsafe"

func sendLocal(title, message string) error {
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))
	cMessage := C.CString(message)
	defer C.free(unsafe.Pointer(cMessage))
	C.SendNativeNotification(cTitle, cMessage)
	return nil
}
