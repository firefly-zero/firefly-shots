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
	wasTouched = isTouched
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
	path := app + "/" + strconv.FormatInt(int64(idx), 10) + ".png"
	firefly.DrawText(path, font, firefly.Point{X: 4, Y: 10}, firefly.ColorBlack)
	// firefly.DrawImage()
}

func listApps() []string {
	result := make([]string, 0)
	for _, author := range sudo.ListDirs("data") {
		for _, app := range sudo.ListDirs("data/" + author) {
			dir := "data/" + author + "/" + app + "/shots"
			hasShots := sudo.LoadFile(dir+"/1.png").Raw != nil
			if hasShots {
				result = append(result, dir)
			}
		}
	}
	return result
}
