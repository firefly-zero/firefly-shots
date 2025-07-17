package main

import (
	"bytes"
	"compress/zlib"
	"errors"
	"io"
	"slices"
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
	if dirty && shot == nil {
		loadShot(apps[appIdx], shotIdx)
	}
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

func loadShot(app string, idx int) {
	path := app + "/" + strconv.FormatInt(int64(idx), 10) + ".png"
	png := sudo.LoadFile(path)
	s, err := parseShot(png)
	if err != nil {
		firefly.LogError(err.Error())
		return
	}
	shot = s
}

func parseShot(png firefly.File) (*firefly.Image, error) {
	if len(png.Raw) == 0 {
		return nil, errors.New("file does not exist")
	}
	raw := png.Raw
	if len(raw) < 100 {
		return nil, errors.New("file is too short")
	}
	if !slices.Equal(raw[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		return nil, errors.New("invalid magic number")
	}

	raw = raw[8:]                // skip magic number
	raw = raw[4+4+4+13:]         // skip IHDR
	raw = raw[4+4+4+16*3:]       // skip PLTE
	raw = raw[:len(raw)-(4+4+4)] // skip IEND
	raw = raw[4+4:]              // skip IDAT header
	raw = raw[:len(raw)-4]       // skip IDAT CRC32

	r, err := zlib.NewReader(bytes.NewBuffer(raw))
	if err != nil {
		return nil, err
	}

	// raw result image header
	const headerSize = 5 + 8
	bodySize := firefly.Width * firefly.Height / 2
	img := make([]byte, headerSize, headerSize+bodySize)
	img[0] = 0x21                     // magic number
	img[1] = 4                        // BPP
	img[2] = byte(firefly.Width)      // width
	img[3] = byte(firefly.Width >> 8) // width
	img[4] = 255                      // transparency

	// color swaps
	var i byte
	for i = range 8 {
		img[5+i] = ((i * 2) << 4) | (i*2 + 1)
	}

	// pixels
	frame, err := io.ReadAll(r)
	_ = r.Close()
	if err != nil {
		return nil, err
	}
	for len(frame) != 0 {
		img = append(img, frame[1:121]...)
		frame = frame[121:]
	}

	result := firefly.File{Raw: img}.Image()
	return &result, nil
}
