// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"context"
	"io"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/sirupsen/logrus"
)

// volume is the interface to a back-end storage device.
type volume interface {
	// Return a unique identifier for the backend device. If
	// possible, this should be chosen such that keepstore
	// processes running on different hosts, and accessing the
	// same backend device, will return the same string.
	//
	// This helps keep-balance avoid redundantly downloading
	// multiple index listings for the same backend device.
	DeviceID() string

	// Copy a block from the backend device to writeTo.
	//
	// As with all volume methods, the hash argument is a
	// 32-character hexadecimal string.
	//
	// Data can be written to writeTo in any order, and concurrent
	// calls to writeTo.WriteAt() are allowed.  However, BlockRead
	// must not do multiple writes that intersect with any given
	// byte offset.
	//
	// BlockRead is not expected to verify data integrity.
	//
	// If the indicated block does not exist, or has been trashed,
	// BlockRead must return os.ErrNotExist.
	BlockRead(ctx context.Context, hash string, writeTo io.WriterAt) error

	// Store a block on the backend device, and set its timestamp
	// to the current time.
	//
	// The implementation must ensure that regardless of any
	// errors encountered while writing, a partially written block
	// is not left behind: a subsequent BlockRead call must return
	// either a) the data previously stored under the given hash,
	// if any, or b) os.ErrNotExist.
	BlockWrite(ctx context.Context, hash string, data []byte) error

	// Update the indicated block's stored timestamp to the
	// current time.
	BlockTouch(hash string) error

	// Return the indicated block's stored timestamp.
	Mtime(hash string) (time.Time, error)

	// Mark the indicated block as trash, such that -- unless it
	// is untrashed before time.Now() + BlobTrashLifetime --
	// BlockRead returns os.ErrNotExist and the block is not
	// listed by Index.
	BlockTrash(hash string) error

	// Un-mark the indicated block as trash. If the block has not
	// been trashed, return os.ErrNotExist.
	BlockUntrash(hash string) error

	// Permanently delete all blocks that have been marked as
	// trash for BlobTrashLifetime or longer.
	EmptyTrash()

	// Write an index of all non-trashed blocks available on the
	// backend device whose hash begins with the given prefix
	// (prefix is a string of zero or more hexadecimal digits).
	//
	// Each block is written as "{hash}+{size} {timestamp}\n"
	// where timestamp is a decimal-formatted number of
	// nanoseconds since the UTC Unix epoch.
	//
	// Index should abort and return ctx.Err() if ctx is cancelled
	// before indexing is complete.
	Index(ctx context.Context, prefix string, writeTo io.Writer) error
}

type volumeDriver func(newVolumeParams) (volume, error)

type newVolumeParams struct {
	UUID         string
	Cluster      *arvados.Cluster
	ConfigVolume arvados.Volume
	Logger       logrus.FieldLogger
	MetricsVecs  *volumeMetricsVecs
	BufferPool   *bufferPool
}

// ioStats tracks I/O statistics for a volume or server
type ioStats struct {
	Errors     uint64
	Ops        uint64
	CompareOps uint64
	GetOps     uint64
	PutOps     uint64
	TouchOps   uint64
	InBytes    uint64
	OutBytes   uint64
}
