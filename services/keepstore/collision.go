package main

import (
	"crypto/md5"
	"fmt"
	"io"
)

// Compute the MD5 digest of a data block (consisting of buf1 + buf2 +
// all bytes readable from rdr). If all data is read successfully,
// return DiskHashError or CollisionError depending on whether it
// matches expectMD5. If an error occurs while reading, return that
// error.
//
// "content has expected MD5" is called a collision because this
// function is used in cases where we have another block in hand with
// the given MD5 but different content.
func collisionOrCorrupt(expectMD5 string, buf1, buf2 []byte, rdr io.Reader) error {
	outcome := make(chan error)
	data := make(chan []byte, 1)
	go func() {
		h := md5.New()
		for b := range data {
			h.Write(b)
		}
		if fmt.Sprintf("%x", h.Sum(nil)) == expectMD5 {
			outcome <- CollisionError
		} else {
			outcome <- DiskHashError
		}
	}()
	data <- buf1
	if buf2 != nil {
		data <- buf2
	}
	var err error
	for rdr != nil && err == nil {
		buf := make([]byte, 1<<18)
		var n int
		n, err = rdr.Read(buf)
		data <- buf[:n]
	}
	close(data)
	if rdr != nil && err != io.EOF {
		<-outcome
		return err
	}
	return <-outcome
}
