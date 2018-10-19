package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"

	"github.com/golang/freetype/truetype"
	resources "github.com/jeffchannell/phibar/main/resources/images"
	"golang.org/x/image/font"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/inpututil"
	"github.com/hajimehoshi/ebiten/text"
	"github.com/jeffchannell/golden"
)

var (
	windowTitle = "PhiBar"

	colorPrimary   color.Color   // primary color
	colorSecondary color.Color   // secondary color
	colorTertiary  color.Color   // tertiary color
	picker         *ebiten.Image // color picker image
	dragging       bool

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
	stepmod    = 1

	// arcadeFont font face
	arcadeFont font.Face
	// fontSize sets the base font size
	fontSize float64 = 16
)

func update(screen *ebiten.Image) (e error) {
	px, py := ebiten.CursorPosition()
	wx, wy := ebiten.MouseWheel()

	// fullscreen
	if inpututil.IsKeyJustReleased(ebiten.KeyF) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	// change step mod
	if inpututil.IsKeyJustPressed(ebiten.KeyControl) {
		stepmod = 5
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyControl) {
		stepmod = 1
	}
	// change step
	if inpututil.IsKeyJustReleased(ebiten.KeyEqual) {
		step += stepmod
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyMinus) {
		step -= stepmod
	}
	// keep step within bounds 1-50)
	if step > 50 {
		step = 50
	} else if step <= 0 {
		step = 1
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
		dragging = true
		primary = px
		brightness = py
	} else if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
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

	// if an error occurred or we don't need to draw, there's nothing left to do
	if (e != nil) || ebiten.IsDrawingSkipped() {
		return
	}

	// app background color
	screen.Fill(color.RGBA{0x33, 0x33, 0x33, 0xff})

	// draw raw picker colors
	b := picker.Bounds()
	op := &ebiten.DrawImageOptions{}
	op.SourceRect = &b
	screen.DrawImage(picker, op)

	// calculate golden ratio offsets
	var d0, d1, d2, sW float64
	sW = float64(screenW)
	d0 = float64(primary)
	d1 = float64(primary + distance)
	d2 = golden.Ratio(d0, d1)
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

	// draw color boxes
	colorPrimary = screen.At(int(math.Round(d0)), brightness)
	colorSecondary = screen.At(int(math.Round(d2)), brightness)
	colorTertiary = screen.At(int(math.Round(d1)), brightness)

	cbounds := image.Rect(0, 0, int((screenW-padding*4)/3), outputH)
	cpop := &ebiten.DrawImageOptions{}
	csop := &ebiten.DrawImageOptions{}
	ctop := &ebiten.DrawImageOptions{}
	cpop.SourceRect = &cbounds
	csop.SourceRect = &cbounds
	ctop.SourceRect = &cbounds
	cpimg, _ := ebiten.NewImage(cbounds.Max.X, cbounds.Max.Y, ebiten.FilterDefault)
	csimg, _ := ebiten.NewImage(cbounds.Max.X, cbounds.Max.Y, ebiten.FilterDefault)
	ctimg, _ := ebiten.NewImage(cbounds.Max.X, cbounds.Max.Y, ebiten.FilterDefault)
	cpimg.Fill(colorPrimary)
	csimg.Fill(colorSecondary)
	ctimg.Fill(colorTertiary)

	cpR32, cpG32, cpB32, _ := colorPrimary.RGBA()
	cpR, cpG, cpB := uint8(cpR32), uint8(cpG32), uint8(cpB32)
	cpC, cpM, cpY, cpK := color.RGBToCMYK(cpR, cpB, cpG)

	csR32, csG32, csB32, _ := colorSecondary.RGBA()
	csR, csG, csB := uint8(csR32), uint8(csG32), uint8(csB32)
	csC, csM, csY, csK := color.RGBToCMYK(csR, csB, csG)

	ctR32, ctG32, ctB32, _ := colorTertiary.RGBA()
	ctR, ctG, ctB := uint8(ctR32), uint8(ctG32), uint8(ctB32)
	ctC, ctM, ctY, ctK := color.RGBToCMYK(ctR, ctB, ctG)

	cpop.GeoM.Translate(float64(padding), float64(pickerH+padding))
	screen.DrawImage(cpimg, cpop)
	csop.GeoM.Translate(float64(padding*2+cbounds.Max.X), float64(pickerH+padding))
	screen.DrawImage(csimg, csop)
	ctop.GeoM.Translate(float64(padding*3+cbounds.Max.X*2), float64(pickerH+padding))
	screen.DrawImage(ctimg, ctop)

	bright := uint8(brightness / 2)
	shadow := color.RGBA{255 - bright, 255 - bright, 255 - bright, 255}
	cpText := color.RGBA{255 - cpR, 255 - cpG, 255 - cpB, 255}
	csText := color.RGBA{255 - csR, 255 - csG, 255 - csB, 255}
	ctText := color.RGBA{255 - ctR, 255 - ctG, 255 - ctB, 255}
	text.Draw(screen, fmt.Sprintf("rgb(%d, %d, %d)", cpR, cpG, cpB), arcadeFont, padding*2+2, pickerH+padding*4+2, shadow)
	text.Draw(screen, fmt.Sprintf("rgb(%d, %d, %d)", cpR, cpG, cpB), arcadeFont, padding*2, pickerH+padding*4, cpText)
	text.Draw(screen, fmt.Sprintf("#%02X%02X%02X", cpR, cpG, cpB), arcadeFont, padding*2+2, pickerH+padding*6+2, shadow)
	text.Draw(screen, fmt.Sprintf("#%02X%02X%02X", cpR, cpG, cpB), arcadeFont, padding*2, pickerH+padding*6, cpText)
	text.Draw(screen, fmt.Sprintf("cmyk(%d, %d, %d, %d)", cpC, cpM, cpY, cpK), arcadeFont, padding*2+2, pickerH+padding*8+2, shadow)
	text.Draw(screen, fmt.Sprintf("cmyk(%d, %d, %d, %d)", cpC, cpM, cpY, cpK), arcadeFont, padding*2, pickerH+padding*8, cpText)

	text.Draw(screen, fmt.Sprintf("rgb(%d, %d, %d)", csR, csG, csB), arcadeFont, padding*3+cbounds.Max.X+2, pickerH+padding*4+2, shadow)
	text.Draw(screen, fmt.Sprintf("rgb(%d, %d, %d)", csR, csG, csB), arcadeFont, padding*3+cbounds.Max.X, pickerH+padding*4, csText)
	text.Draw(screen, fmt.Sprintf("#%02X%02X%02X", csR, csG, csB), arcadeFont, padding*3+cbounds.Max.X+2, pickerH+padding*6+2, shadow)
	text.Draw(screen, fmt.Sprintf("#%02X%02X%02X", csR, csG, csB), arcadeFont, padding*3+cbounds.Max.X, pickerH+padding*6, csText)
	text.Draw(screen, fmt.Sprintf("cmyk(%d, %d, %d, %d)", csC, csM, csY, csK), arcadeFont, padding*3+cbounds.Max.X+2, pickerH+padding*8+2, shadow)
	text.Draw(screen, fmt.Sprintf("cmyk(%d, %d, %d, %d)", csC, csM, csY, csK), arcadeFont, padding*3+cbounds.Max.X, pickerH+padding*8, csText)

	text.Draw(screen, fmt.Sprintf("rgb(%d, %d, %d)", ctR, ctG, ctB), arcadeFont, padding*4+cbounds.Max.X*2+2, pickerH+padding*4+2, shadow)
	text.Draw(screen, fmt.Sprintf("rgb(%d, %d, %d)", ctR, ctG, ctB), arcadeFont, padding*4+cbounds.Max.X*2, pickerH+padding*4, ctText)
	text.Draw(screen, fmt.Sprintf("#%02X%02X%02X", ctR, ctG, ctB), arcadeFont, padding*4+cbounds.Max.X*2+2, pickerH+padding*6+2, shadow)
	text.Draw(screen, fmt.Sprintf("#%02X%02X%02X", ctR, ctG, ctB), arcadeFont, padding*4+cbounds.Max.X*2, pickerH+padding*6, ctText)
	text.Draw(screen, fmt.Sprintf("cmyk(%d, %d, %d, %d)", ctC, ctM, ctY, ctK), arcadeFont, padding*4+cbounds.Max.X*2+2, pickerH+padding*8+2, shadow)
	text.Draw(screen, fmt.Sprintf("cmyk(%d, %d, %d, %d)", ctC, ctM, ctY, ctK), arcadeFont, padding*4+cbounds.Max.X*2, pickerH+padding*8, ctText)

	// draw guides
	// TODO make these fancier
	ebitenutil.DrawLine(screen, d0, 0, d0, float64(pickerH), color.RGBA{255, 0, 0, 255})
	ebitenutil.DrawLine(screen, d1, 0, d1, float64(pickerH), color.RGBA{0, 255, 0, 255})
	ebitenutil.DrawLine(screen, d2, 0, d2, float64(pickerH), color.RGBA{0, 0, 255, 255})
	ebitenutil.DrawLine(screen, 0, float64(brightness), float64(screenW), float64(brightness), color.RGBA{bright, bright, bright, 255})

	// debug info
	// ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %v\n%d, %d\n%v %v %v %v\n%v %v %v", ebiten.CurrentFPS(), px, py, d0, d1, d2, bright, colorPrimary, colorSecondary, colorTertiary))

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
