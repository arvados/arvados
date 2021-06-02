// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"git.arvados.org/arvados.git/sdk/go/manifest"
)

type printfer interface {
	Printf(string, ...interface{})
}

var errTooManySymlinks = errors.New("too many symlinks, or symlink cycle")

const limitFollowSymlinks = 10

type filetodo struct {
	src  string
	dst  string
	size int64
}

// copier copies data from a finished container's output path to a new
// Arvados collection.
//
// Regular files (and symlinks to regular files) in hostOutputDir are
// copied from the local filesystem.
//
// Symlinks to mounted collections, and any collections mounted under
// ctrOutputDir, are copied by transforming the relevant parts of the
// existing manifests, without moving any data around.
//
// Symlinks to other parts of the container's filesystem result in
// errors.
//
// Use:
//
//	manifest, err := (&copier{...}).Copy()
type copier struct {
	client        *arvados.Client
	arvClient     IArvadosClient
	keepClient    IKeepClient
	hostOutputDir string
	ctrOutputDir  string
	bindmounts    map[string]bindmount
	mounts        map[string]arvados.Mount
	secretMounts  map[string]arvados.Mount
	logger        printfer

	dirs     []string
	files    []filetodo
	manifest string

	manifestCache map[string]*manifest.Manifest
}

// Copy copies data as needed, and returns a new manifest.
func (cp *copier) Copy() (string, error) {
	err := cp.walkMount("", cp.ctrOutputDir, limitFollowSymlinks, true)
	if err != nil {
		return "", fmt.Errorf("error scanning files to copy to output: %v", err)
	}
	fs, err := (&arvados.Collection{ManifestText: cp.manifest}).FileSystem(cp.client, cp.keepClient)
	if err != nil {
		return "", fmt.Errorf("error creating Collection.FileSystem: %v", err)
	}
	for _, d := range cp.dirs {
		err = fs.Mkdir(d, 0777)
		if err != nil && err != os.ErrExist {
			return "", fmt.Errorf("error making directory %q in output collection: %v", d, err)
		}
	}
	var unflushed int64
	var lastparentdir string
	for _, f := range cp.files {
		// If a dir has just had its last file added, do a
		// full Flush. Otherwise, do a partial Flush (write
		// full-size blocks, but leave the last short block
		// open so f's data can be packed with it).
		dir, _ := filepath.Split(f.dst)
		if dir != lastparentdir || unflushed > keepclient.BLOCKSIZE {
			if err := fs.Flush("/"+lastparentdir, dir != lastparentdir); err != nil {
				return "", fmt.Errorf("error flushing output collection file data: %v", err)
			}
			unflushed = 0
		}
		lastparentdir = dir

		n, err := cp.copyFile(fs, f)
		if err != nil {
			return "", fmt.Errorf("error copying file %q into output collection: %v", f, err)
		}
		unflushed += n
	}
	return fs.MarshalManifest(".")
}

func (cp *copier) copyFile(fs arvados.CollectionFileSystem, f filetodo) (int64, error) {
	cp.logger.Printf("copying %q (%d bytes)", f.dst, f.size)
	dst, err := fs.OpenFile(f.dst, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return 0, err
	}
	src, err := os.Open(f.src)
	if err != nil {
		dst.Close()
		return 0, err
	}
	defer src.Close()
	n, err := io.Copy(dst, src)
	if err != nil {
		dst.Close()
		return n, err
	}
	return n, dst.Close()
}

// Append to cp.manifest, cp.files, and cp.dirs so as to copy src (an
// absolute path in the container's filesystem) to dest (an absolute
// path in the output collection, or "" for output root).
//
// src must be (or be a descendant of) a readonly "collection" mount,
// a writable collection mounted at ctrOutputPath, or a "tmp" mount.
//
// If walkMountsBelow is true, include contents of any collection
// mounted below src as well.
func (cp *copier) walkMount(dest, src string, maxSymlinks int, walkMountsBelow bool) error {
	// srcRoot, srcMount indicate the innermost mount that
	// contains src.
	var srcRoot string
	var srcMount arvados.Mount
	for root, mnt := range cp.mounts {
		if len(root) > len(srcRoot) && strings.HasPrefix(src+"/", root+"/") {
			srcRoot, srcMount = root, mnt
		}
	}
	for root := range cp.secretMounts {
		if len(root) > len(srcRoot) && strings.HasPrefix(src+"/", root+"/") {
			// Silently omit secrets, and symlinks to
			// secrets.
			return nil
		}
	}
	if srcRoot == "" {
		return fmt.Errorf("cannot output file %q: not in any mount", src)
	}

	// srcRelPath is the path to the file/dir we are trying to
	// copy, relative to its mount point -- ".", "./foo.txt", ...
	srcRelPath := filepath.Join(".", srcMount.Path, src[len(srcRoot):])

	switch {
	case srcMount.ExcludeFromOutput:
	case srcMount.Kind == "tmp":
		// Handle by walking the host filesystem.
		return cp.walkHostFS(dest, src, maxSymlinks, walkMountsBelow)
	case srcMount.Kind != "collection":
		return fmt.Errorf("%q: unsupported mount %q in output (kind is %q)", src, srcRoot, srcMount.Kind)
	case !srcMount.Writable:
		mft, err := cp.getManifest(srcMount.PortableDataHash)
		if err != nil {
			return err
		}
		cp.manifest += mft.Extract(srcRelPath, dest).Text
	default:
		hostRoot, err := cp.hostRoot(srcRoot)
		if err != nil {
			return err
		}
		f, err := os.Open(filepath.Join(hostRoot, ".arvados#collection"))
		if err != nil {
			return err
		}
		defer f.Close()
		var coll arvados.Collection
		err = json.NewDecoder(f).Decode(&coll)
		if err != nil {
			return err
		}
		mft := manifest.Manifest{Text: coll.ManifestText}
		cp.manifest += mft.Extract(srcRelPath, dest).Text
	}
	if walkMountsBelow {
		return cp.walkMountsBelow(dest, src)
	}
	return nil
}

func (cp *copier) walkMountsBelow(dest, src string) error {
	for mnt, mntinfo := range cp.mounts {
		if !strings.HasPrefix(mnt, src+"/") {
			continue
		}
		if cp.copyRegularFiles(mntinfo) {
			// These got copied into the nearest parent
			// mount as regular files during setup, so
			// they get copied as regular files when we
			// process the parent. Output will reflect any
			// changes and deletions done by the
			// container.
			continue
		}
		// Example: we are processing dest=/foo src=/mnt1/dir1
		// (perhaps we followed a symlink /outdir/foo ->
		// /mnt1/dir1). Caller has already processed the
		// collection mounted at /mnt1, but now we find that
		// /mnt1/dir1/mnt2 is also a mount, so we need to copy
		// src=/mnt1/dir1/mnt2 to dest=/foo/mnt2.
		//
		// We handle all descendants of /mnt1/dir1 in this
		// loop instead of using recursion:
		// /mnt1/dir1/mnt2/mnt3 is a child of both /mnt1 and
		// /mnt1/dir1/mnt2, but we only want to walk it
		// once. (This simplification is safe because mounted
		// collections cannot contain symlinks.)
		err := cp.walkMount(dest+mnt[len(src):], mnt, 0, false)
		if err != nil {
			return err
		}
	}
	return nil
}

// Add entries to cp.dirs and cp.files so as to copy src (an absolute
// path in the container's filesystem which corresponds to a real file
// or directory in cp.hostOutputDir) to dest (an absolute path in the
// output collection, or "" for output root).
//
// Always follow symlinks.
//
// If includeMounts is true, include mounts at and below src.
// Otherwise, skip them.
func (cp *copier) walkHostFS(dest, src string, maxSymlinks int, includeMounts bool) error {
	if includeMounts {
		err := cp.walkMountsBelow(dest, src)
		if err != nil {
			return err
		}
	}

	hostsrc := cp.hostOutputDir + src[len(cp.ctrOutputDir):]

	// If src is a symlink, walk its target.
	fi, err := os.Lstat(hostsrc)
	if err != nil {
		return fmt.Errorf("lstat %q: %s", src, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		if maxSymlinks < 0 {
			return errTooManySymlinks
		}
		target, err := os.Readlink(hostsrc)
		if err != nil {
			return fmt.Errorf("readlink %q: %s", src, err)
		}
		if !strings.HasPrefix(target, "/") {
			target = filepath.Join(filepath.Dir(src), target)
		}
		return cp.walkMount(dest, target, maxSymlinks-1, true)
	}

	// If src is a regular directory, append it to cp.dirs and
	// walk each of its children. (If there are no children,
	// create an empty file "dest/.keep".)
	if fi.Mode().IsDir() {
		if dest != "" {
			cp.dirs = append(cp.dirs, dest)
		}
		dir, err := os.Open(hostsrc)
		if err != nil {
			return fmt.Errorf("open %q: %s", src, err)
		}
		names, err := dir.Readdirnames(-1)
		dir.Close()
		if err != nil {
			return fmt.Errorf("readdirnames %q: %s", src, err)
		}
		if len(names) == 0 {
			if dest != "" {
				cp.files = append(cp.files, filetodo{
					src: os.DevNull,
					dst: dest + "/.keep",
				})
			}
			return nil
		}
		sort.Strings(names)
		for _, name := range names {
			dest, src := dest+"/"+name, src+"/"+name
			if _, isSecret := cp.secretMounts[src]; isSecret {
				continue
			}
			if mntinfo, isMount := cp.mounts[src]; isMount && !cp.copyRegularFiles(mntinfo) {
				// If a regular file/dir somehow
				// exists at a path that's also a
				// mount target, ignore the file --
				// the mount has already been included
				// with walkMountsBelow().
				//
				// (...except mount types that are
				// handled as regular files.)
				continue
			}
			err = cp.walkHostFS(dest, src, maxSymlinks, false)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// If src is a regular file, append it to cp.files.
	if fi.Mode().IsRegular() {
		cp.files = append(cp.files, filetodo{
			src:  hostsrc,
			dst:  dest,
			size: fi.Size(),
		})
		return nil
	}
	cp.logger.Printf("Skipping unsupported file type (mode %o) in output dir: %q", fi.Mode(), src)
	return nil
}

// Return the host path that was mounted at the given path in the
// container.
func (cp *copier) hostRoot(ctrRoot string) (string, error) {
	if ctrRoot == cp.ctrOutputDir {
		return cp.hostOutputDir, nil
	}
	if mnt, ok := cp.bindmounts[ctrRoot]; ok {
		return mnt.HostPath, nil
	}
	return "", fmt.Errorf("not bind-mounted: %q", ctrRoot)
}

func (cp *copier) copyRegularFiles(m arvados.Mount) bool {
	return m.Kind == "text" || m.Kind == "json" || (m.Kind == "collection" && m.Writable)
}

func (cp *copier) getManifest(pdh string) (*manifest.Manifest, error) {
	if mft, ok := cp.manifestCache[pdh]; ok {
		return mft, nil
	}
	var coll arvados.Collection
	err := cp.arvClient.Get("collections", pdh, nil, &coll)
	if err != nil {
		return nil, fmt.Errorf("error retrieving collection record for %q: %s", pdh, err)
	}
	mft := &manifest.Manifest{Text: coll.ManifestText}
	if cp.manifestCache == nil {
		cp.manifestCache = map[string]*manifest.Manifest{pdh: mft}
	} else {
		cp.manifestCache[pdh] = mft
	}
	return mft, nil
}
