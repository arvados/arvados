package main

import (
	"encoding/base64"
	"encoding/xml"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
)

const (
	// The same fake credentials used by Microsoft's Azure emulator
	emulatorAccountName = "devstoreaccount1"
	emulatorAccountKey = "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
)

var azureTestContainer string

func init() {
	flag.StringVar(
		&azureTestContainer,
		"test.azure-storage-container-volume",
		"",
		"Name of Azure container to use for testing. Do not use a container with real data! Use -azure-storage-account-name and -azure-storage-key-file arguments to supply credentials.")
}

type azBlob struct{
	Data        []byte
	Mtime       time.Time
	Uncommitted map[string][]byte
}

type azStubHandler struct {
	sync.Mutex
	blobs  map[string]*azBlob
}

func newAzStubHandler() *azStubHandler {
	return &azStubHandler{
		blobs: make(map[string]*azBlob),
	}
}

func (h *azStubHandler) TouchWithDate(container, hash string, t time.Time) {
	if blob, ok := h.blobs[container + "|" + hash]; !ok {
		return
	} else {
		blob.Mtime = t
	}
}

func (h *azStubHandler) PutRaw(container, hash string, data []byte) {
	h.Lock()
	defer h.Unlock()
	h.blobs[container + "|" + hash] = &azBlob{
		Data: data,
		Mtime: time.Now(),
		Uncommitted: make(map[string][]byte),
	}
}

func (h *azStubHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	h.Lock()
	defer h.Unlock()
	// defer log.Printf("azStubHandler: %+v", r)

	path := strings.Split(r.URL.Path, "/")
	container := path[1]
	hash := ""
	if len(path) > 2 {
		hash = path[2]
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("azStubHandler(%+v): %s", r, err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	type blockListRequestBody struct {
		XMLName     xml.Name `xml:"BlockList"`
		Uncommitted []string
	}

	blob, blobExists := h.blobs[container + "|" + hash]

	switch {
	case r.Method == "PUT" && r.Form.Get("comp") == "" && r.Header.Get("Content-Length") == "0":
		rw.WriteHeader(http.StatusCreated)
		h.blobs[container + "|" + hash] = &azBlob{
			Data:  body,
			Mtime: time.Now(),
			Uncommitted: make(map[string][]byte),
		}
	case r.Method == "PUT" && r.Form.Get("comp") == "block":
		if !blobExists {
			log.Printf("Got block for nonexistent blob: %+v", r)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		blockID, err := base64.StdEncoding.DecodeString(r.Form.Get("blockid"))
		if err != nil || len(blockID) == 0 {
			log.Printf("Invalid blockid: %+q", r.Form.Get("blockid"))
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		blob.Uncommitted[string(blockID)] = body
		rw.WriteHeader(http.StatusCreated)
	case r.Method == "PUT" && r.Form.Get("comp") == "blocklist":
		bl := &blockListRequestBody{}
		if err := xml.Unmarshal(body, bl); err != nil {
			log.Printf("xml Unmarshal: %s", err)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		for _, encBlockID := range bl.Uncommitted {
			blockID, err := base64.StdEncoding.DecodeString(encBlockID)
			if err != nil || len(blockID) == 0 || blob.Uncommitted[string(blockID)] == nil {
				log.Printf("Invalid blockid: %+q", encBlockID)
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			blob.Data = blob.Uncommitted[string(blockID)]
			log.Printf("body %+q, bl %+v, blockID %+q, data %+q", body, bl, blockID, blob.Data)
		}
		rw.WriteHeader(http.StatusCreated)
	case (r.Method == "GET" || r.Method == "HEAD") && hash != "":
		if !blobExists {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		rw.Header().Set("Last-Modified", blob.Mtime.Format(time.RFC1123))
		rw.Header().Set("Content-Length", strconv.Itoa(len(blob.Data)))
		if r.Method == "GET" {
			if _, err := rw.Write(blob.Data); err != nil {
				log.Printf("write %+q: %s", blob.Data, err)
			}
		}
	case r.Method == "DELETE" && hash != "":
		if !blobExists {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		delete(h.blobs, container + "|" + hash)
		rw.WriteHeader(http.StatusAccepted)
	case r.Method == "GET" && r.Form.Get("comp") == "list" && r.Form.Get("restype") == "container":
		prefix := container + "|" + r.Form.Get("prefix")
		marker := r.Form.Get("marker")

		maxResults := 2
		if n, err := strconv.Atoi(r.Form.Get("maxresults")); err == nil && n >= 1 && n <= 5000 {
			maxResults = n
		}

		resp := storage.BlobListResponse{
			Marker: marker,
			NextMarker: "",
			MaxResults: int64(maxResults),
		}
		var hashes sort.StringSlice
		for k := range h.blobs {
			if strings.HasPrefix(k, prefix) {
				hashes = append(hashes, k[len(container)+1:])
			}
		}
		hashes.Sort()
		for _, hash := range hashes {
			if len(resp.Blobs) == maxResults {
				resp.NextMarker = hash
				break
			}
			if len(resp.Blobs) > 0 || marker == "" || marker == hash {
				blob := h.blobs[container + "|" + hash]
				resp.Blobs = append(resp.Blobs, storage.Blob{
					Name: hash,
					Properties: storage.BlobProperties{
						LastModified: blob.Mtime.Format(time.RFC1123),
						ContentLength: int64(len(blob.Data)),
					},
				})
			}
		}
		buf, err := xml.Marshal(resp)
		if err != nil {
			log.Print(err)
			rw.WriteHeader(http.StatusInternalServerError)
		}
		rw.Write(buf)
	default:
		log.Printf("azStubHandler: not implemented: %+v Body:%+q", r, body)
		rw.WriteHeader(http.StatusNotImplemented)
	}
}

// azStubDialer is a net.Dialer that notices when the Azure driver
// tries to connect to "devstoreaccount1.blob.127.0.0.1:46067", and
// in such cases transparently dials "127.0.0.1:46067" instead.
type azStubDialer struct {
	net.Dialer
}

var localHostPortRe = regexp.MustCompile(`(127\.0\.0\.1|localhost|\[::1\]):\d+`)
func (d *azStubDialer) Dial(network, address string) (net.Conn, error) {
	if hp := localHostPortRe.FindString(address); hp != "" {
		log.Println("azStubDialer: dial", hp, "instead of", address)
		address = hp
	}
	return d.Dialer.Dial(network, address)
}

type TestableAzureBlobVolume struct {
	*AzureBlobVolume
	azHandler *azStubHandler
	azStub    *httptest.Server
	t         *testing.T
}

func NewTestableAzureBlobVolume(t *testing.T, readonly bool, replication int) TestableVolume {
	azHandler := newAzStubHandler()
	azStub := httptest.NewServer(azHandler)

	var azClient storage.Client

	container := azureTestContainer
	if container == "" {
		// Connect to stub instead of real Azure storage service
		stubURLBase := strings.Split(azStub.URL, "://")[1]
		var err error
		if azClient, err = storage.NewClient(emulatorAccountName, emulatorAccountKey, stubURLBase, storage.DefaultAPIVersion, false); err != nil {
			t.Fatal(err)
		}
		container = "fakecontainername"
	} else {
		// Connect to real Azure storage service
		accountKey, err := readKeyFromFile(azureStorageAccountKeyFile)
		if err != nil {
			t.Fatal(err)
		}
		azClient, err = storage.NewBasicClient(azureStorageAccountName, accountKey)
		if err != nil {
			t.Fatal(err)
		}
	}

	v := NewAzureBlobVolume(azClient, container, readonly, replication)

	return &TestableAzureBlobVolume{
		AzureBlobVolume: v,
		azHandler: azHandler,
		azStub: azStub,
		t: t,
	}
}

func TestAzureBlobVolumeWithGeneric(t *testing.T) {
	defer func(t http.RoundTripper) {
		http.DefaultTransport = t
	}(http.DefaultTransport)
	http.DefaultTransport = &http.Transport{
		Dial: (&azStubDialer{}).Dial,
	}
	DoGenericVolumeTests(t, func(t *testing.T) TestableVolume {
		return NewTestableAzureBlobVolume(t, false, azureStorageReplication)
	})
}

func TestReadonlyAzureBlobVolumeWithGeneric(t *testing.T) {
	defer func(t http.RoundTripper) {
		http.DefaultTransport = t
	}(http.DefaultTransport)
	http.DefaultTransport = &http.Transport{
		Dial: (&azStubDialer{}).Dial,
	}
	DoGenericVolumeTests(t, func(t *testing.T) TestableVolume {
		return NewTestableAzureBlobVolume(t, true, azureStorageReplication)
	})
}

func TestAzureBlobVolumeReplication(t *testing.T) {
	for r := 1; r <= 4; r++ {
		v := NewTestableAzureBlobVolume(t, false, r)
		defer v.Teardown()
		if n := v.Replication(); n != r {
			t.Errorf("Got replication %d, expected %d", n, r)
		}
	}
}

func (v *TestableAzureBlobVolume) PutRaw(locator string, data []byte) {
	v.azHandler.PutRaw(v.containerName, locator, data)
}

func (v *TestableAzureBlobVolume) TouchWithDate(locator string, lastPut time.Time) {
	v.azHandler.TouchWithDate(v.containerName, locator, lastPut)
}

func (v *TestableAzureBlobVolume) Teardown() {
	v.azStub.Close()
}
