package main

import (
	"strconv"
	"unsafe"

	"github.com/firefly-zero/firefly-go/firefly"
	"github.com/firefly-zero/firefly-go/firefly/sudo"
)

var (
	// Loaded on startup
	apps []string
	font firefly.Font

	// Updated on input
	appIdx     int
	shotIdx    int = 1
	shot       *firefly.Image
	showUI     bool
	dirty      bool = true
	wasTouched bool
	oldBtns    firefly.Buttons
)

func init() {
	firefly.Boot = boot
	firefly.Update = update
	firefly.Render = render
}

func boot() {
	font = firefly.LoadFile("font", nil).Font()
	apps = listApps()
	oldBtns = firefly.ReadButtons(firefly.Combined)
}

func update() {
	newPad, isTouched := firefly.ReadPad(firefly.Combined)
	if !wasTouched && isTouched {
		handlePad(newPad)
	}
	newBtns := firefly.ReadButtons(firefly.Combined)
	handleBtns(newBtns)
	oldBtns = newBtns
	wasTouched = isTouched
	if dirty && shot == nil && len(apps) != 0 {
		rawShot := loadRawShot(apps[appIdx], shotIdx)
		// If trying to load a shot after the last one, loop around.
		if rawShot == nil && shotIdx > 1 {
			shotIdx = 1
			rawShot = loadRawShot(apps[appIdx], shotIdx)
		}
		if rawShot != nil {
			loadShot(rawShot)
		}
	}
}

func loadRawShot(app string, idx int) []uint8 {
	path := app + "/" + formatInt(idx) + ".ffs"
	rawShot := sudo.LoadFile(path).Raw
	if len(rawShot) == 0 {
		return nil
	}
	return rawShot
}

func loadShot(rawShot []uint8) {
	if len(rawShot) != 0x4b31 {
		firefly.LogDebug(strconv.FormatInt(int64(len(rawShot)), 16))
		firefly.LogError("invalid file size")
		return
	}
	if rawShot[0x00] != 0x41 {
		firefly.LogError("invalid magic number")
		return
	}
	setPalette(rawShot[0x01:0x31])
	const headerSize = 5 + 8
	image := rawShot[0x31-headerSize:]
	image[0] = 0x21                      // magic number
	image[1] = 4                         // BPP
	image[2] = byte(firefly.Width)       // width
	image[3] = byte(firefly.Height >> 8) // with
	image[4] = 255                       // transparency

	// color swaps
	var i byte
	for i = range 8 {
		image[5+i] = ((i * 2) << 4) | (i*2 + 1)
	}
	switchIndianness(image[headerSize:])
	img := firefly.File{Raw: image}.Image()
	shot = &img
}

func switchIndianness(raw []uint8) {
	for i, b := range raw {
		raw[i] = (b << 4) | (b >> 4)
	}
}

func setPalette(raw []uint8) {
	for i := range 16 {
		rgb := firefly.RGB{
			R: raw[i*3],
			G: raw[i*3+1],
			B: raw[i*3+2],
		}
		firefly.SetColor(firefly.Color(i+1), rgb)
	}
}

func formatInt(i int) string {
	buf := []byte{
		'0' + byte(i/100),
		'0' + byte((i%100)/10),
		'0' + byte(i%10),
	}
	return unsafe.String(&buf[0], 3)
}

func handlePad(newPad firefly.Pad) {
	newDPad := newPad.DPad()
	if newDPad.Left && shotIdx > 1 {
		shot = nil
		dirty = true
		shotIdx -= 1
	}
	if newDPad.Right {
		shot = nil
		dirty = true
		shotIdx += 1
	}
	if newDPad.Up && appIdx > 0 {
		shot = nil
		dirty = true
		appIdx -= 1
	}
	if newDPad.Down && appIdx < len(apps)-1 {
		shot = nil
		dirty = true
		appIdx += 1
	}
}

func handleBtns(newBtns firefly.Buttons) {
	justPressed := newBtns.JustPressed(oldBtns)
	if justPressed.S {
		showUI = !showUI
		dirty = true
	}
}

func render() {
	// if !dirty {
	// 	return
	// }
	dirty = false
	if len(apps) == 0 {
		renderNoShots()
	} else {
		renderShot(apps[appIdx], shotIdx)
	}
}

func renderNoShots() {
	firefly.ClearScreen(firefly.ColorWhite)
	firefly.DrawText("no screenshots", font, firefly.Point{X: 40, Y: 40}, firefly.ColorBlack)
}

func renderShot(app string, idx int) {
	firefly.ClearScreen(firefly.ColorBlack)
	firefly.DrawText("cannot load image", font, firefly.Point{X: 66, Y: 80}, firefly.ColorWhite)

	if shot != nil {
		firefly.DrawImage(*shot, firefly.Point{})
	}

	if showUI {
		firefly.DrawRect(
			firefly.Point{X: -1, Y: -1},
			firefly.Size{W: firefly.Width + 2, H: 16},
			firefly.Style{
				FillColor:   firefly.ColorWhite,
				StrokeColor: firefly.ColorBlack,
				StrokeWidth: 1,
			},
		)
		path := app + "/" + formatInt(idx) + ".ffs"
		firefly.DrawText(path, font, firefly.Point{X: 4, Y: 10}, firefly.ColorBlack)
	}
}

func listApps() []string {
	result := make([]string, 0)
	for _, author := range sudo.ListDirs("data") {
		for _, app := range sudo.ListDirs("data/" + author) {
			dir := "data/" + author + "/" + app + "/shots"
			hasShots := len(sudo.LoadFile(dir+"/001.ffs").Raw) != 0
			if hasShots {
				result = append(result, dir)
			}
		}
	}
	return result
}
