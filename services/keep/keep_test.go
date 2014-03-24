package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

var TEST_BLOCK = []byte("The quick brown fox jumps over the lazy dog.")
var TEST_HASH = "e4d909c290d0fb1ca068ffaddf22cbd0"

func TestGetBlockOK(t *testing.T) {
	var err error

	// Manually populate keep1 and keep2 with a block.
	KeepVolumes = make([]string, 2)
	for i := range KeepVolumes {
		if dir, err := ioutil.TempDir(os.TempDir(), "keeptest"); err == nil {
			KeepVolumes[i] = dir + "/keep"
		} else {
			t.Fatal(err)
		}

		blockdir := fmt.Sprintf("%s/%s", KeepVolumes[i], TEST_HASH[:3])
		if err := os.MkdirAll(blockdir, 0755); err != nil {
			t.Fatal(err)
		}

		blockpath := fmt.Sprintf("%s/%s", blockdir, TEST_HASH)
		if f, err := os.Create(blockpath); err == nil {
			f.Write(TEST_BLOCK)
			f.Close()
		} else {
			t.Fatal(err)
		}

	}

	// Check that GetBlock returns success.
	result, err := GetBlock(TEST_HASH)
	if err != nil {
		t.Errorf("GetBlock error: %s", err)
	}
	if fmt.Sprint(result) != fmt.Sprint(TEST_BLOCK) {
		t.Errorf("expected %s, got %s", TEST_BLOCK, result)
	}

	for _, vol := range KeepVolumes {
		os.RemoveAll(path.Dir(vol))
	}
}
