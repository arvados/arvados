package main

// Originally based on sdk/go/crunchrunner/upload.go
//
// Unlike the original, which iterates over a directory tree and uploads each
// file sequentially, this version supports opening and writing multiple files
// in a collection simultaneously.
//
// Eventually this should move into the Arvados Go SDK for a more comprehensive
// implementation of Collections.

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/sdk/go/manifest"
	"io"
	"strings"
)

// Block is a data block in a manifest stream
type Block struct {
	data   []byte
	offset int64
}

// CollectionFileWriter is a Writer that permits writing to a file in a Keep Collection.
type CollectionFileWriter struct {
	IKeepClient
	*manifest.ManifestStream
	offset uint64
	length uint64
	*Block
	uploader chan *Block
	finish   chan []error
	fn       string
}

// Write to a file in a keep collection
func (m *CollectionFileWriter) Write(p []byte) (int, error) {
	n, err := m.ReadFrom(bytes.NewReader(p))
	return int(n), err
}

// ReadFrom a Reader and write to the Keep collection file.
func (m *CollectionFileWriter) ReadFrom(r io.Reader) (n int64, err error) {
	var total int64
	var count int

	for err == nil {
		if m.Block == nil {
			m.Block = &Block{make([]byte, keepclient.BLOCKSIZE), 0}
		}
		count, err = r.Read(m.Block.data[m.Block.offset:])
		total += int64(count)
		m.Block.offset += int64(count)
		if m.Block.offset == keepclient.BLOCKSIZE {
			m.uploader <- m.Block
			m.Block = nil
		}
	}

	m.length += uint64(total)

	if err == io.EOF {
		return total, nil
	}
	return total, err
}

// Close stops writing a file and adds it to the parent manifest.
func (m *CollectionFileWriter) Close() error {
	m.ManifestStream.FileStreamSegments = append(m.ManifestStream.FileStreamSegments,
		manifest.FileStreamSegment{m.offset, m.length, m.fn})
	return nil
}

func (m *CollectionFileWriter) goUpload() {
	var errors []error
	uploader := m.uploader
	finish := m.finish
	for block := range uploader {
		hash := fmt.Sprintf("%x", md5.Sum(block.data[0:block.offset]))
		signedHash, _, err := m.IKeepClient.PutHB(hash, block.data[0:block.offset])
		if err != nil {
			errors = append(errors, err)
		} else {
			m.ManifestStream.Blocks = append(m.ManifestStream.Blocks, signedHash)
		}
	}
	finish <- errors
}

// CollectionWriter makes implements creating new Keep collections by opening files
// and writing to them.
type CollectionWriter struct {
	IKeepClient
	Streams []*CollectionFileWriter
}

// Open a new file for writing in the Keep collection.
func (m *CollectionWriter) Open(path string) io.WriteCloser {
	var dir string
	var fn string

	i := strings.Index(path, "/")
	if i > -1 {
		dir = "./" + path[0:i]
		fn = path[i+1:]
	} else {
		dir = "."
		fn = path
	}

	fw := &CollectionFileWriter{
		m.IKeepClient,
		&manifest.ManifestStream{StreamName: dir},
		0,
		0,
		nil,
		make(chan *Block),
		make(chan []error),
		fn}
	go fw.goUpload()

	m.Streams = append(m.Streams, fw)

	return fw
}

// Finish writing the collection, wait for all blocks to complete uploading.
func (m *CollectionWriter) Finish() error {
	var errstring string
	for _, stream := range m.Streams {
		if stream.uploader == nil {
			continue
		}
		if stream.Block != nil {
			stream.uploader <- stream.Block
		}
		close(stream.uploader)
		stream.uploader = nil

		errors := <-stream.finish
		close(stream.finish)
		stream.finish = nil

		for _, r := range errors {
			errstring = fmt.Sprintf("%v%v\n", errstring, r.Error())
		}
	}
	if errstring != "" {
		return errors.New(errstring)
	}
	return nil
}

// ManifestText returns the manifest text of the collection.  Calls Finish()
// first to ensure that all blocks are written and that signed locators and
// available.
func (m *CollectionWriter) ManifestText() (mt string, err error) {
	err = m.Finish()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer

	for _, v := range m.Streams {
		k := v.StreamName
		if k == "." {
			buf.WriteString(".")
		} else {
			k = strings.Replace(k, " ", "\\040", -1)
			k = strings.Replace(k, "\n", "", -1)
			buf.WriteString("./" + k)
		}
		for _, b := range v.Blocks {
			buf.WriteString(" ")
			buf.WriteString(b)
		}
		for _, f := range v.FileStreamSegments {
			buf.WriteString(" ")
			name := strings.Replace(f.Name, " ", "\\040", -1)
			name = strings.Replace(name, "\n", "", -1)
			buf.WriteString(fmt.Sprintf("%v:%v:%v", f.SegPos, f.SegLen, name))
		}
		buf.WriteString("\n")
	}
	return buf.String(), nil
}
