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
	var errors []error
	uploader := m.uploader
	finish := m.finish
	for true {
		select {
		case block, valid := <-uploader:
			if !valid {
				finish <- errors
				return
			}
			hash := fmt.Sprintf("%x", md5.Sum(block.data[0:block.offset]))
			signedHash, _, err := m.ManifestWriter.IKeepClient.PutHB(hash, block.data[0:block.offset])
			if err != nil {
				errors = append(errors, err)
			} else {
				m.ManifestStream.Blocks = append(m.ManifestStream.Blocks, signedHash)
			}
		}
	}
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
	if err != nil && err != io.EOF {
		return err
	}

	stream.offset += count

	stream.ManifestStream.Files = append(stream.ManifestStream.Files,
		fmt.Sprintf("%v:%v:%v", fileStart, count, fn))

	return nil
}

func (m *ManifestWriter) Finish() error {
	var errstring string
	for _, v := range m.Streams {
		if v.uploader != nil {
			if v.Block != nil {
				v.uploader <- v.Block
			}
			close(v.uploader)
			v.uploader = nil

			errors := <-v.finish
			close(v.finish)
			v.finish = nil

			if errors != nil {
				for _, r := range errors {
					errstring = fmt.Sprintf("%v%v\n", errstring, r.Error())
				}
			}
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
		for _, f := range v.Files {
			buf.WriteString(" ")
			f = strings.Replace(f, " ", "\\040", -1)
			f = strings.Replace(f, "\n", "", -1)
			buf.WriteString(f)
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
