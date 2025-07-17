# firefly-shots

Screenshot viewer for [Firefly Zero](https://fireflyzero.com/).

Currently, doesn't work for one simple reason: screenshots are stored as PNG which uses zlib compression. And initializing Go zlib decompressor consumes an insane amount of fuel which Firefly runtime terminates.
