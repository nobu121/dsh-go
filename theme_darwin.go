//go:build darwin

package main

import (
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
)

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit
#include <stdlib.h>
#import <AppKit/AppKit.h>

void dshSetWindowAppearance(void *nsWindow, const char *name) {
	if (nsWindow == NULL || name == NULL) {
		return;
	}
	NSWindow *window = (__bridge NSWindow *)nsWindow;
	NSString *appearanceName = [NSString stringWithUTF8String:name];
	NSAppearance *appearance = [NSAppearance appearanceNamed:appearanceName];
	dispatch_async(dispatch_get_main_queue(), ^{
		[window setAppearance:appearance];
		[window invalidateShadow];
		[window displayIfNeeded];
	});
}

bool dshSystemDark(void) {
	NSAppearance *appearance = [NSApp effectiveAppearance];
	if (appearance == nil) {
		appearance = [NSAppearance currentDrawingAppearance];
	}
	NSAppearanceName match = [appearance bestMatchFromAppearancesWithNames:@[
		NSAppearanceNameAqua,
		NSAppearanceNameDarkAqua
	]];
	return [match isEqualToString:NSAppearanceNameDarkAqua];
}
*/
import "C"

func systemDark() bool {
	return bool(C.dshSystemDark())
}

func applyNativeChrome(win *application.WebviewWindow, dark bool) {
	ns := win.NativeWindow()
	if ns == nil {
		return
	}
	name := "NSAppearanceNameAqua"
	if dark {
		name = "NSAppearanceNameDarkAqua"
	}
	cname := C.CString(name)
	C.dshSetWindowAppearance(ns, cname)
	C.free(unsafe.Pointer(cname))
}
