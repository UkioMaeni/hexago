package gfx

import (
	"strconv"
	"strings"
)

type Color struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

func ColorFromHex(hex string) Color {

	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return Color{R: 0, G: 0, B: 0, A: 255}
	}
	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return Color{R: 0, G: 0, B: 0, A: 255}
	}
	r := uint8(v >> 16)
	g := uint8(v >> 8)
	b := uint8(v)
	a := uint8(255)
	return Color{
		R: r,
		G: g,
		B: b,
		A: a,
	}
}
