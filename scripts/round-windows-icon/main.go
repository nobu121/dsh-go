package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

// Windows 11–like corner radius relative to the shortest side.
const cornerRatio = 0.22

var icoSizes = []int{256, 128, 64, 48, 32, 16}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <src.png> <dest.ico>\n", filepath.Base(os.Args[0]))
		os.Exit(2)
	}
	if err := generate(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(srcPath, icoPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	src, err := png.Decode(srcFile)
	if err != nil {
		return fmt.Errorf("decode %s: %w", srcPath, err)
	}

	rounded := applyRound(src, cornerRatio)
	images := make([]image.Image, 0, len(icoSizes))
	for _, size := range icoSizes {
		images = append(images, resizeArea(rounded, size, size))
	}
	return writeICO(icoPath, images)
}

func applyRound(src image.Image, ratio float64) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	radius := float64(min(w, h)) * ratio
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			cr, cg, cb, ca := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			alpha := uint8(float64(ca>>8) * coverage(float64(x)+0.5, float64(y)+0.5, float64(w), float64(h), radius))
			out.SetRGBA(x, y, color.RGBA{R: uint8(cr >> 8), G: uint8(cg >> 8), B: uint8(cb >> 8), A: alpha})
		}
	}
	return out
}

func coverage(px, py, w, h, r float64) float64 {
	if r < 1 {
		return 1
	}
	cx := clamp(px, r, w-r)
	cy := clamp(py, r, h-r)
	d := math.Hypot(px-cx, py-cy)
	return clamp(r+0.5-d, 0, 1)
}

func resizeArea(src *image.RGBA, nw, nh int) *image.RGBA {
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		for x := 0; x < nw; x++ {
			x0 := x * sw / nw
			x1 := (x + 1) * sw / nw
			y0 := y * sh / nh
			y1 := (y + 1) * sh / nh
			if x1 <= x0 {
				x1 = x0 + 1
			}
			if y1 <= y0 {
				y1 = y0 + 1
			}
			var r, g, b, a, n uint32
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					c := src.RGBAAt(sx, sy)
					r += uint32(c.R)
					g += uint32(c.G)
					b += uint32(c.B)
					a += uint32(c.A)
					n++
				}
			}
			dst.SetRGBA(x, y, color.RGBA{R: uint8(r / n), G: uint8(g / n), B: uint8(b / n), A: uint8(a / n)})
		}
	}
	return dst
}

func writeICO(path string, images []image.Image) error {
	blobs := make([][]byte, len(images))
	for i, im := range images {
		var buf bytes.Buffer
		if err := png.Encode(&buf, im); err != nil {
			return err
		}
		blobs[i] = buf.Bytes()
	}

	var hdr bytes.Buffer
	_ = binary.Write(&hdr, binary.LittleEndian, uint16(0))
	_ = binary.Write(&hdr, binary.LittleEndian, uint16(1))
	_ = binary.Write(&hdr, binary.LittleEndian, uint16(len(blobs)))

	offset := 6 + 16*len(blobs)
	for i, im := range images {
		b := im.Bounds()
		w, h := b.Dx(), b.Dy()
		wb, hb := byte(w), byte(h)
		if w >= 256 {
			wb = 0
		}
		if h >= 256 {
			hb = 0
		}
		hdr.Write([]byte{wb, hb, 0, 0})
		_ = binary.Write(&hdr, binary.LittleEndian, uint16(1))
		_ = binary.Write(&hdr, binary.LittleEndian, uint16(32))
		_ = binary.Write(&hdr, binary.LittleEndian, uint32(len(blobs[i])))
		_ = binary.Write(&hdr, binary.LittleEndian, uint32(offset))
		offset += len(blobs[i])
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(hdr.Bytes()); err != nil {
		return err
	}
	for _, blob := range blobs {
		if _, err := f.Write(blob); err != nil {
			return err
		}
	}
	return nil
}

func clamp(v, lo, hi float64) float64 {
	return math.Min(hi, math.Max(lo, v))
}
