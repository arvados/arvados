package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

type download struct {
	URL  string
	Dest string
	Size int64
	Mode os.FileMode
	Hash string
}

func (d *download) String() string {
	return fmt.Sprintf("Download %q from %q", d.Dest, d.URL)
}

func (d *download) Check() error {
	fi, err := os.Stat(d.Dest)
	if err != nil {
		return err
	}
	if d.Size > 0 && fi.Size() != d.Size {
		return fmt.Errorf("Size mismatch: %q is %d bytes, expected %d", d.Dest, fi.Size(), d.Size)
	}
	if d.Mode > 0 && fi.Mode() != d.Mode {
		return fmt.Errorf("Mode mismatch: %q is %s, expected %s", d.Dest, fi.Mode(), d.Mode)
	}
	return nil
}

func (d *download) CanFix() bool {
	return true
}

func (d *download) Fix() error {
	out, err := ioutil.TempFile(path.Dir(d.Dest), path.Base(d.Dest))
	if err != nil {
		return err
	}
	defer func() {
		if out != nil {
			os.Remove(out.Name())
			out.Close()
		}
	}()

	resp, err := http.Get(d.URL)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	if strings.HasSuffix(d.URL, ".zip") && !strings.HasSuffix(d.Dest, ".zip") {
		r, err := zip.NewReader(out, n)
		if err != nil {
			return err
		}
		defer os.Remove(out.Name())
		out = nil

		found := false
		for _, f := range r.File {
			if !strings.HasSuffix(d.Dest, "/"+f.Name) {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			out, err = ioutil.TempFile(path.Dir(d.Dest), path.Base(d.Dest))
			if err != nil {
				return err
			}

			n, err = io.Copy(out, rc)
			if err != nil {
				return err
			}
			found = true
			break
		}
		if !found {
			return fmt.Errorf("File not found in archive")
		}
	}

	if d.Size > 0 && d.Size != n {
		return fmt.Errorf("Size mismatch: got %d bytes, expected %d", n, d.Size)
	} else if d.Size == 0 {
		log.Printf("%s: size was %d", d, n)
	}
	if err = out.Close(); err != nil {
		return err
	}
	if err = os.Chmod(out.Name(), d.Mode); err != nil {
		return err
	}
	err = os.Rename(out.Name(), d.Dest)
	if err == nil {
		// skip deferred os.Remove(out.Name())
		out = nil
	}
	return err
}
