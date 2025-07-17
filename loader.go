package main

import (
	"bytes"
	"compress/zlib"
	"errors"
	"io"
	"slices"

	"github.com/firefly-zero/firefly-go/firefly"
)

type Loader struct {
	raw    []byte
	read   int
	reader io.ReadCloser
}

func NewLoader(png firefly.File) (*Loader, error) {
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
	img := make([]byte, headerSize+bodySize)
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

	return &Loader{
		raw:    img,
		read:   headerSize,
		reader: r,
	}, nil
}

func (l *Loader) Close() {
	_ = l.reader.Close()
}

func (l *Loader) Image() *firefly.Image {
	result := firefly.File{Raw: l.raw}.Image()
	return &result
}

func (l *Loader) Next() (bool, error) {
	_, _ = l.reader.Read([]byte{0})
	_, err := io.ReadFull(l.reader, l.raw[l.read:l.read+120])
	if err != nil {
		return false, err
	}
	l.read += 120
	if l.read >= len(l.raw) {
		return true, nil
	}
	return false, nil
}
