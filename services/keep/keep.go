package main

import (
	"bufio"
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"strings"
)

const DEFAULT_ADDR = ":25107"
const BLOCKSIZE = 64 * 1024 * 1024

var PROC_MOUNTS = "/proc/mounts"

var KeepVolumes []string

type KeepError struct {
	HTTPCode int
	Err      error
}

func (e *KeepError) Error() string {
	return fmt.Sprintf("Error %d: %s", e.HTTPCode, e.Err.Error())
}

func main() {
	// Parse command-line flags.
	var listen, keepvols string
	flag.StringVar(&listen, "listen", DEFAULT_ADDR,
		"interface on which to listen for requests")
	flag.StringVar(&keepvols, "volumes", "",
		"comma-separated list of directories to use for Keep volumes")
	flag.Parse()

	// Look for local keep volumes.
	if keepvols == "" {
		KeepVolumes = FindKeepVolumes()
	} else {
		KeepVolumes = strings.Split(keepvols, ",")
	}

	if len(KeepVolumes) == 0 {
		log.Fatal("could not find any keep volumes")
	}
	for _, v := range KeepVolumes {
		log.Println("keep volume:", v)
	}

	// Set up REST handlers.
	//
	// Start with a router that will route each URL path to an
	// appropriate handler.
	//
	rest := mux.NewRouter()
	rest.HandleFunc("/{hash:[0-9a-f]{32}}", GetBlockHandler).Methods("GET")
	rest.HandleFunc("/{hash:[0-9a-f]{32}}", PutBlockHandler).Methods("PUT")

	// Tell the built-in HTTP server to direct all requests to the REST
	// router.
	http.Handle("/", rest)

	// Start listening for requests.
	http.ListenAndServe(listen, nil)
}

// FindKeepVolumes
//     Returns a list of Keep volumes mounted on this system.
//
//     A Keep volume is a normal or tmpfs volume with a /keep
//     directory at the top level of the mount point.
//
func FindKeepVolumes() []string {
	vols := make([]string, 0)

	if f, err := os.Open(PROC_MOUNTS); err != nil {
		log.Fatalf("opening %s: %s\n", PROC_MOUNTS, err)
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

func PutBlockHandler(w http.ResponseWriter, req *http.Request) {
	hash := mux.Vars(req)["hash"]

	// Read the block data to be stored.
	// TODO(twp): decide what to do when the input stream contains
	// more than BLOCKSIZE bytes.
	//
	buf := make([]byte, BLOCKSIZE)
	if nread, err := req.Body.Read(buf); err == nil {
		if err := PutBlock(buf[:nread], hash); err == nil {
			w.WriteHeader(http.StatusOK)
		} else {
			ke := err.(*KeepError)
			http.Error(w, ke.Error(), ke.HTTPCode)
		}
	} else {
		log.Println("error reading request: ", err)
		http.Error(w, err.Error(), 500)
	}
}

func GetBlock(hash string) ([]byte, error) {
	var buf = make([]byte, BLOCKSIZE)

	// Attempt to read the requested hash from a keep volume.
	for _, vol := range KeepVolumes {
		var f *os.File
		var err error
		var nread int

		blockFilename := fmt.Sprintf("%s/%s/%s", vol, hash[0:3], hash)

		f, err = os.Open(blockFilename)
		if err != nil {
			if !os.IsNotExist(err) {
				// A block is stored on only one Keep disk,
				// so os.IsNotExist is expected.  Report any other errors.
				log.Printf("%s: opening %s: %s\n", vol, blockFilename, err)
			}
			continue
		}

		nread, err = f.Read(buf)
		if err != nil {
			log.Printf("%s: reading %s: %s\n", vol, blockFilename, err)
			continue
		}

		// Double check the file checksum.
		//
		filehash := fmt.Sprintf("%x", md5.Sum(buf[:nread]))
		if filehash != hash {
			// TODO(twp): this condition probably represents a bad disk and
			// should raise major alarm bells for an administrator: e.g.
			// they should be sent directly to an event manager at high
			// priority or logged as urgent problems.
			//
			log.Printf("%s: checksum mismatch: %s (actual hash %s)\n",
				vol, blockFilename, filehash)
			continue
		}

		// Success!
		return buf[:nread], nil
	}

	log.Printf("%s: not found on any volumes, giving up\n", hash)
	return buf, &KeepError{404, errors.New("not found: " + hash)}
}

/* PutBlock(block, hash)
   Stores the BLOCK (identified by the content id HASH) in Keep.

   The MD5 checksum of the block must be identical to the content id HASH.
   If not, an error is returned.

   PutBlock stores the BLOCK on the first Keep volume with free space.
   A failure code is returned to the user only if all volumes fail.

   On success, PutBlock returns nil.
   On failure, it returns a KeepError with one of the following codes:

   401 MD5Fail
         -- The MD5 hash of the BLOCK does not match the argument HASH.
   503 Full
         -- There was not enough space left in any Keep volume to store
            the object.
   500 Fail
         -- The object could not be stored for some other reason (e.g.
            all writes failed). The text of the error message should
            provide as much detail as possible.
*/

func PutBlock(block []byte, hash string) error {
	// Check that BLOCK's checksum matches HASH.
	blockhash := fmt.Sprintf("%x", md5.Sum(block))
	if blockhash != hash {
		log.Printf("%s: MD5 checksum %s did not match request", hash, blockhash)
		return &KeepError{401, errors.New("MD5Fail")}
	}

	for _, vol := range KeepVolumes {

		// TODO(twp): check for a full volume here before trying to write.

		blockDir := fmt.Sprintf("%s/%s", vol, hash[0:3])
		if err := os.MkdirAll(blockDir, 0755); err != nil {
			log.Printf("%s: could not create directory %s: %s",
				hash, blockDir, err)
			continue
		}

		blockFilename := fmt.Sprintf("%s/%s", blockDir, hash)
		f, err := os.OpenFile(blockFilename, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			// if the block already exists, just return success.
			// TODO(twp): should we check here whether the file on disk
			// matches the file we were asked to store?
			if os.IsExist(err) {
				return nil
			} else {
				// Open failed for some other reason.
				log.Printf("%s: creating %s: %s\n", vol, blockFilename, err)
				continue
			}
		}

		if _, err := f.Write(block); err == nil {
			f.Close()
			return nil
		} else {
			log.Printf("%s: writing to %s: %s\n", vol, blockFilename, err)
			continue
		}
	}

	// All volumes failed; report the failure and return an error.
	//
	log.Printf("all Keep volumes failed")
	return &KeepError{500, errors.New("Fail")}
}
