package main

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/sdk/go/manifest"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Block struct {
	data   []byte
	offset int64
}

type ManifestStreamWriter struct {
	*ManifestWriter
	*manifest.ManifestStream
	offset int64
	*Block
	uploader chan *Block
	finish   chan []error
}

type IKeepClient interface {
	PutHB(hash string, buf []byte) (string, int, error)
}

func (m *ManifestStreamWriter) Write(p []byte) (int, error) {
	n, err := m.ReadFrom(bytes.NewReader(p))
	return int(n), err
}

func (m *ManifestStreamWriter) ReadFrom(r io.Reader) (n int64, err error) {
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

	if err == io.EOF {
		return total, nil
	} else {
		return total, err
	}

}

func (m *ManifestStreamWriter) goUpload() {
	var errors []error
	uploader := m.uploader
	finish := m.finish
	for block := range uploader {
		hash := fmt.Sprintf("%x", md5.Sum(block.data[0:block.offset]))
		signedHash, _, err := m.ManifestWriter.IKeepClient.PutHB(hash, block.data[0:block.offset])
		if err != nil {
			errors = append(errors, err)
		} else {
			m.ManifestStream.Blocks = append(m.ManifestStream.Blocks, signedHash)
		}
	}
	finish <- errors
}

type ManifestWriter struct {
	IKeepClient
	stripPrefix string
	Streams     map[string]*ManifestStreamWriter
}

func (m *ManifestWriter) WalkFunc(path string, info os.FileInfo, err error) error {
	if info.IsDir() {
		return nil
	}

	var dir string
	if len(path) > (len(m.stripPrefix) + len(info.Name()) + 1) {
		dir = path[len(m.stripPrefix)+1 : (len(path) - len(info.Name()) - 1)]
	}
	if dir == "" {
		dir = "."
	}

	fn := path[(len(path) - len(info.Name())):]

	if m.Streams[dir] == nil {
		m.Streams[dir] = &ManifestStreamWriter{
			m,
			&manifest.ManifestStream{StreamName: dir},
			0,
			nil,
			make(chan *Block),
			make(chan []error)}
		go m.Streams[dir].goUpload()
	}

	stream := m.Streams[dir]

	fileStart := stream.offset

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	log.Printf("Uploading %v/%v (%v bytes)", dir, fn, info.Size())

	var count int64
	count, err = io.Copy(stream, file)
	if err != nil {
		return err
	}

	stream.offset += count

	stream.ManifestStream.FileStreamSegments = append(stream.ManifestStream.FileStreamSegments,
		manifest.FileStreamSegment{uint64(fileStart), uint64(count), fn})

	return nil
}

func (m *ManifestWriter) Finish() error {
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
	} else {
		return nil
	}
}

func (m *ManifestWriter) ManifestText() string {
	m.Finish()
	var buf bytes.Buffer

	dirs := make([]string, len(m.Streams))
	i := 0
	for k := range m.Streams {
		dirs[i] = k
		i++
	}
	sort.Strings(dirs)

	for _, k := range dirs {
		v := m.Streams[k]

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
			buf.WriteString(fmt.Sprintf("%d:%d:%s", f.SegPos, f.SegLen, name))
		}
		buf.WriteString("\n")
	}
	return buf.String()
}

func WriteTree(kc IKeepClient, root string) (manifest string, err error) {
	mw := ManifestWriter{kc, root, map[string]*ManifestStreamWriter{}}
	err = filepath.Walk(root, mw.WalkFunc)

	if err != nil {
		return "", err
	}

	err = mw.Finish()
	if err != nil {
		return "", err
	}

	return mw.ManifestText(), nil
}
