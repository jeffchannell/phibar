package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
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
	val        float64     // stop value, different from offset (for calculating the others)
	off        float64     // x or y offset
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
	s.off = val
	sW := float64(screenW)
	if s.off < 0 {
		for s.off < 0 {
			s.off = sW + s.off
		}
	} else if s.off > sW {
		for s.off > sW {
			s.off -= sW
		}
	}
}

var (
	windowTitle = "PhiBar"

	picker   *ebiten.Image // color picker image
	copy     bool
	dragging bool
	ctrlDown bool // ctrl button is down

	outputH = 300
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
	stopmax    = 16
	stopmin    = 3
	stops      = 3
	stoplist   []colorStop

	colorMatrix [16][8]colorPoint // color selection matrix

	// arcadeFont font face
	arcadeFont font.Face
	// fontSize sets the base font size
	fontSize float64 = 16
)

func exportGPL() []byte {
	output := "GIMP Palette\nName: PhiBar\nColumns: 4\n#"
	for i := range stoplist {
		if i == stops {
			break
		}
		var r, g, b uint8
		if (i < len(stoplist)) && i < stops {
			r = stoplist[i].r
			g = stoplist[i].g
			b = stoplist[i].b
		}
		output = fmt.Sprintf("%s\n%d %d %d Index%d", output, r, g, b, i)
	}
	return []byte(output)
}

func exportPAL() []byte {
	num := 16
	// TODO support 256 colors as well as 16 (based on vertical stops)
	output := fmt.Sprintf("JASC-PAL\n0100\n%d", num)
	for i := 0; i < num; i++ {
		var r, g, b uint8
		if (i < len(stoplist)) && i < stops {
			r = stoplist[i].r
			g = stoplist[i].g
			b = stoplist[i].b
		}
		output = fmt.Sprintf("%s\n%d %d %d", output, r, g, b)
	}
	return []byte(output)
}

func init() {
	stoplist = make([]colorStop, stopmax)
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
	// export
	if inpututil.IsKeyJustReleased(ebiten.KeyE) {
		// TODO figure out which type of file to export
		filename, success, err := File("Select file", "*.gpl", false)
		if success && (err == nil) {
			filedata := exportGPL()
			filebytes := []byte(filedata)
			err := ioutil.WriteFile(filename, filebytes, 0644)
			if err != nil {
				panic(err)
			}
		}
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
	// keep stops within bounds
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
	// keep step within bounds
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
		stoplist[i].setColor(screen.At(int(math.Round(stoplist[i].off)), brightness))
	}

	bright := uint8(brightness / 2)

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
		ebitenutil.DrawLine(screen, stoplist[i].off, 0, stoplist[i].off, float64(pickerH), stoplist[i].negative())
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
	ebitenutil.DrawLine(screen, 0, float64(brightness), float64(screenW), float64(brightness), color.RGBA{bright, bright, bright, 255})

	// debug info
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %v TPS: %v\nx: %d, y: %d, bright: %v, steps: %d, stops: %d", ebiten.CurrentFPS(), ebiten.CurrentTPS(), px, py, bright, step, stops))

	return
}
