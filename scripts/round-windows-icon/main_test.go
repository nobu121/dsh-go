package main

import (
	"image"
	"image/color"
	"testing"
)

func TestApplyRoundCornersTransparent(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			src.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
		}
	}
	out := applyRound(src, 0.22)
	if out.RGBAAt(0, 0).A != 0 {
		t.Fatalf("corner alpha = %d, want 0", out.RGBAAt(0, 0).A)
	}
	if out.RGBAAt(32, 0).A != 255 {
		t.Fatalf("top-edge alpha = %d, want 255", out.RGBAAt(32, 0).A)
	}
	if out.RGBAAt(32, 32).A != 255 {
		t.Fatalf("center alpha = %d, want 255", out.RGBAAt(32, 32).A)
	}
}
