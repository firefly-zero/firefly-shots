package main

import (
	"github.com/firefly-zero/firefly-go/firefly"
	"github.com/firefly-zero/firefly-go/firefly/sudo"
)

type App struct {
	author string
	app    string
	shots  []string
}

var (
	// Loaded on startup
	apps []App
	font firefly.Font

	// Updated on input
	appIdx     int
	shotIdx    int
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
		rawShot := loadRawShot(apps[appIdx])
		if rawShot != nil {
			loadShot(rawShot)
		}
	}
}

func loadRawShot(app App) []uint8 {
	path := "data/" + app.author + "/" + app.app + "/shots/" + app.shots[shotIdx]
	rawShot := sudo.LoadFile(path)
	if len(rawShot) == 0 {
		return nil
	}
	return rawShot
}

func loadShot(rawShot []uint8) {
	if len(rawShot) != 0x4b31 {
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
	img := firefly.File(image).Image()
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

func handlePad(newPad firefly.Pad) {
	newDPad := newPad.DPad4()
	if newDPad != firefly.DPad4None {
		shot = nil
		dirty = true
	}
	switch newDPad {
	case firefly.DPad4Left:
		if shotIdx > 0 {
			shotIdx -= 1
		}
	case firefly.DPad4Right:
		if shotIdx < len(apps[appIdx].shots)-1 {
			shotIdx += 1
		}
	case firefly.DPad4Up:
		if appIdx > 0 {
			appIdx -= 1
			shotIdx = 0
		}
	case firefly.DPad4Down:
		if appIdx < len(apps)-1 {
			appIdx += 1
			shotIdx = 0
		}
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
		renderShot(apps[appIdx])
	}
}

func renderNoShots() {
	firefly.ClearScreen(firefly.ColorWhite)
	firefly.DrawText("no screenshots", font, firefly.Point{X: 40, Y: 40}, firefly.ColorBlack)
}

func renderShot(app App) {
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
		path := app.author + "." + app.app + " " + app.shots[shotIdx][:3]
		firefly.DrawText(path, font, firefly.Point{X: 4, Y: 10}, firefly.ColorBlack)
	}
}

func listApps() []App {
	result := make([]App, 0)
	for _, author := range sudo.ListDirs("data") {
		for _, app := range sudo.ListDirs("data/" + author) {
			dir := "data/" + author + "/" + app + "/shots"
			shots := sudo.ListFiles(dir)
			sort(shots)
			if len(shots) != 0 {
				result = append(result, App{
					author: author,
					app:    app,
					shots:  shots,
				})
			}
		}
	}
	return result
}

// Bubble sort! Because stdlib sorting is too fat.
func sort(items []string) {
	for i := range len(items) - 1 {
		for j := range len(items) - i - 1 {
			if items[j] > items[j+1] {
				items[j], items[j+1] = items[j+1], items[j]
			}
		}
	}
}
