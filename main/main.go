package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"

	resources "github.com/jeffchannell/phibar/main/resources/images"

	"github.com/golang/freetype/truetype"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/inpututil"
	"github.com/jeffchannell/golden"
	"golang.org/x/image/font"
)

// colorStop objects hold data about each colored stop
type colorStop struct {
	color      color.Color // the color
	val        float64     // stop value, different from x (for calculating the others)
	x          float64     // x offset
	c, m, y, k uint8       // CMYK colors
	r, g, b    uint8       // RGB colors
}

// cmyk generates the display string for CMYK colors
func (s *colorStop) cmyk() string {
	return fmt.Sprintf("cmyk(%d, %d, %d, %d)", s.c, s.m, s.y, s.k)
}

// hex generates the display string for hexadecimal codes
func (s *colorStop) hex() string {
	return fmt.Sprintf("#%02X%02X%02X", s.r, s.g, s.b)
}

// negative color from this stop
func (s *colorStop) negative() color.Color {
	return color.RGBA{255 - s.r, 255 - s.g, 255 - s.b, 255}
}

// rgb generates the display string for RGB colors
func (s *colorStop) rgb() string {
	return fmt.Sprintf("rgb(%d, %d, %d)", s.r, s.g, s.b)
}

// setColor updates all the different internal values for this stop
func (s *colorStop) setColor(c color.Color) {
	s.color = c
	r, g, b, _ := c.RGBA()
	s.r, s.g, s.b = uint8(r), uint8(g), uint8(b)
	s.c, s.m, s.y, s.k = color.RGBToCMYK(s.r, s.g, s.b)
}

func (s *colorStop) setVal(val float64) {
	s.val = val
	s.x = val
	sW := float64(screenW)
	if s.x < 0 {
		for s.x < 0 {
			s.x = sW + s.x
		}
	} else if s.x > sW {
		for s.x > sW {
			s.x -= sW
		}
	}
}

var (
	windowTitle = "PhiBar"

	colorPrimary   color.Color   // primary color
	colorSecondary color.Color   // secondary color
	colorTertiary  color.Color   // tertiary color
	picker         *ebiten.Image // color picker image
	copy           bool
	dragging       bool
	ctrlDown       bool // ctrl button is down

	outputH = 200
	padding = 20
	pickerH = 511
	pickerW = 1536

	screenH = pickerH + outputH + padding*2
	screenW = pickerW

	primary    = 830
	distance   = -200
	brightness = 230
	step       = 10
	stepmax    = 50
	stepmin    = 1
	stepmod    = 1
	stopmax    = 10
	stopmin    = 3
	stops      = 3
	stoplist   []colorStop

	// arcadeFont font face
	arcadeFont font.Face
	// fontSize sets the base font size
	fontSize float64 = 16
)

func init() {
	stoplist = make([]colorStop, stopmax)
}

func update(screen *ebiten.Image) (e error) {
	px, py := ebiten.CursorPosition()
	wx, wy := ebiten.MouseWheel()
	cursor := image.Pt(px, py)
	b := picker.Bounds()

	var ctrlDown bool

	// fullscreen
	if inpututil.IsKeyJustReleased(ebiten.KeyF) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	// ctrl button
	if inpututil.IsKeyJustPressed(ebiten.KeyControl) {
		ctrlDown = true
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyControl) {
		ctrlDown = false
	}
	// copy
	if inpututil.IsKeyJustReleased(ebiten.KeyC) {
		copy = true
	}
	// change step mod
	if ctrlDown {
		stepmod = 10
	} else {
		stepmod = 1
	}
	// change stops
	if inpututil.IsKeyJustReleased(ebiten.KeyEqual) {
		stops++
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyMinus) {
		stops--
	}
	// keep stops within bounds 2-8)
	if stops > stopmax {
		stops = stopmax
	} else if stops < stopmin {
		stops = stopmin
	}
	// change step
	if inpututil.IsKeyJustReleased(ebiten.KeyRightBracket) {
		step += stepmod
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyLeftBracket) {
		step -= stepmod
	}
	// keep step within bounds 1-50)
	if step > stepmax {
		step = stepmax
	} else if step < stepmin {
		step = stepmin
	}
	// change distance
	if inpututil.IsKeyJustReleased(ebiten.KeyUp) {
		distance += step * stepmod
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyDown) {
		distance -= step * stepmod
	}
	if wx != 0 {
		distance -= int(math.Round(wx)) * step
	}
	// don't let distance be too big
	if distance > screenW {
		distance = screenW
	} else if distance < -screenW {
		distance = -screenW
	}
	// change primary position
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if cursor.In(b) {
			dragging = true
			primary = px
			brightness = py
		} else {
			// TODO click colors to copy
		}
	} else if dragging && inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		dragging = false
		distance = px - primary
	} else if dragging {
		brightness = py
		distance = px - primary
	} else if inpututil.IsKeyJustReleased(ebiten.KeyLeft) {
		primary -= step * stepmod
	} else if inpututil.IsKeyJustReleased(ebiten.KeyRight) {
		primary += step * stepmod
	} else if inpututil.IsKeyJustReleased(ebiten.KeyPageUp) {
		brightness -= step * stepmod
	} else if inpututil.IsKeyJustReleased(ebiten.KeyPageDown) {
		brightness += step * stepmod
	}
	if wy != 0 {
		brightness -= int(math.Round(wy)) * (step * stepmod)
	}
	// don't let brightness be out of bounds
	if brightness < 0 {
		brightness = 0
	} else if brightness >= pickerH {
		brightness = pickerH - 1
	}
	// don't let position be outside the bounds of the picker
	if primary < 0 {
		primary += screenW
	} else if primary > screenW {
		primary -= screenW
	}

	for i := range stoplist {
		if i == stops {
			break
		}
		switch i {
		case 0:
			stoplist[i].setVal(float64(primary))
		case 1:
			stoplist[i].setVal(float64(primary + distance))
		default:
			stoplist[i].setVal(golden.Next(stoplist[i-2].val, stoplist[i-1].val))
		}
	}

	// calculate golden ratio offsets
	var d0, d1, d2, sW float64
	sW = float64(screenW)
	d0 = float64(primary)
	d1 = float64(primary + distance)
	d2 = golden.Next(d0, d1)
	if d0 < 0 {
		d0 = sW + d0
	} else if d0 > sW {
		d0 -= sW
	}
	if d1 < 0 {
		d1 = sW + d1
	} else if d1 > sW {
		d1 -= sW
	}
	if d2 < 0 {
		d2 = sW + d2
	} else if d2 > sW {
		d2 -= sW
	}

	// if an error occurred or we don't need to draw, there's nothing left to do
	if (e != nil) || ebiten.IsDrawingSkipped() {
		return
	}

	// app background color
	screen.Fill(color.RGBA{0x33, 0x33, 0x33, 0xff})

	// draw raw picker colors
	op := &ebiten.DrawImageOptions{}
	op.SourceRect = &b
	screen.DrawImage(picker, op)

	// set colors in a loop by itself
	// this is done to prevent the drawing from pulling colors from the guides
	for i := range stoplist {
		if i == stops {
			break
		}
		stoplist[i].setColor(screen.At(int(math.Round(stoplist[i].x)), brightness))
	}

	// // draw color boxes
	// colorPrimary = screen.At(int(math.Round(d0)), brightness)
	// colorSecondary = screen.At(int(math.Round(d1)), brightness)
	// colorTertiary = screen.At(int(math.Round(d2)), brightness)

	// // convert colors to CMYK
	// cpR32, cpG32, cpB32, _ := colorPrimary.RGBA()
	// cpR, cpG, cpB := uint8(cpR32), uint8(cpG32), uint8(cpB32)
	// cpC, cpM, cpY, cpK := color.RGBToCMYK(cpR, cpG, cpB)

	// csR32, csG32, csB32, _ := colorSecondary.RGBA()
	// csR, csG, csB := uint8(csR32), uint8(csG32), uint8(csB32)
	// csC, csM, csY, csK := color.RGBToCMYK(csR, csG, csB)

	// ctR32, ctG32, ctB32, _ := colorTertiary.RGBA()
	// ctR, ctG, ctB := uint8(ctR32), uint8(ctG32), uint8(ctB32)
	// ctC, ctM, ctY, ctK := color.RGBToCMYK(ctR, ctG, ctB)

	// cpcontrast := color.RGBA{255 - cpR, 255 - cpG, 255 - cpB, 255}
	// cscontrast := color.RGBA{255 - csR, 255 - csG, 255 - csB, 255}
	// ctcontrast := color.RGBA{255 - ctR, 255 - ctG, 255 - ctB, 255}

	// // create the color texts
	// colorPrimaryCMYK := fmt.Sprintf("cmyk(%d, %d, %d, %d)", cpC, cpM, cpY, cpK)
	// colorPrimaryHex := fmt.Sprintf("#%02X%02X%02X", cpR, cpG, cpB)
	// colorPrimaryRGB := fmt.Sprintf("rgb(%d, %d, %d)", cpR, cpG, cpB)
	// colorSecondaryCMYK := fmt.Sprintf("cmyk(%d, %d, %d, %d)", csC, csM, csY, csK)
	// colorSecondaryHex := fmt.Sprintf("#%02X%02X%02X", csR, csG, csB)
	// colorSecondaryRGB := fmt.Sprintf("rgb(%d, %d, %d)", csR, csG, csB)
	// colorTertiaryCMYK := fmt.Sprintf("cmyk(%d, %d, %d, %d)", ctC, ctM, ctY, ctK)
	// colorTertiaryHex := fmt.Sprintf("#%02X%02X%02X", ctR, ctG, ctB)
	// colorTertiaryRGB := fmt.Sprintf("rgb(%d, %d, %d)", ctR, ctG, ctB)

	// cbounds := image.Rect(0, 0, int((screenW-padding*4)/3), outputH)
	// cminy := pickerH + padding
	// cmaxy := cminy + outputH
	// cpbounds := image.Rect(padding, cminy, padding+cbounds.Max.X, cmaxy)
	// csbounds := image.Rect(padding+cpbounds.Max.X, cminy, padding+cpbounds.Max.X+cbounds.Max.X, cmaxy)
	// ctbounds := image.Rect(padding+csbounds.Max.X, cminy, padding+csbounds.Max.X+cbounds.Max.X, cmaxy)
	// cpop := &ebiten.DrawImageOptions{}
	// csop := &ebiten.DrawImageOptions{}
	// ctop := &ebiten.DrawImageOptions{}
	// cpop.SourceRect = &cbounds
	// csop.SourceRect = &cbounds
	// ctop.SourceRect = &cbounds
	// cpimg, _ := ebiten.NewImage(cbounds.Max.X, cbounds.Max.Y, ebiten.FilterDefault)
	// csimg, _ := ebiten.NewImage(cbounds.Max.X, cbounds.Max.Y, ebiten.FilterDefault)
	// ctimg, _ := ebiten.NewImage(cbounds.Max.X, cbounds.Max.Y, ebiten.FilterDefault)
	// cpimg.Fill(colorPrimary)
	// csimg.Fill(colorSecondary)
	// ctimg.Fill(colorTertiary)

	// if copy && ctrlDown {
	// 	var clipdata string
	// 	switch {
	// 	case cursor.In(ctbounds):
	// 		clipdata = colorTertiaryHex
	// 	case cursor.In(csbounds):
	// 		clipdata = colorSecondaryHex
	// 	default:
	// 		clipdata = colorPrimaryHex
	// 	}
	// 	if err := clipboard.WriteAll(clipdata); err != nil {
	// 		fmt.Printf("Could not copy '%s' to clipboard: %v\n", clipdata, err)
	// 	}
	// 	copy = false
	// }

	// cpop.GeoM.Translate(float64(cpbounds.Min.X), float64(cpbounds.Min.Y))
	// csop.GeoM.Translate(float64(csbounds.Min.X), float64(csbounds.Min.Y))
	// ctop.GeoM.Translate(float64(ctbounds.Min.X), float64(ctbounds.Min.Y))
	// screen.DrawImage(cpimg, cpop)
	// screen.DrawImage(csimg, csop)
	// screen.DrawImage(ctimg, ctop)

	bright := uint8(brightness / 2)
	// shadow := color.RGBA{255 - bright, 255 - bright, 255 - bright, 255}
	// text.Draw(screen, colorPrimaryRGB, arcadeFont, padding*2+2, pickerH+padding*4+2, shadow)
	// text.Draw(screen, colorPrimaryRGB, arcadeFont, padding*2, pickerH+padding*4, cpcontrast)
	// text.Draw(screen, colorPrimaryHex, arcadeFont, padding*2+2, pickerH+padding*6+2, shadow)
	// text.Draw(screen, colorPrimaryHex, arcadeFont, padding*2, pickerH+padding*6, cpcontrast)
	// text.Draw(screen, colorPrimaryCMYK, arcadeFont, padding*2+2, pickerH+padding*8+2, shadow)
	// text.Draw(screen, colorPrimaryCMYK, arcadeFont, padding*2, pickerH+padding*8, cpcontrast)

	// text.Draw(screen, colorSecondaryRGB, arcadeFont, padding*3+cbounds.Max.X+2, pickerH+padding*4+2, shadow)
	// text.Draw(screen, colorSecondaryRGB, arcadeFont, padding*3+cbounds.Max.X, pickerH+padding*4, cscontrast)
	// text.Draw(screen, colorSecondaryHex, arcadeFont, padding*3+cbounds.Max.X+2, pickerH+padding*6+2, shadow)
	// text.Draw(screen, colorSecondaryHex, arcadeFont, padding*3+cbounds.Max.X, pickerH+padding*6, cscontrast)
	// text.Draw(screen, colorSecondaryCMYK, arcadeFont, padding*3+cbounds.Max.X+2, pickerH+padding*8+2, shadow)
	// text.Draw(screen, colorSecondaryCMYK, arcadeFont, padding*3+cbounds.Max.X, pickerH+padding*8, cscontrast)

	// text.Draw(screen, colorTertiaryRGB, arcadeFont, padding*4+cbounds.Max.X*2+2, pickerH+padding*4+2, shadow)
	// text.Draw(screen, colorTertiaryRGB, arcadeFont, padding*4+cbounds.Max.X*2, pickerH+padding*4, ctcontrast)
	// text.Draw(screen, colorTertiaryHex, arcadeFont, padding*4+cbounds.Max.X*2+2, pickerH+padding*6+2, shadow)
	// text.Draw(screen, colorTertiaryHex, arcadeFont, padding*4+cbounds.Max.X*2, pickerH+padding*6, ctcontrast)
	// text.Draw(screen, colorTertiaryCMYK, arcadeFont, padding*4+cbounds.Max.X*2+2, pickerH+padding*8+2, shadow)
	// text.Draw(screen, colorTertiaryCMYK, arcadeFont, padding*4+cbounds.Max.X*2, pickerH+padding*8, ctcontrast)

	// each selected color box image will share a generic bounds rectangle so they are the same size
	selectedBounds := image.Rect(0, 0, int((screenW-padding*(stops+1))/stops), outputH)
	// each selected color box will share a common minimum and maximum y coordinate
	selectedMinY := pickerH + padding
	selectedMaxY := selectedMinY + outputH
	// draw graphics for each stop
	for i := range stoplist {
		if i == stops {
			break
		}
		// draw the guide line in the negtive color from the value of the stop
		ebitenutil.DrawLine(screen, stoplist[i].x, 0, stoplist[i].x, float64(pickerH), stoplist[i].negative())
		// draw the box that represents this color
		stopOffset := i * (selectedBounds.Max.X + padding)
		stopBounds := image.Rect(padding+stopOffset, selectedMinY, padding+stopOffset+selectedBounds.Max.X, selectedMaxY)
		stopImg, _ := ebiten.NewImage(selectedBounds.Max.X, selectedBounds.Max.Y, ebiten.FilterDefault)
		stopImg.Fill(stoplist[i].color)
		stopOptions := &ebiten.DrawImageOptions{}
		stopOptions.SourceRect = &selectedBounds
		stopOptions.GeoM.Translate(float64(stopBounds.Min.X), float64(stopBounds.Min.Y))
		screen.DrawImage(stopImg, stopOptions)
	}
	// TODO make these fancier
	// ebitenutil.DrawLine(screen, d0, 0, d0, float64(pickerH), color.RGBA{255, 0, 0, 255})
	// ebitenutil.DrawLine(screen, d1, 0, d1, float64(pickerH), color.RGBA{0, 255, 0, 255})
	// ebitenutil.DrawLine(screen, d2, 0, d2, float64(pickerH), color.RGBA{0, 0, 255, 255})
	ebitenutil.DrawLine(screen, 0, float64(brightness), float64(screenW), float64(brightness), color.RGBA{bright, bright, bright, 255})

	// ebitenutil.DrawLine(screen, float64(cpbounds.Min.X), float64(cpbounds.Min.Y), float64(cpbounds.Min.X), float64(cpbounds.Max.Y), cpcontrast)
	// ebitenutil.DrawLine(screen, float64(cpbounds.Max.X), float64(cpbounds.Min.Y), float64(cpbounds.Max.X), float64(cpbounds.Max.Y), cpcontrast)
	// ebitenutil.DrawLine(screen, float64(cpbounds.Min.X), float64(cpbounds.Min.Y), float64(cpbounds.Max.X), float64(cpbounds.Min.Y), cpcontrast)
	// ebitenutil.DrawLine(screen, float64(cpbounds.Min.X), float64(cpbounds.Max.Y), float64(cpbounds.Max.X), float64(cpbounds.Max.Y), cpcontrast)

	// ebitenutil.DrawLine(screen, float64(csbounds.Min.X), float64(csbounds.Min.Y), float64(csbounds.Min.X), float64(csbounds.Max.Y), cscontrast)
	// ebitenutil.DrawLine(screen, float64(csbounds.Max.X), float64(csbounds.Min.Y), float64(csbounds.Max.X), float64(csbounds.Max.Y), cscontrast)
	// ebitenutil.DrawLine(screen, float64(csbounds.Min.X), float64(csbounds.Min.Y), float64(csbounds.Max.X), float64(csbounds.Min.Y), cscontrast)
	// ebitenutil.DrawLine(screen, float64(csbounds.Min.X), float64(csbounds.Max.Y), float64(csbounds.Max.X), float64(csbounds.Max.Y), cscontrast)

	// ebitenutil.DrawLine(screen, float64(ctbounds.Min.X), float64(ctbounds.Min.Y), float64(ctbounds.Min.X), float64(ctbounds.Max.Y), ctcontrast)
	// ebitenutil.DrawLine(screen, float64(ctbounds.Max.X), float64(ctbounds.Min.Y), float64(ctbounds.Max.X), float64(ctbounds.Max.Y), ctcontrast)
	// ebitenutil.DrawLine(screen, float64(ctbounds.Min.X), float64(ctbounds.Min.Y), float64(ctbounds.Max.X), float64(ctbounds.Min.Y), ctcontrast)
	// ebitenutil.DrawLine(screen, float64(ctbounds.Min.X), float64(ctbounds.Max.Y), float64(ctbounds.Max.X), float64(ctbounds.Max.Y), ctcontrast)

	// debug info
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %v TPS: %v\nx: %d, y: %d\nd0: %v d1: %v d2: %v, bright: %v, steps: %d, stops: %d\nc1: %v, c2: %v, c3: %v", ebiten.CurrentFPS(), ebiten.CurrentTPS(), px, py, d0, d1, d2, bright, step, stops, colorPrimary, colorSecondary, colorTertiary))

	return
}

func main() {
	tt, err := truetype.Parse(fonts.ArcadeN_ttf)
	if err != nil {
		log.Fatal(err)
	}
	const dpi = 72
	arcadeFont = truetype.NewFace(tt, &truetype.Options{
		Size:    fontSize,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})

	img, _, err := image.Decode(bytes.NewReader(resources.Palette_png))
	if err != nil {
		log.Fatal(err)
	}
	picker, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)

	if err := ebiten.Run(update, screenW, screenH, 1, windowTitle); err != nil {
		panic(err)
	}
}
