// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Package keepstore implements the keepstore service component and
// back-end storage drivers.
//
// It is an internal module, only intended to be imported by
// /cmd/arvados-server and other server-side components in this
// repository.
package keepstore

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// Maximum size of a keep block is 64 MiB.
const BlockSize = 1 << 26

var (
	errChecksum          = httpserver.ErrorWithStatus(errors.New("checksum mismatch in stored data"), http.StatusBadGateway)
	errNoTokenProvided   = httpserver.ErrorWithStatus(errors.New("no token provided in Authorization header"), http.StatusUnauthorized)
	errMethodNotAllowed  = httpserver.ErrorWithStatus(errors.New("method not allowed"), http.StatusMethodNotAllowed)
	errVolumeUnavailable = httpserver.ErrorWithStatus(errors.New("volume unavailable"), http.StatusServiceUnavailable)
	errCollision         = httpserver.ErrorWithStatus(errors.New("hash collision"), http.StatusInternalServerError)
	errExpiredSignature  = httpserver.ErrorWithStatus(errors.New("expired signature"), http.StatusUnauthorized)
	errInvalidSignature  = httpserver.ErrorWithStatus(errors.New("invalid signature"), http.StatusBadRequest)
	errInvalidLocator    = httpserver.ErrorWithStatus(errors.New("invalid locator"), http.StatusBadRequest)
	errFull              = httpserver.ErrorWithStatus(errors.New("insufficient storage"), http.StatusInsufficientStorage)
	errTooLarge          = httpserver.ErrorWithStatus(errors.New("request entity too large"), http.StatusRequestEntityTooLarge)
	driver               = make(map[string]volumeDriver)
)

type indexOptions struct {
	MountUUID string
	Prefix    string
	WriteTo   io.Writer
}

type mount struct {
	arvados.KeepMount
	volume
	priority int
}

type keepstore struct {
	cluster    *arvados.Cluster
	logger     logrus.FieldLogger
	serviceURL arvados.URL
	mounts     map[string]*mount
	mountsR    []*mount
	mountsW    []*mount
	bufferPool *bufferPool

	iostats map[volume]*ioStats

	remoteClients    map[string]*keepclient.KeepClient
	remoteClientsMtx sync.Mutex
}

func newKeepstore(ctx context.Context, cluster *arvados.Cluster, token string, reg *prometheus.Registry, serviceURL arvados.URL) (*keepstore, error) {
	logger := ctxlog.FromContext(ctx)

	if cluster.API.MaxConcurrentRequests > 0 && cluster.API.MaxConcurrentRequests < cluster.API.MaxKeepBlobBuffers {
		logger.Warnf("Possible configuration mistake: not useful to set API.MaxKeepBlobBuffers (%d) higher than API.MaxConcurrentRequests (%d)", cluster.API.MaxKeepBlobBuffers, cluster.API.MaxConcurrentRequests)
	}

	if cluster.Collections.BlobSigningKey != "" {
	} else if cluster.Collections.BlobSigning {
		return nil, errors.New("cannot enable Collections.BlobSigning with no Collections.BlobSigningKey")
	} else {
		logger.Warn("Running without a blob signing key. Block locators returned by this server will not be signed, and will be rejected by a server that enforces permissions. To fix this, configure Collections.BlobSigning and Collections.BlobSigningKey.")
	}

	if cluster.API.MaxKeepBlobBuffers <= 0 {
		return nil, fmt.Errorf("API.MaxKeepBlobBuffers must be greater than zero")
	}
	bufferPool := newBufferPool(logger, cluster.API.MaxKeepBlobBuffers, reg)

	ks := &keepstore{
		cluster:       cluster,
		logger:        logger,
		serviceURL:    serviceURL,
		bufferPool:    bufferPool,
		remoteClients: make(map[string]*keepclient.KeepClient),
	}

	err := ks.setupMounts(newVolumeMetricsVecs(reg))
	if err != nil {
		return nil, err
	}

	return ks, nil
}

func (ks *keepstore) setupMounts(metrics *volumeMetricsVecs) error {
	ks.mounts = make(map[string]*mount)
	if len(ks.cluster.Volumes) == 0 {
		return errors.New("no volumes configured")
	}
	for uuid, cfgvol := range ks.cluster.Volumes {
		va, ok := cfgvol.AccessViaHosts[ks.serviceURL]
		if !ok && len(cfgvol.AccessViaHosts) > 0 {
			continue
		}
		dri, ok := driver[cfgvol.Driver]
		if !ok {
			return fmt.Errorf("volume %s: invalid driver %q", uuid, cfgvol.Driver)
		}
		vol, err := dri(newVolumeParams{
			UUID:         uuid,
			Cluster:      ks.cluster,
			ConfigVolume: cfgvol,
			Logger:       ks.logger,
			MetricsVecs:  metrics,
			BufferPool:   ks.bufferPool,
		})
		if err != nil {
			return fmt.Errorf("error initializing volume %s: %s", uuid, err)
		}
		sc := cfgvol.StorageClasses
		if len(sc) == 0 {
			sc = map[string]bool{"default": true}
		}
		repl := cfgvol.Replication
		if repl < 1 {
			repl = 1
		}
		pri := 0
		for class, in := range cfgvol.StorageClasses {
			p := ks.cluster.StorageClasses[class].Priority
			if in && p > pri {
				pri = p
			}
		}
		mnt := &mount{
			volume:   vol,
			priority: pri,
			KeepMount: arvados.KeepMount{
				UUID:           uuid,
				DeviceID:       vol.DeviceID(),
				AllowWrite:     !va.ReadOnly && !cfgvol.ReadOnly,
				AllowTrash:     !va.ReadOnly && (!cfgvol.ReadOnly || cfgvol.AllowTrashWhenReadOnly),
				Replication:    repl,
				StorageClasses: sc,
			},
		}
		ks.mounts[uuid] = mnt
		ks.logger.Printf("started volume %s (%s), AllowWrite=%v, AllowTrash=%v", uuid, vol.DeviceID(), mnt.AllowWrite, mnt.AllowTrash)
	}
	if len(ks.mounts) == 0 {
		return fmt.Errorf("no volumes configured for %s", ks.serviceURL)
	}

	ks.mountsR = nil
	ks.mountsW = nil
	for _, mnt := range ks.mounts {
		ks.mountsR = append(ks.mountsR, mnt)
		if mnt.AllowWrite {
			ks.mountsW = append(ks.mountsW, mnt)
		}
	}
	// Sorting mounts by UUID makes behavior more predictable, and
	// is convenient for testing -- for example, "index all
	// volumes" and "trash block on all volumes" will visit
	// volumes in predictable order.
	sort.Slice(ks.mountsR, func(i, j int) bool { return ks.mountsR[i].UUID < ks.mountsR[j].UUID })
	sort.Slice(ks.mountsW, func(i, j int) bool { return ks.mountsW[i].UUID < ks.mountsW[j].UUID })
	return nil
}

// checkLocatorSignature checks that locator has a valid signature.
// If the BlobSigning config is false, it returns nil even if the
// signature is invalid or missing.
func (ks *keepstore) checkLocatorSignature(ctx context.Context, locator string) error {
	if !ks.cluster.Collections.BlobSigning {
		return nil
	}
	token := ctxToken(ctx)
	if token == "" {
		return errNoTokenProvided
	}
	err := arvados.VerifySignature(locator, token, ks.cluster.Collections.BlobSigningTTL.Duration(), []byte(ks.cluster.Collections.BlobSigningKey))
	if err == arvados.ErrSignatureExpired {
		return errExpiredSignature
	} else if err != nil {
		return errInvalidSignature
	}
	return nil
}

// signLocator signs the locator for the given token, if possible.
// Note this signs if the BlobSigningKey config is available, even if
// the BlobSigning config is false.
func (ks *keepstore) signLocator(token, locator string) string {
	if token == "" || len(ks.cluster.Collections.BlobSigningKey) == 0 {
		return locator
	}
	ttl := ks.cluster.Collections.BlobSigningTTL.Duration()
	return arvados.SignLocator(locator, token, time.Now().Add(ttl), ttl, []byte(ks.cluster.Collections.BlobSigningKey))
}

func (ks *keepstore) BlockRead(ctx context.Context, opts arvados.BlockReadOptions) (n int, err error) {
	li, err := getLocatorInfo(opts.Locator)
	if err != nil {
		return 0, err
	}
	out := opts.WriteTo
	if rw, ok := out.(http.ResponseWriter); ok && li.size > 0 {
		out = &setSizeOnWrite{ResponseWriter: rw, size: li.size}
	}
	if li.remote && !li.signed {
		return ks.blockReadRemote(ctx, opts)
	}
	if err := ks.checkLocatorSignature(ctx, opts.Locator); err != nil {
		return 0, err
	}
	hashcheck := md5.New()
	if li.size > 0 {
		out = newHashCheckWriter(out, hashcheck, int64(li.size), li.hash)
	} else {
		out = io.MultiWriter(out, hashcheck)
	}

	buf, err := ks.bufferPool.GetContext(ctx)
	if err != nil {
		return 0, err
	}
	defer ks.bufferPool.Put(buf)
	streamer := newStreamWriterAt(out, 65536, buf)
	defer streamer.Close()

	var errToCaller error = os.ErrNotExist
	for _, mnt := range ks.rendezvous(li.hash, ks.mountsR) {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		err := mnt.BlockRead(ctx, li.hash, streamer)
		if err != nil {
			if streamer.WroteAt() != 0 {
				// BlockRead encountered an error
				// after writing some data, so it's
				// too late to try another
				// volume. Flush streamer before
				// calling Wrote() to ensure our
				// return value accurately reflects
				// the number of bytes written to
				// opts.WriteTo.
				streamer.Close()
				return streamer.Wrote(), err
			}
			if !os.IsNotExist(err) {
				errToCaller = err
			}
			continue
		}
		if li.size == 0 {
			// hashCheckingWriter isn't in use because we
			// don't know the expected size. All we can do
			// is check after writing all the data, and
			// trust the caller is doing a HEAD request so
			// it's not too late to set an error code in
			// the response header.
			err = streamer.Close()
			if hash := fmt.Sprintf("%x", hashcheck.Sum(nil)); hash != li.hash && err == nil {
				err = errChecksum
			}
			if rw, ok := opts.WriteTo.(http.ResponseWriter); ok {
				// We didn't set the content-length header
				// above because we didn't know the block size
				// until now.
				rw.Header().Set("Content-Length", fmt.Sprintf("%d", streamer.WroteAt()))
			}
			return streamer.WroteAt(), err
		} else if streamer.WroteAt() != li.size {
			// If the backend read fewer bytes than
			// expected but returns no error, we can
			// classify this as a checksum error (even
			// though hashCheckWriter doesn't know that
			// yet, it's just waiting for the next
			// write). If our caller is serving a GET
			// request it's too late to do anything about
			// it anyway, but if it's a HEAD request the
			// caller can still change the response status
			// code.
			return streamer.WroteAt(), errChecksum
		}
		// Ensure streamer flushes all buffered data without
		// errors.
		err = streamer.Close()
		return streamer.Wrote(), err
	}
	return 0, errToCaller
}

func (ks *keepstore) blockReadRemote(ctx context.Context, opts arvados.BlockReadOptions) (int, error) {
	token := ctxToken(ctx)
	if token == "" {
		return 0, errNoTokenProvided
	}
	var remoteClient *keepclient.KeepClient
	var parts []string
	var size int
	for i, part := range strings.Split(opts.Locator, "+") {
		switch {
		case i == 0:
			// don't try to parse hash part as hint
		case strings.HasPrefix(part, "A"):
			// drop local permission hint
			continue
		case len(part) > 7 && part[0] == 'R' && part[6] == '-':
			remoteID := part[1:6]
			remote, ok := ks.cluster.RemoteClusters[remoteID]
			if !ok {
				return 0, httpserver.ErrorWithStatus(errors.New("remote cluster not configured"), http.StatusBadRequest)
			}
			kc, err := ks.remoteClient(remoteID, remote, token)
			if err == auth.ErrObsoleteToken {
				return 0, httpserver.ErrorWithStatus(err, http.StatusBadRequest)
			} else if err != nil {
				return 0, err
			}
			remoteClient = kc
			part = "A" + part[7:]
		case len(part) > 0 && part[0] >= '0' && part[0] <= '9':
			size, _ = strconv.Atoi(part)
		}
		parts = append(parts, part)
	}
	if remoteClient == nil {
		return 0, httpserver.ErrorWithStatus(errors.New("invalid remote hint"), http.StatusBadRequest)
	}
	locator := strings.Join(parts, "+")
	if opts.LocalLocator == nil {
		// Read from remote cluster and stream response back
		// to caller
		if rw, ok := opts.WriteTo.(http.ResponseWriter); ok && size > 0 {
			rw.Header().Set("Content-Length", fmt.Sprintf("%d", size))
		}
		return remoteClient.BlockRead(ctx, arvados.BlockReadOptions{
			Locator: locator,
			WriteTo: opts.WriteTo,
		})
	}
	// We must call LocalLocator before writing any data to
	// opts.WriteTo, otherwise the caller can't put the local
	// locator in a response header.  So we copy into memory,
	// generate the local signature, then copy from memory to
	// opts.WriteTo.
	buf, err := ks.bufferPool.GetContext(ctx)
	if err != nil {
		return 0, err
	}
	defer ks.bufferPool.Put(buf)
	writebuf := bytes.NewBuffer(buf[:0])
	ks.logger.Infof("blockReadRemote(%s): remote read(%s)", opts.Locator, locator)
	_, err = remoteClient.BlockRead(ctx, arvados.BlockReadOptions{
		Locator: locator,
		WriteTo: writebuf,
	})
	if err != nil {
		return 0, err
	}
	resp, err := ks.BlockWrite(ctx, arvados.BlockWriteOptions{
		Hash: locator,
		Data: writebuf.Bytes(),
	})
	if err != nil {
		return 0, err
	}
	opts.LocalLocator(resp.Locator)
	if rw, ok := opts.WriteTo.(http.ResponseWriter); ok {
		rw.Header().Set("Content-Length", fmt.Sprintf("%d", writebuf.Len()))
	}
	n, err := io.Copy(opts.WriteTo, bytes.NewReader(writebuf.Bytes()))
	return int(n), err
}

func (ks *keepstore) remoteClient(remoteID string, remoteCluster arvados.RemoteCluster, token string) (*keepclient.KeepClient, error) {
	ks.remoteClientsMtx.Lock()
	kc, ok := ks.remoteClients[remoteID]
	ks.remoteClientsMtx.Unlock()
	if !ok {
		c := &arvados.Client{
			APIHost:   remoteCluster.Host,
			AuthToken: "xxx",
			Insecure:  remoteCluster.Insecure,
		}
		ac, err := arvadosclient.New(c)
		if err != nil {
			return nil, err
		}
		kc, err = keepclient.MakeKeepClient(ac)
		if err != nil {
			return nil, err
		}
		kc.DiskCacheSize = keepclient.DiskCacheDisabled

		ks.remoteClientsMtx.Lock()
		ks.remoteClients[remoteID] = kc
		ks.remoteClientsMtx.Unlock()
	}
	accopy := *kc.Arvados
	accopy.ApiToken = token
	kccopy := kc.Clone()
	kccopy.Arvados = &accopy
	token, err := auth.SaltToken(token, remoteID)
	if err != nil {
		return nil, err
	}
	kccopy.Arvados.ApiToken = token
	return kccopy, nil
}

// BlockWrite writes a block to one or more volumes.
func (ks *keepstore) BlockWrite(ctx context.Context, opts arvados.BlockWriteOptions) (arvados.BlockWriteResponse, error) {
	var resp arvados.BlockWriteResponse
	var hash string
	if opts.Data == nil {
		buf, err := ks.bufferPool.GetContext(ctx)
		if err != nil {
			return resp, err
		}
		defer ks.bufferPool.Put(buf)
		w := bytes.NewBuffer(buf[:0])
		h := md5.New()
		limitedReader := &io.LimitedReader{R: opts.Reader, N: BlockSize}
		n, err := io.Copy(io.MultiWriter(w, h), limitedReader)
		if err != nil {
			return resp, err
		}
		if limitedReader.N == 0 {
			// Data size is either exactly BlockSize, or too big.
			n, err := opts.Reader.Read(make([]byte, 1))
			if n > 0 {
				return resp, httpserver.ErrorWithStatus(err, http.StatusRequestEntityTooLarge)
			}
			if err != io.EOF {
				return resp, err
			}
		}
		opts.Data = buf[:n]
		if opts.DataSize != 0 && int(n) != opts.DataSize {
			return resp, httpserver.ErrorWithStatus(fmt.Errorf("content length %d did not match specified data size %d", n, opts.DataSize), http.StatusBadRequest)
		}
		hash = fmt.Sprintf("%x", h.Sum(nil))
	} else {
		hash = fmt.Sprintf("%x", md5.Sum(opts.Data))
	}
	if opts.Hash != "" && !strings.HasPrefix(opts.Hash, hash) {
		return resp, httpserver.ErrorWithStatus(fmt.Errorf("content hash %s did not match specified locator %s", hash, opts.Hash), http.StatusBadRequest)
	}
	rvzmounts := ks.rendezvous(hash, ks.mountsW)
	result := newPutProgress(opts.StorageClasses)
	for _, mnt := range rvzmounts {
		if !result.Want(mnt) {
			continue
		}
		cmp := &checkEqual{Expect: opts.Data}
		if err := mnt.BlockRead(ctx, hash, cmp); err == nil {
			if !cmp.Equal() {
				return resp, errCollision
			}
			err := mnt.BlockTouch(hash)
			if err == nil {
				result.Add(mnt)
			}
		}
	}
	var allFull atomic.Bool
	allFull.Store(true)
	// pending tracks what result will be if all outstanding
	// writes succeed.
	pending := result.Copy()
	cond := sync.NewCond(new(sync.Mutex))
	cond.L.Lock()
	var wg sync.WaitGroup
nextmnt:
	for _, mnt := range rvzmounts {
		for {
			if result.Done() || ctx.Err() != nil {
				break nextmnt
			}
			if !result.Want(mnt) {
				continue nextmnt
			}
			if pending.Want(mnt) {
				break
			}
			// This mount might not be needed, depending
			// on the outcome of pending writes. Wait for
			// a pending write to finish, then check
			// again.
			cond.Wait()
		}
		mnt := mnt
		logger := ks.logger.WithField("mount", mnt.UUID)
		pending.Add(mnt)
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Debug("start write")
			err := mnt.BlockWrite(ctx, hash, opts.Data)
			cond.L.Lock()
			defer cond.L.Unlock()
			defer cond.Broadcast()
			if err != nil {
				logger.Debug("write failed")
				pending.Sub(mnt)
				if err != errFull {
					allFull.Store(false)
				}
			} else {
				result.Add(mnt)
				pending.Sub(mnt)
			}
		}()
	}
	cond.L.Unlock()
	wg.Wait()
	if ctx.Err() != nil {
		return resp, ctx.Err()
	}
	if result.Done() || result.totalReplication > 0 {
		resp = arvados.BlockWriteResponse{
			Locator:        ks.signLocator(ctxToken(ctx), fmt.Sprintf("%s+%d", hash, len(opts.Data))),
			Replicas:       result.totalReplication,
			StorageClasses: result.classDone,
		}
		return resp, nil
	}
	if allFull.Load() {
		return resp, errFull
	}
	return resp, errVolumeUnavailable
}

// rendezvous sorts the given mounts by descending priority, then by
// rendezvous order for the given locator.
func (*keepstore) rendezvous(locator string, mnts []*mount) []*mount {
	hash := locator
	if len(hash) > 32 {
		hash = hash[:32]
	}
	// copy the provided []*mount before doing an in-place sort
	mnts = append([]*mount(nil), mnts...)
	weight := make(map[*mount]string)
	for _, mnt := range mnts {
		uuidpart := mnt.UUID
		if len(uuidpart) == 27 {
			// strip zzzzz-yyyyy- prefixes
			uuidpart = uuidpart[12:]
		}
		weight[mnt] = fmt.Sprintf("%x", md5.Sum([]byte(hash+uuidpart)))
	}
	sort.Slice(mnts, func(i, j int) bool {
		if p := mnts[i].priority - mnts[j].priority; p != 0 {
			return p > 0
		}
		return weight[mnts[i]] < weight[mnts[j]]
	})
	return mnts
}

// checkEqual reports whether the data written to it (via io.WriterAt
// interface) is equal to the expected data.
//
// Expect should not be changed after the first Write.
//
// Results are undefined if WriteAt is called with overlapping ranges.
type checkEqual struct {
	Expect   []byte
	equal    atomic.Int64
	notequal atomic.Bool
}

func (ce *checkEqual) Equal() bool {
	return !ce.notequal.Load() && ce.equal.Load() == int64(len(ce.Expect))
}

func (ce *checkEqual) WriteAt(p []byte, offset int64) (int, error) {
	endpos := int(offset) + len(p)
	if offset >= 0 && endpos <= len(ce.Expect) && bytes.Equal(p, ce.Expect[int(offset):endpos]) {
		ce.equal.Add(int64(len(p)))
	} else {
		ce.notequal.Store(true)
	}
	return len(p), nil
}

func (ks *keepstore) BlockUntrash(ctx context.Context, locator string) error {
	li, err := getLocatorInfo(locator)
	if err != nil {
		return err
	}
	var errToCaller error = os.ErrNotExist
	for _, mnt := range ks.mountsW {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := mnt.BlockUntrash(li.hash)
		if err == nil {
			errToCaller = nil
		} else if !os.IsNotExist(err) && errToCaller != nil {
			errToCaller = err
		}
	}
	return errToCaller
}

func (ks *keepstore) BlockTouch(ctx context.Context, locator string) error {
	li, err := getLocatorInfo(locator)
	if err != nil {
		return err
	}
	var errToCaller error = os.ErrNotExist
	for _, mnt := range ks.mountsW {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := mnt.BlockTouch(li.hash)
		if err == nil {
			return nil
		}
		if !os.IsNotExist(err) {
			errToCaller = err
		}
	}
	return errToCaller
}

func (ks *keepstore) BlockTrash(ctx context.Context, locator string) error {
	if !ks.cluster.Collections.BlobTrash {
		return errMethodNotAllowed
	}
	li, err := getLocatorInfo(locator)
	if err != nil {
		return err
	}
	var errToCaller error = os.ErrNotExist
	for _, mnt := range ks.mounts {
		if !mnt.AllowTrash {
			continue
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		t, err := mnt.Mtime(li.hash)
		if err == nil && time.Now().Sub(t) > ks.cluster.Collections.BlobSigningTTL.Duration() {
			err = mnt.BlockTrash(li.hash)
		}
		if os.IsNotExist(errToCaller) || (errToCaller == nil && !os.IsNotExist(err)) {
			errToCaller = err
		}
	}
	return errToCaller
}

func (ks *keepstore) Mounts() []*mount {
	return ks.mountsR
}

func (ks *keepstore) Index(ctx context.Context, opts indexOptions) error {
	mounts := ks.mountsR
	if opts.MountUUID != "" {
		mnt, ok := ks.mounts[opts.MountUUID]
		if !ok {
			return os.ErrNotExist
		}
		mounts = []*mount{mnt}
	}
	for _, mnt := range mounts {
		err := mnt.Index(ctx, opts.Prefix, opts.WriteTo)
		if err != nil {
			return err
		}
	}
	return nil
}

func ctxToken(ctx context.Context) string {
	if c, ok := auth.FromContext(ctx); ok && len(c.Tokens) > 0 {
		return c.Tokens[0]
	} else {
		return ""
	}
}

// locatorInfo expresses the attributes of a locator that are relevant
// for keepstore decision-making.
type locatorInfo struct {
	hash   string
	size   int
	remote bool // locator has a +R hint
	signed bool // locator has a +A hint
}

func getLocatorInfo(loc string) (locatorInfo, error) {
	var li locatorInfo
	plus := 0    // number of '+' chars seen so far
	partlen := 0 // chars since last '+'
	for i, c := range loc + "+" {
		if c == '+' {
			if partlen == 0 {
				// double/leading/trailing '+'
				return li, errInvalidLocator
			}
			if plus == 0 {
				if i != 32 {
					return li, errInvalidLocator
				}
				li.hash = loc[:i]
			}
			if plus == 1 {
				if size, err := strconv.Atoi(loc[i-partlen : i]); err == nil {
					li.size = size
				}
			}
			plus++
			partlen = 0
			continue
		}
		partlen++
		if partlen == 1 {
			if c == 'A' {
				li.signed = true
			}
			if c == 'R' {
				li.remote = true
			}
			if plus > 1 && c >= '0' && c <= '9' {
				// size, if present at all, must come first
				return li, errInvalidLocator
			}
		}
		if plus == 0 && !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			// non-hexadecimal char in hash part
			return li, errInvalidLocator
		}
	}
	return li, nil
}
