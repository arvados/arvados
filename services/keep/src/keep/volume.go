package main

import (
	"crypto/md5"
	"fmt"
	"log"
	"os"
)

// A Volume is an interface that represents a Keep back-end volume.

type Volume interface {
	Read(locator string) ([]byte, error)
	Write(locator string, block []byte) error
}

// A UnixVolume is a Volume that writes to a locally mounted disk.
type UnixVolume struct {
	root string // path to this volume
}

func (v *UnixVolume) Read(locator string) ([]byte, error) {
	var f *os.File
	var err error
	var nread int

	blockFilename := fmt.Sprintf("%s/%s/%s", v.root, locator[0:3], locator)

	f, err = os.Open(blockFilename)
	if err != nil {
		return nil, err
	}

	var buf = make([]byte, BLOCKSIZE)
	nread, err = f.Read(buf)
	if err != nil {
		log.Printf("%s: reading %s: %s\n", v.root, blockFilename, err)
		return buf, err
	}

	// Double check the file checksum.
	//
	filehash := fmt.Sprintf("%x", md5.Sum(buf[:nread]))
	if filehash != locator {
		// TODO(twp): this condition probably represents a bad disk and
		// should raise major alarm bells for an administrator: e.g.
		// they should be sent directly to an event manager at high
		// priority or logged as urgent problems.
		//
		log.Printf("%s: checksum mismatch: %s (actual locator %s)\n",
			v.root, blockFilename, filehash)
		return buf, CorruptError
	}

	// Success!
	return buf[:nread], nil
}
