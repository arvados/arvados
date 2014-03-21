package main

import (
	"bufio"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

const DEFAULT_PORT = 25107

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
	rest.HandleFunc("/{hash}", GetBlock).Methods("GET")
	http.Handle("/", rest)

	port := fmt.Sprintf(":%d", DEFAULT_PORT)
	http.ListenAndServe(port, nil)
}

func GetBlock(w http.ResponseWriter, req *http.Request) {
	hash := mux.Vars(req)["hash"]

	// Attempt to read the requested hash from a keep volume.
	for _, vol := range KeepVolumes {
		path := fmt.Sprintf("%s/%s/%s", vol, hash[0:3], hash)
		if f, err := os.Open(path); err == nil {
			io.Copy(w, f)
			break
		} else {
			log.Printf("%s: reading block %s: %s\n", vol, hash, err)
		}
	}
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
