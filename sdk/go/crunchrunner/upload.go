package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/sdk/go/manifest"
	"io"
	"log"
	"os"
	"path/filepath"
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
}

type IKeepClient interface {
	PutHB(hash string, buf []byte) (string, int, error)
}

func (m *ManifestStreamWriter) Write(p []byte) (n int, err error) {
	// Needed to conform to Writer interface, but not implemented
	// because io.Copy will actually use ReadFrom instead.
	return 0, nil
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
		if count > 0 {
			if m.Block.offset == keepclient.BLOCKSIZE {
				m.uploader <- m.Block
				m.Block = nil
			}
		}
	}

	return total, err
}

func (m *ManifestStreamWriter) goUpload() {
	select {
	case block, valid := <-m.uploader:
		if !valid {
			return
		}
		hash := fmt.Sprintf("%x", md5.Sum(block.data[0:block.offset]))
		signedHash, _, _ := m.ManifestWriter.IKeepClient.PutHB(hash, block.data[0:block.offset])
		m.ManifestStream.Blocks = append(m.ManifestStream.Blocks, signedHash)
	}

}

type ManifestWriter struct {
	IKeepClient
	stripPrefix string
	Streams     map[string]*ManifestStreamWriter
}

type walker struct {
	currentDir string
	m          *ManifestWriter
}

func (w walker) WalkFunc(path string, info os.FileInfo, err error) error {
	log.Print("path ", path, " ", info.Name(), " ", info.IsDir())

	if info.IsDir() {
		if path == w.currentDir {
			return nil
		}
		return filepath.Walk(path, walker{path, w.m}.WalkFunc)
	}
	m := w.m

	dir := path[len(m.stripPrefix)+1 : (len(path) - len(info.Name()))]
	fn := path[(len(path) - len(info.Name())):]

	if m.Streams[dir] == nil {
		m.Streams[dir] = &ManifestStreamWriter{
			m,
			&manifest.ManifestStream{StreamName: dir},
			0,
			nil,
			make(chan *Block)}
		go m.Streams[dir].goUpload()
	}

	stream := m.Streams[dir]

	fileStart := stream.offset

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	var count int64
	count, err = io.Copy(stream, file)
	if err != nil && err != io.EOF {
		return err
	}

	stream.offset += count

	stream.ManifestStream.Files = append(stream.ManifestStream.Files,
		fmt.Sprintf("%v:%v:%v", fileStart, count, fn))

	return nil
}

func (m *ManifestWriter) Finish() {
	for _, v := range m.Streams {
		if v.uploader != nil {
			if v.Block != nil {
				v.uploader <- v.Block
			}
			close(v.uploader)
			v.uploader = nil
		}
	}
}

func (m *ManifestWriter) ManifestText() string {
	m.Finish()
	var buf bytes.Buffer
	for k, v := range m.Streams {
		if k == "" {
			buf.WriteString(".")
		} else {
			buf.WriteString("./" + k)
		}
		for _, b := range v.Blocks {
			buf.WriteString(" ")
			buf.WriteString(b)
		}
		for _, f := range v.Files {
			buf.WriteString(" ")
			buf.WriteString(f)
		}
		buf.WriteString("\n")
	}
	return buf.String()
}

func WriteTree(kc IKeepClient, root string) (manifest string, err error) {
	mw := ManifestWriter{kc, root, map[string]*ManifestStreamWriter{}}
	err = filepath.Walk(root, walker{root, &mw}.WalkFunc)
	mw.Finish()

	if err != nil {
		return "", err
	} else {
		return mw.ManifestText(), nil
	}

}
