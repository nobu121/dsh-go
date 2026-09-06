//go:build darwin

package main

import "C"

//export dshGoThemeDetected
func dshGoThemeDetected(dark C.int) {
	go onThemeDetected(dark != 0)
}
