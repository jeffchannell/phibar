package main

import (
	"image/color"
)

// colorPoint objects hold data about each colored stop
type colorPoint struct {
	color                  color.RGBA // the color
	cx, cy, px, py         float64    // color and value offsets
	xmin, xmax, ymin, ymax float64    // coordinate limits
}

// negative color from this stop
func (s *colorPoint) negative() color.Color {
	return color.RGBA{255 - s.color.R, 255 - s.color.G, 255 - s.color.B, s.color.A}
}

func (s *colorPoint) point(x, y float64) {
	// x offsets
	s.cx = x
	s.px = calcColorPointValue(x, s.xmin, s.xmax)
	// y offsets
	s.cy = y
	s.py = calcColorPointValue(y, s.ymin, s.ymax)
}

func calcColorPointValue(f, min, max float64) float64 {
	v := f
	if v < min {
		for v < min {
			v = max + v
		}
	} else if v > max {
		for v > max {
			v -= max
		}
	}
	return v
}
