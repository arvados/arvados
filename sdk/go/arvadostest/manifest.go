// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvadostest

import (
	"bytes"
	"fmt"
	"math/rand"
)

func FakeManifest(dirCount, filesPerDir, blocksPerFile, interleaveChunk int) string {
	const blksize = 1 << 26
	mb := bytes.NewBuffer(make([]byte, 0, 40000000))
	blkid := 0
	for i := 0; i < dirCount; i++ {
		fmt.Fprintf(mb, "./dir%d", i)
		for j := 0; j < filesPerDir; j++ {
			for k := 0; k < blocksPerFile; k++ {
				blkid++
				fmt.Fprintf(mb, " %032x+%d+A%040x@%08x", blkid, blksize, blkid, blkid)
			}
		}
		for j := 0; j < filesPerDir; j++ {
			if interleaveChunk == 0 {
				fmt.Fprintf(mb, " %d:%d:dir%d/file%d", (filesPerDir-j-1)*blocksPerFile*blksize, blocksPerFile*blksize, j, j)
				continue
			}
			for todo := int64(blocksPerFile) * int64(blksize); todo > 0; todo -= int64(interleaveChunk) {
				size := int64(interleaveChunk)
				if size > todo {
					size = todo
				}
				offset := rand.Int63n(int64(blocksPerFile)*int64(blksize)*int64(filesPerDir) - size)
				fmt.Fprintf(mb, " %d:%d:dir%d/file%d", offset, size, j, j)
			}
		}
		mb.Write([]byte{'\n'})
	}
	return mb.String()
}
