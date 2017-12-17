package mount

import (
	"io"
	"os"
	"sync"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"github.com/curoverse/cgofuse/fuse"
)

type keepFS struct {
	fuse.FileSystemBase
	Client     *arvados.Client
	KeepClient *keepclient.KeepClient
	ReadOnly   bool

	root   arvados.FileSystem
	open   map[uint64]arvados.File
	lastFH uint64
	sync.Mutex
}

var (
	invalidFH = ^uint64(0)
)

func (fs *keepFS) newFH(f arvados.File) uint64 {
	fs.Lock()
	defer fs.Unlock()
	if fs.open == nil {
		fs.open = make(map[uint64]arvados.File)
	}
	fs.lastFH++
	fh := fs.lastFH
	fs.open[fh] = f
	return fh
}

func (fs *keepFS) Init() {
	fs.root = fs.Client.SiteFileSystem(fs.KeepClient)
}

func (fs *keepFS) Create(path string, flags int, mode uint32) (errc int, fh uint64) {
	if fs.ReadOnly {
		return -fuse.EROFS, invalidFH
	}
	f, err := fs.root.OpenFile(path, flags|os.O_CREATE, os.FileMode(mode))
	if err == os.ErrExist {
		return -fuse.EEXIST, invalidFH
	} else if err != nil {
		return -fuse.EINVAL, invalidFH
	}
	return 0, fs.newFH(f)
}

func (fs *keepFS) Open(path string, flags int) (errc int, fh uint64) {
	if fs.ReadOnly && flags&(os.O_RDWR|os.O_WRONLY|os.O_CREATE) != 0 {
		return -fuse.EROFS, invalidFH
	}
	f, err := fs.root.OpenFile(path, flags, 0)
	if err != nil {
		return -fuse.ENOENT, invalidFH
	} else if fi, err := f.Stat(); err != nil {
		return -fuse.EIO, invalidFH
	} else if fi.IsDir() {
		f.Close()
		return -fuse.EISDIR, invalidFH
	}
	return 0, fs.newFH(f)
}

func (fs *keepFS) Utimens(path string, tmsp []fuse.Timespec) int {
	if fs.ReadOnly {
		return -fuse.EROFS
	}
	f, err := fs.root.OpenFile(path, 0, 0)
	if err != nil {
		return fs.errCode(err)
	}
	f.Close()
	return 0
}

func (fs *keepFS) errCode(err error) int {
	if os.IsNotExist(err) {
		return -fuse.ENOENT
	}
	switch err {
	case os.ErrExist:
		return -fuse.EEXIST
	case arvados.ErrInvalidArgument:
		return -fuse.EINVAL
	case arvados.ErrInvalidOperation:
		return -fuse.ENOSYS
	case nil:
		return 0
	default:
		return -fuse.EIO
	}
}

func (fs *keepFS) Mkdir(path string, mode uint32) int {
	if fs.ReadOnly {
		return -fuse.EROFS
	}
	f, err := fs.root.OpenFile(path, os.O_CREATE|os.O_EXCL, os.FileMode(mode)|os.ModeDir)
	if err != nil {
		return fs.errCode(err)
	}
	f.Close()
	return 0
}

func (fs *keepFS) Opendir(path string) (errc int, fh uint64) {
	f, err := fs.root.OpenFile(path, 0, 0)
	if err != nil {
		return fs.errCode(err), invalidFH
	} else if fi, err := f.Stat(); err != nil {
		return fs.errCode(err), invalidFH
	} else if !fi.IsDir() {
		f.Close()
		return -fuse.ENOTDIR, invalidFH
	}
	return 0, fs.newFH(f)
}

func (fs *keepFS) Releasedir(path string, fh uint64) int {
	return fs.Release(path, fh)
}

func (fs *keepFS) Release(path string, fh uint64) int {
	fs.Lock()
	defer fs.Unlock()
	defer delete(fs.open, fh)
	if f := fs.open[fh]; f != nil {
		err := f.Close()
		if err != nil {
			return -fuse.EIO
		}
	}
	return 0
}

func (fs *keepFS) Getattr(path string, stat *fuse.Stat_t, fh uint64) (errc int) {
	fi, err := fs.root.Stat(path)
	if err != nil {
		return -fuse.ENOENT
	}
	fs.fillStat(stat, fi)
	return 0
}

func (*keepFS) fillStat(stat *fuse.Stat_t, fi os.FileInfo) {
	var m uint32
	if fi.IsDir() {
		m = m | fuse.S_IFDIR
	} else {
		m = m | fuse.S_IFREG
	}
	m = m | uint32(fi.Mode()&os.ModePerm)
	stat.Mode = m
	stat.Nlink = 1
	stat.Size = fi.Size()
	t := fuse.NewTimespec(fi.ModTime())
	stat.Mtim = t
	stat.Ctim = t
	stat.Atim = t
	stat.Birthtim = t
	stat.Blksize = 1024
	stat.Blocks = (stat.Size + stat.Blksize - 1) / stat.Blksize
}

func (fs *keepFS) Write(path string, buf []byte, ofst int64, fh uint64) (n int) {
	if fs.ReadOnly {
		return -fuse.EROFS
	}
	f := fs.lookupFH(fh)
	if f == nil {
		return -fuse.EBADF
	}
	_, err := f.Seek(ofst, io.SeekStart)
	if err != nil {
		return -fuse.EINVAL
	}
	n, _ = f.Write(buf)
	return
}

func (fs *keepFS) Read(path string, buf []byte, ofst int64, fh uint64) (n int) {
	f := fs.lookupFH(fh)
	if f == nil {
		return 0
	}
	_, err := f.Seek(ofst, io.SeekStart)
	if err != nil {
		return 0
	}
	n, _ = f.Read(buf)
	return
}

func (fs *keepFS) Readdir(path string,
	fill func(name string, stat *fuse.Stat_t, ofst int64) bool,
	ofst int64,
	fh uint64) (errc int) {
	f := fs.lookupFH(fh)
	if f == nil {
		return -fuse.EBADF
	}
	fill(".", nil, 0)
	fill("..", nil, 0)
	var stat fuse.Stat_t
	fis, err := f.Readdir(-1)
	if err != nil {
		return -fuse.ENOSYS // ???
	}
	for _, fi := range fis {
		fs.fillStat(&stat, fi)
		//fill(fi.Name(), &stat, 0)
		fill(fi.Name(), nil, 0)
	}
	return 0
}

func (fs *keepFS) lookupFH(fh uint64) arvados.File {
	fs.Lock()
	defer fs.Unlock()
	return fs.open[fh]
}
