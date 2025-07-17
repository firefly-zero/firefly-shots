package main

import (
	"strconv"

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
	loader     *Loader
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
	if dirty && shot == nil && loader == nil {
		makeLoader(apps[appIdx], shotIdx)
	} else {
		advanceLoader()
	}
}

func handlePad(newPad firefly.Pad) {
	newDPad := newPad.DPad()
	if newDPad.Left && shotIdx > 1 {
		shot = nil
		loader = nil
		dirty = true
		shotIdx -= 1
	}
	if newDPad.Right {
		shot = nil
		loader = nil
		dirty = true
		shotIdx += 1
	}
	if newDPad.Up && appIdx > 0 {
		shot = nil
		loader = nil
		dirty = true
		appIdx -= 1
	}
	if newDPad.Down && appIdx < len(apps)-1 {
		shot = nil
		loader = nil
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
	firefly.DrawText("Loading...", font, firefly.Point{X: 86, Y: 80}, firefly.ColorWhite)

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
		path := app + "/" + strconv.FormatInt(int64(idx), 10) + ".png"
		firefly.DrawText(path, font, firefly.Point{X: 4, Y: 10}, firefly.ColorBlack)
	}
}

func listApps() []string {
	result := make([]string, 0)
	for _, author := range sudo.ListDirs("data") {
		for _, app := range sudo.ListDirs("data/" + author) {
			dir := "data/" + author + "/" + app + "/shots"
			hasShots := len(sudo.LoadFile(dir+"/1.png").Raw) != 0
			if hasShots {
				result = append(result, dir)
			}
		}
	}
	return result
}

func makeLoader(app string, idx int) {
	path := app + "/" + strconv.FormatInt(int64(idx), 10) + ".png"
	png := sudo.LoadFile(path)
	l, err := NewLoader(png)
	if err != nil {
		firefly.LogError(err.Error())
		return
	}
	loader = l
}

func advanceLoader() {
	if loader == nil {
		return
	}
	dirty = true
	done, err := loader.Next()
	if err != nil {
		firefly.LogError(err.Error())
		loader.Close()
		return
	}
	shot = loader.Image()
	if done {
		loader.Close()
		loader = nil
		return
	}
}
