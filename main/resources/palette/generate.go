package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
)

var (
	pickerH = 511
	pickerW = 1536
)

func main() {
	// all parts share a common rectangle and starting point
	p := image.Pt(0, 0)
	r := image.Rect(0, 0, pickerW, pickerH)
	bgrad := image.NewRGBA(r)  // bottom gradient
	tgrad := image.NewRGBA(r)  // top gradient mask
	colors := image.NewRGBA(r) // colors
	screen := image.NewRGBA(r) // final image
	wimg := image.NewRGBA(r)   // white background
	// y controls the brightness
	for y := 0; y < pickerH; y++ {
		var bcolor, tcolor *color.RGBA
		// set gradient
		if pickerH/2 <= y {
			bcolor = &color.RGBA{R: 0, G: 0, B: 0, A: uint8(y) - 255}
			tcolor = &color.RGBA{R: 0, G: 0, B: 0, A: 255}
		} else {
			bcolor = &color.RGBA{R: 0, G: 0, B: 0, A: 0}
			tcolor = &color.RGBA{R: 0, G: 0, B: 0, A: uint8(y)}
		}
		// x controls the color
		for x := 0; x < pickerW; x++ {
			// create the initial color
			var c *color.RGBA
			// calculate the stops
			switch {
			// red to yellow (FF0000 to FFFF00)
			case x < 256:
				c = &color.RGBA{R: 255, G: uint8(x), B: uint8(0), A: 255}
			// yellow to green (FFFF00 to 00FF00)
			case x < 512:
				c = &color.RGBA{R: 255 - uint8(x), G: 255, B: 0, A: 255}
			// green to cyan (00FF00 to 00FFFF)
			case x < 768:
				c = &color.RGBA{R: 0, G: 255, B: uint8(x), A: 255}
			// cyan to blue (00FFFF to 0000FF)
			case x < 1024:
				c = &color.RGBA{R: 0, G: 255 - uint8(x), B: 255, A: 255}
			// blue to pink (0000FF to FF00FF)
			case x < 1280:
				c = &color.RGBA{R: uint8(x), G: 0, B: 255, A: 255}
			// violet to red
			default:
				c = &color.RGBA{R: 255, G: 0, B: 255 - uint8(x), A: 255}
			}
			// set color
			colors.Set(x, y, c)
			bgrad.Set(x, y, bcolor)
			tgrad.Set(x, y, tcolor)
			wimg.Set(x, y, color.White)
		}
	}

	// fill in the screen with white
	draw.Draw(screen, r, wimg, p, draw.Over)
	// draw the colors, masked at the top so white shows through
	draw.DrawMask(screen, r, colors, p, tgrad, p, draw.Over)
	// draw the bottom gradient on top for black
	draw.Draw(screen, r, bgrad, p, draw.Over)

	// export the screen
	f, err := os.OpenFile("../images/palette.png", os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	png.Encode(f, screen)
}
