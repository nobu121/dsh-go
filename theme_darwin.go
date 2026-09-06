//go:build darwin

package main

import (
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
)

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework WebKit
#include <stdlib.h>
#import <AppKit/AppKit.h>
#import <WebKit/WebKit.h>

extern void dshGoThemeDetected(int dark);

void dshSetWindowAppearance(void *nsWindow, const char *name) {
	if (nsWindow == NULL || name == NULL) {
		return;
	}
	NSWindow *window = (__bridge NSWindow *)nsWindow;
	NSString *appearanceName = [NSString stringWithUTF8String:name];
	NSAppearance *appearance = [NSAppearance appearanceNamed:appearanceName];
	dispatch_async(dispatch_get_main_queue(), ^{
		[window setTitlebarAppearsTransparent:YES];
		[window setTitleVisibility:NSWindowTitleHidden];
		[window setStyleMask:[window styleMask] | NSWindowStyleMaskFullSizeContentView];
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

void dshPollWindowTheme(void *nsWindow, const char *js) {
	if (nsWindow == NULL || js == NULL) {
		return;
	}
	NSWindow *window = (__bridge NSWindow *)nsWindow;
	WKWebView *webView = [window valueForKey:@"webView"];
	if (webView == nil) {
		return;
	}
	NSString *script = [NSString stringWithUTF8String:js];
	dispatch_async(dispatch_get_main_queue(), ^{
		[webView evaluateJavaScript:script completionHandler:^(id result, NSError *error) {
			if (error != nil || result == nil) {
				return;
			}
			dshGoThemeDetected([result boolValue] ? 1 : 0);
		}];
	});
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

func pollWindowTheme(win *application.WebviewWindow) {
	ns := win.NativeWindow()
	if ns == nil {
		return
	}
	cjs := C.CString(themeQueryJS)
	C.dshPollWindowTheme(ns, cjs)
	C.free(unsafe.Pointer(cjs))
}
