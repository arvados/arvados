// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"git.arvados.org/arvados.git/sdk/go/manifest"
	"github.com/bmatcuk/doublestar/v4"
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
	keepClient    IKeepClient
	hostOutputDir string
	ctrOutputDir  string
	globs         []string
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
	collfs, err := (&arvados.Collection{ManifestText: cp.manifest}).FileSystem(cp.client, cp.keepClient)
	if err != nil {
		return "", fmt.Errorf("error creating Collection.FileSystem: %v", err)
	}

	// Remove files/dirs that don't match globs (the ones that
	// were added during cp.walkMount() by copying subtree
	// manifests into cp.manifest).
	err = cp.applyGlobsToCollectionFS(collfs)
	if err != nil {
		return "", fmt.Errorf("error while removing non-matching files from output collection: %w", err)
	}
	// Remove files/dirs that don't match globs (the ones that are
	// stored on the local filesystem and would need to be copied
	// in copyFile() below).
	cp.applyGlobsToFilesAndDirs()
	for _, d := range cp.dirs {
		err = collfs.Mkdir(d, 0777)
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
			if err := collfs.Flush("/"+lastparentdir, dir != lastparentdir); err != nil {
				return "", fmt.Errorf("error flushing output collection file data: %v", err)
			}
			unflushed = 0
		}
		lastparentdir = dir

		n, err := cp.copyFile(collfs, f)
		if err != nil {
			return "", fmt.Errorf("error copying file %q into output collection: %v", f, err)
		}
		unflushed += n
	}
	return collfs.MarshalManifest(".")
}

func (cp *copier) matchGlobs(path string, isDir bool) bool {
	// An entry in the top level of the output directory looks
	// like "/foo", but globs look like "foo", so we strip the
	// leading "/" before matching.
	path = strings.TrimLeft(path, "/")
	for _, glob := range cp.globs {
		if !isDir && strings.HasSuffix(glob, "/**") {
			// doublestar.Match("f*/**", "ff") and
			// doublestar.Match("f*/**", "ff/gg") both
			// return true, but (to be compatible with
			// bash shopt) "ff" should match only if it is
			// a directory.
			//
			// To avoid errant matches, we add the file's
			// basename to the end of the pattern:
			//
			// Match("f*/**/ff", "ff") => false
			// Match("f*/**/gg", "ff/gg") => true
			//
			// Of course, we need to escape basename in
			// case it contains *, ?, \, etc.
			_, name := filepath.Split(path)
			escapedName := strings.TrimSuffix(strings.Replace(name, "", "\\", -1), "\\")
			if match, _ := doublestar.Match(glob+"/"+escapedName, path); match {
				return true
			}
		} else if match, _ := doublestar.Match(glob, path); match {
			return true
		} else if isDir {
			// Workaround doublestar bug (v4.6.1).
			// "foo*/**" should match "foo", but does not,
			// because isZeroLengthPattern does not accept
			// "*/**" as a zero length pattern.
			if trunc := strings.TrimSuffix(glob, "*/**"); trunc != glob {
				if match, _ := doublestar.Match(trunc, path); match {
					return true
				}
			}
		}
	}
	return false
}

// Delete entries from cp.files that do not match cp.globs.
//
// Delete entries from cp.dirs that do not match cp.globs.
//
// Ensure parent/ancestor directories of remaining cp.files and
// cp.dirs entries are still present in cp.dirs, even if they do not
// match cp.globs themselves.
func (cp *copier) applyGlobsToFilesAndDirs() {
	if len(cp.globs) == 0 {
		return
	}
	keepdirs := make(map[string]bool)
	for _, path := range cp.dirs {
		if cp.matchGlobs(path, true) {
			keepdirs[path] = true
		}
	}
	for path := range keepdirs {
		for i, c := range path {
			if i > 0 && c == '/' {
				keepdirs[path[:i]] = true
			}
		}
	}
	var keepfiles []filetodo
	for _, file := range cp.files {
		if cp.matchGlobs(file.dst, false) {
			keepfiles = append(keepfiles, file)
		}
	}
	for _, file := range keepfiles {
		for i, c := range file.dst {
			if i > 0 && c == '/' {
				keepdirs[file.dst[:i]] = true
			}
		}
	}
	cp.dirs = nil
	for path := range keepdirs {
		cp.dirs = append(cp.dirs, path)
	}
	sort.Strings(cp.dirs)
	cp.files = keepfiles
}

// Delete files in collfs that do not match cp.globs.  Also delete
// directories that are empty (after deleting non-matching files) and
// do not match cp.globs themselves.
func (cp *copier) applyGlobsToCollectionFS(collfs arvados.CollectionFileSystem) error {
	if len(cp.globs) == 0 {
		return nil
	}
	include := make(map[string]bool)
	err := fs.WalkDir(arvados.FS(collfs), "", func(path string, ent fs.DirEntry, err error) error {
		if cp.matchGlobs(path, ent.IsDir()) {
			for i, c := range path {
				if i > 0 && c == '/' {
					include[path[:i]] = true
				}
			}
			include[path] = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	err = fs.WalkDir(arvados.FS(collfs), "", func(path string, ent fs.DirEntry, err error) error {
		if err != nil || path == "" {
			return err
		}
		if !include[path] {
			err := collfs.RemoveAll(path)
			if err != nil {
				return err
			}
			if ent.IsDir() {
				return fs.SkipDir
			}
		}
		return nil
	})
	return err
}

// Return true if it's possible for any descendant of the given path
// to match anything in cp.globs.  Used by walkMount to avoid loading
// collections that are mounted underneath ctrOutputPath but excluded
// by globs.
func (cp *copier) subtreeCouldMatch(path string) bool {
	if len(cp.globs) == 0 {
		return true
	}
	pathdepth := 1 + strings.Count(path, "/")
	for _, glob := range cp.globs {
		globdepth := 0
		lastsep := 0
		for i, c := range glob {
			if c != '/' || !doublestar.ValidatePattern(glob[:i]) {
				// Escaped "/", or "/" in a character
				// class, is not a path separator.
				continue
			}
			if glob[lastsep:i] == "**" {
				return true
			}
			lastsep = i + 1
			if globdepth++; globdepth == pathdepth {
				if match, _ := doublestar.Match(glob[:i]+"/*", path+"/z"); match {
					return true
				}
				break
			}
		}
		if globdepth < pathdepth && glob[lastsep:] == "**" {
			return true
		}
	}
	return false
}

func (cp *copier) copyFile(fs arvados.CollectionFileSystem, f filetodo) (int64, error) {
	cp.logger.Printf("copying %q (%d bytes)", strings.TrimLeft(f.dst, "/"), f.size)
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

	// outputRelPath is the destination path relative to the
	// output directory. Used for logging and glob matching.
	var outputRelPath = ""
	if strings.HasPrefix(src, cp.ctrOutputDir) {
		outputRelPath = strings.TrimPrefix(src[len(cp.ctrOutputDir):], "/")
	}
	if outputRelPath == "" {
		// blank means copy a whole directory, so replace it
		// with a wildcard to make it a little clearer what's
		// going on since outputRelPath is only used for logging
		outputRelPath = "*"
	}

	switch {
	case srcMount.ExcludeFromOutput:
	case outputRelPath != "*" && !cp.subtreeCouldMatch(outputRelPath):
		cp.logger.Printf("not copying %q because contents cannot match output globs", outputRelPath)
		return nil
	case srcMount.Kind == "tmp":
		// Handle by walking the host filesystem.
		return cp.walkHostFS(dest, src, maxSymlinks, walkMountsBelow)
	case srcMount.Kind != "collection":
		return fmt.Errorf("%q: unsupported mount %q in output (kind is %q)", src, srcRoot, srcMount.Kind)
	case !srcMount.Writable:
		cp.logger.Printf("copying %q from %v/%v", outputRelPath, srcMount.PortableDataHash, strings.TrimPrefix(srcRelPath, "./"))
		mft, err := cp.getManifest(srcMount.PortableDataHash)
		if err != nil {
			return err
		}
		cp.manifest += mft.Extract(srcRelPath, dest).Text
	default:
		cp.logger.Printf("copying %q", outputRelPath)
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
			} else if isMount && !cp.subtreeCouldMatch(src[len(cp.ctrOutputDir)+1:]) {
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
	err := cp.client.RequestAndDecode(&coll, "GET", "arvados/v1/collections/"+pdh, nil, nil)
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
