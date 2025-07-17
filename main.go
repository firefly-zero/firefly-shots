package main

import (
	"strconv"

	"github.com/firefly-zero/firefly-go/firefly"
	"github.com/firefly-zero/firefly-go/firefly/sudo"
)

var (
	apps    []string
	appIdx  int
	shotIdx int
	font    firefly.Font
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
	// ...
}

func render() {
	renderShot(apps[appIdx], shotIdx)
}

func renderShot(app string, idx int) {
	path := app + "/" + strconv.FormatInt(int64(idx), 10) + ".png"
	firefly.DrawText(path, font, firefly.Point{X: 40, Y: 40}, firefly.ColorBlack)
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
