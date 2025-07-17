package main

import (
	"strconv"

	"github.com/firefly-zero/firefly-go/firefly"
	"github.com/firefly-zero/firefly-go/firefly/sudo"
)

var (
	apps       []string
	appIdx     int
	shotIdx    int = 1
	font       firefly.Font
	shot       *firefly.Image
	dirty      bool = true
	wasTouched bool
)

func init() {
	firefly.Boot = boot
	firefly.Update = update
	firefly.Render = render
}

func boot() {
	font = firefly.LoadFile("font", nil).Font()
	apps = listApps()
}

func update() {
	newPad, isTouched := firefly.ReadPad(firefly.Combined)
	if !wasTouched && isTouched {
		handlePad(newPad)
	}
	wasTouched = isTouched
	if dirty {
		loadShot(apps[appIdx], shotIdx)
	}
}

func handlePad(newPad firefly.Pad) {
	newDPad := newPad.DPad()
	if newDPad.Left && shotIdx > 1 {
		dirty = true
		shotIdx -= 1
	}
	if newDPad.Right {
		dirty = true
		shotIdx += 1
	}
	if newDPad.Up && appIdx > 0 {
		dirty = true
		appIdx -= 1
	}
	if newDPad.Down && appIdx < len(apps)-1 {
		dirty = true
		appIdx += 1
	}
}

func render() {
	if !dirty {
		return
	}
	dirty = false
	firefly.ClearScreen(firefly.ColorWhite)
	if len(apps) == 0 {
		renderNoShots()
	} else {
		renderShot(apps[appIdx], shotIdx)
	}
}

func renderNoShots() {
	firefly.DrawText("no screenshots", font, firefly.Point{X: 40, Y: 40}, firefly.ColorBlack)
}

func renderShot(app string, idx int) {
	if shot != nil {
		firefly.DrawImage(*shot, firefly.Point{})
	}

	path := app + "/" + strconv.FormatInt(int64(idx), 10) + ".png"
	firefly.DrawText(path, font, firefly.Point{X: 4, Y: 10}, firefly.ColorBlack)
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

func loadShot(app string, idx int) {
	path := app + "/" + strconv.FormatInt(int64(idx), 10) + ".png"
	png := sudo.LoadFile(path)
	shot = parseShot(png)
}

func parseShot(png firefly.File) *firefly.Image {
	if png.Raw == nil {
		return nil
	}
	img := make([]byte, 0)

	raw := png.Raw
	if len(raw) < 100 {
		return nil
	}
	raw = raw[8:]                // skip magic number
	raw = raw[4+4+8+13:]         // skip IHDR
	raw = raw[4+4+8+16*3:]       // skip PLTE
	raw = raw[:len(raw)-(4+4+8)] // skip IEND

	firefly.LogDebug(strconv.FormatInt(int64(len(raw)), 10))
	img = append(img,
		4,      // BPP
		0, 240, // width
		17, // transparent color (no transparency)
	)
	img = append(img, raw...)

	result := firefly.File{Raw: img}.Image()
	return &result
}
