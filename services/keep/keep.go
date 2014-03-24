package main

import (
	"bufio"
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"strings"
)

const DEFAULT_PORT = 25107
const BLOCKSIZE = 64 * 1024 * 1024

var KeepVolumes []string

func main() {
	// Look for local keep volumes.
	KeepVolumes = FindKeepVolumes()
	if len(KeepVolumes) == 0 {
		log.Fatal("could not find any keep volumes")
	}
	for _, v := range KeepVolumes {
		log.Println("keep volume:", v)
	}

	// Set up REST handlers.
	rest := mux.NewRouter()
	rest.HandleFunc("/{hash:[0-9a-f]{32}}", GetBlockHandler).Methods("GET")
	http.Handle("/", rest)

	port := fmt.Sprintf(":%d", DEFAULT_PORT)
	http.ListenAndServe(port, nil)
}

func GetBlockHandler(w http.ResponseWriter, req *http.Request) {
	hash := mux.Vars(req)["hash"]

	block, err := GetBlock(hash)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}

	_, err = w.Write(block)
	if err != nil {
		log.Printf("GetBlockHandler: writing response: %s", err)
	}

	return
}

func GetBlock(hash string) ([]byte, error) {
	var buf = make([]byte, BLOCKSIZE)

	// Attempt to read the requested hash from a keep volume.
	for _, vol := range KeepVolumes {
		var f *os.File
		var err error
		var nread int

		path := fmt.Sprintf("%s/%s/%s", vol, hash[0:3], hash)

		f, err = os.Open(path)
		if err != nil {
			log.Printf("%s: opening %s: %s\n", vol, path, err)
			continue
		}

		nread, err = f.Read(buf)
		if err != nil {
			log.Printf("%s: reading %s: %s\n", vol, path, err)
			continue
		}

		// Double check the file checksum.
		filehash := fmt.Sprintf("%x", md5.Sum(buf[:nread]))
		if filehash != hash {
			log.Printf("%s: checksum mismatch: %s (actual hash %s)\n",
				vol, path, filehash)
			continue
		}

		// Success!
		return buf[:nread], nil
	}

	log.Printf("%s: all keep volumes failed, giving up\n", hash)
	return buf, errors.New("not found: " + hash)
}

// FindKeepVolumes
//     Returns a list of Keep volumes mounted on this system.
//
//     A Keep volume is a normal or tmpfs volume with a /keep
//     directory at the top level of the mount point.
//
func FindKeepVolumes() []string {
	vols := make([]string, 0)

	if f, err := os.Open("/proc/mounts"); err != nil {
		log.Fatal("could not read /proc/mounts: ", err)
	} else {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			args := strings.Fields(scanner.Text())
			dev, mount := args[0], args[1]
			if (dev == "tmpfs" || strings.HasPrefix(dev, "/dev/")) && mount != "/" {
				keep := mount + "/keep"
				if st, err := os.Stat(keep); err == nil && st.IsDir() {
					vols = append(vols, keep)
				}
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}
	return vols
}
