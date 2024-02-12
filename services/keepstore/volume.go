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
	BlockRead(ctx context.Context, hash string, writeTo io.Writer) (int, error)
	BlockWrite(ctx context.Context, hash string, data []byte) error
	DeviceID() string
	BlockTouch(hash string) error
	BlockTrash(hash string) error
	BlockUntrash(hash string) error
	Index(ctx context.Context, prefix string, writeTo io.Writer) error
	Mtime(hash string) (time.Time, error)
	EmptyTrash()
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

type InternalStatser interface {
	InternalStats() interface{}
}
