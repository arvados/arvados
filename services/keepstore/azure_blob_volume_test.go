// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	check "gopkg.in/check.v1"
)

const (
	// This cannot be the fake account name "devstoreaccount1"
	// used by Microsoft's Azure emulator: the Azure SDK
	// recognizes that magic string and changes its behavior to
	// cater to the Azure SDK's own test suite.
	fakeAccountName = "fakeaccountname"
	fakeAccountKey  = "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
)

var (
	azureTestContainer string
	azureTestDebug     = os.Getenv("ARVADOS_DEBUG") != ""
)

func init() {
	flag.StringVar(
		&azureTestContainer,
		"test.azure-storage-container-volume",
		"",
		"Name of Azure container to use for testing. Do not use a container with real data! Use -azure-storage-account-name and -azure-storage-key-file arguments to supply credentials.")
}

type azBlob struct {
	Data        []byte
	Etag        string
	Metadata    map[string]string
	Mtime       time.Time
	Uncommitted map[string][]byte
}

type azStubHandler struct {
	sync.Mutex
	logger     logrus.FieldLogger
	blobs      map[string]*azBlob
	race       chan chan struct{}
	didlist503 bool
}

func newAzStubHandler(c *check.C) *azStubHandler {
	return &azStubHandler{
		blobs:  make(map[string]*azBlob),
		logger: ctxlog.TestLogger(c),
	}
}

func (h *azStubHandler) TouchWithDate(container, hash string, t time.Time) {
	blob, ok := h.blobs[container+"|"+hash]
	if !ok {
		return
	}
	blob.Mtime = t
}

func (h *azStubHandler) PutRaw(container, hash string, data []byte) {
	h.Lock()
	defer h.Unlock()
	h.blobs[container+"|"+hash] = &azBlob{
		Data:        data,
		Mtime:       time.Now(),
		Metadata:    make(map[string]string),
		Uncommitted: make(map[string][]byte),
	}
}

func (h *azStubHandler) unlockAndRace() {
	if h.race == nil {
		return
	}
	h.Unlock()
	// Signal caller that race is starting by reading from
	// h.race. If we get a channel, block until that channel is
	// ready to receive. If we get nil (or h.race is closed) just
	// proceed.
	if c := <-h.race; c != nil {
		c <- struct{}{}
	}
	h.Lock()
}

var rangeRegexp = regexp.MustCompile(`^bytes=(\d+)-(\d+)$`)

func (h *azStubHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	h.Lock()
	defer h.Unlock()
	if azureTestDebug {
		defer h.logger.Printf("azStubHandler: %+v", r)
	}

	path := strings.Split(r.URL.Path, "/")
	container := path[1]
	hash := ""
	if len(path) > 2 {
		hash = path[2]
	}

	if err := r.ParseForm(); err != nil {
		h.logger.Printf("azStubHandler(%+v): %s", r, err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	if (r.Method == "PUT" || r.Method == "POST") && r.Header.Get("Content-Length") == "" {
		rw.WriteHeader(http.StatusLengthRequired)
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

	blob, blobExists := h.blobs[container+"|"+hash]

	switch {
	case r.Method == "PUT" && r.Form.Get("comp") == "":
		// "Put Blob" API
		if _, ok := h.blobs[container+"|"+hash]; !ok {
			// Like the real Azure service, we offer a
			// race window during which other clients can
			// list/get the new blob before any data is
			// committed.
			h.blobs[container+"|"+hash] = &azBlob{
				Mtime:       time.Now(),
				Uncommitted: make(map[string][]byte),
				Metadata:    make(map[string]string),
				Etag:        makeEtag(),
			}
			h.unlockAndRace()
		}
		metadata := make(map[string]string)
		for k, v := range r.Header {
			if strings.HasPrefix(strings.ToLower(k), "x-ms-meta-") {
				name := k[len("x-ms-meta-"):]
				metadata[strings.ToLower(name)] = v[0]
			}
		}
		h.blobs[container+"|"+hash] = &azBlob{
			Data:        body,
			Mtime:       time.Now(),
			Uncommitted: make(map[string][]byte),
			Metadata:    metadata,
			Etag:        makeEtag(),
		}
		rw.WriteHeader(http.StatusCreated)
	case r.Method == "PUT" && r.Form.Get("comp") == "block":
		// "Put Block" API
		if !blobExists {
			h.logger.Printf("Got block for nonexistent blob: %+v", r)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		blockID, err := base64.StdEncoding.DecodeString(r.Form.Get("blockid"))
		if err != nil || len(blockID) == 0 {
			h.logger.Printf("Invalid blockid: %+q", r.Form.Get("blockid"))
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		blob.Uncommitted[string(blockID)] = body
		rw.WriteHeader(http.StatusCreated)
	case r.Method == "PUT" && r.Form.Get("comp") == "blocklist":
		// "Put Block List" API
		bl := &blockListRequestBody{}
		if err := xml.Unmarshal(body, bl); err != nil {
			h.logger.Printf("xml Unmarshal: %s", err)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		for _, encBlockID := range bl.Uncommitted {
			blockID, err := base64.StdEncoding.DecodeString(encBlockID)
			if err != nil || len(blockID) == 0 || blob.Uncommitted[string(blockID)] == nil {
				h.logger.Printf("Invalid blockid: %+q", encBlockID)
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			blob.Data = blob.Uncommitted[string(blockID)]
			blob.Etag = makeEtag()
			blob.Mtime = time.Now()
			delete(blob.Uncommitted, string(blockID))
		}
		rw.WriteHeader(http.StatusCreated)
	case r.Method == "PUT" && r.Form.Get("comp") == "metadata":
		// "Set Metadata Headers" API. We don't bother
		// stubbing "Get Metadata Headers": AzureBlobVolume
		// sets metadata headers only as a way to bump Etag
		// and Last-Modified.
		if !blobExists {
			h.logger.Printf("Got metadata for nonexistent blob: %+v", r)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		blob.Metadata = make(map[string]string)
		for k, v := range r.Header {
			if strings.HasPrefix(strings.ToLower(k), "x-ms-meta-") {
				name := k[len("x-ms-meta-"):]
				blob.Metadata[strings.ToLower(name)] = v[0]
			}
		}
		blob.Mtime = time.Now()
		blob.Etag = makeEtag()
	case (r.Method == "GET" || r.Method == "HEAD") && r.Form.Get("comp") == "metadata" && hash != "":
		// "Get Blob Metadata" API
		if !blobExists {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		for k, v := range blob.Metadata {
			rw.Header().Set(fmt.Sprintf("x-ms-meta-%s", k), v)
		}
		return
	case (r.Method == "GET" || r.Method == "HEAD") && hash != "":
		// "Get Blob" API
		if !blobExists {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		data := blob.Data
		if rangeSpec := rangeRegexp.FindStringSubmatch(r.Header.Get("Range")); rangeSpec != nil {
			b0, err0 := strconv.Atoi(rangeSpec[1])
			b1, err1 := strconv.Atoi(rangeSpec[2])
			if err0 != nil || err1 != nil || b0 >= len(data) || b1 >= len(data) || b0 > b1 {
				rw.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", len(data)))
				rw.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			rw.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", b0, b1, len(data)))
			rw.WriteHeader(http.StatusPartialContent)
			data = data[b0 : b1+1]
		}
		rw.Header().Set("Last-Modified", blob.Mtime.Format(time.RFC1123))
		rw.Header().Set("Content-Length", strconv.Itoa(len(data)))
		if r.Method == "GET" {
			if _, err := rw.Write(data); err != nil {
				h.logger.Printf("write %+q: %s", data, err)
			}
		}
		h.unlockAndRace()
	case r.Method == "DELETE" && hash != "":
		// "Delete Blob" API
		if !blobExists {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		delete(h.blobs, container+"|"+hash)
		rw.WriteHeader(http.StatusAccepted)
	case r.Method == "GET" && r.Form.Get("comp") == "list" && r.Form.Get("restype") == "container":
		// "List Blobs" API
		if !h.didlist503 {
			h.didlist503 = true
			rw.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		prefix := container + "|" + r.Form.Get("prefix")
		marker := r.Form.Get("marker")

		maxResults := 2
		if n, err := strconv.Atoi(r.Form.Get("maxresults")); err == nil && n >= 1 && n <= 5000 {
			maxResults = n
		}

		resp := storage.BlobListResponse{
			Marker:     marker,
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
				blob := h.blobs[container+"|"+hash]
				bmeta := map[string]string(nil)
				if r.Form.Get("include") == "metadata" {
					bmeta = blob.Metadata
				}
				b := storage.Blob{
					Name: hash,
					Properties: storage.BlobProperties{
						LastModified:  storage.TimeRFC1123(blob.Mtime),
						ContentLength: int64(len(blob.Data)),
						Etag:          blob.Etag,
					},
					Metadata: bmeta,
				}
				resp.Blobs = append(resp.Blobs, b)
			}
		}
		buf, err := xml.Marshal(resp)
		if err != nil {
			h.logger.Error(err)
			rw.WriteHeader(http.StatusInternalServerError)
		}
		rw.Write(buf)
	default:
		h.logger.Printf("azStubHandler: not implemented: %+v Body:%+q", r, body)
		rw.WriteHeader(http.StatusNotImplemented)
	}
}

// azStubDialer is a net.Dialer that notices when the Azure driver
// tries to connect to "devstoreaccount1.blob.127.0.0.1:46067", and
// in such cases transparently dials "127.0.0.1:46067" instead.
type azStubDialer struct {
	logger logrus.FieldLogger
	net.Dialer
}

var localHostPortRe = regexp.MustCompile(`(127\.0\.0\.1|localhost|\[::1\]):\d+`)

func (d *azStubDialer) Dial(network, address string) (net.Conn, error) {
	if hp := localHostPortRe.FindString(address); hp != "" {
		if azureTestDebug {
			d.logger.Debug("azStubDialer: dial", hp, "instead of", address)
		}
		address = hp
	}
	return d.Dialer.Dial(network, address)
}

type TestableAzureBlobVolume struct {
	*AzureBlobVolume
	azHandler *azStubHandler
	azStub    *httptest.Server
	t         TB
}

func (s *StubbedAzureBlobSuite) newTestableAzureBlobVolume(t TB, cluster *arvados.Cluster, volume arvados.Volume, metrics *volumeMetricsVecs) *TestableAzureBlobVolume {
	azHandler := newAzStubHandler(t.(*check.C))
	azStub := httptest.NewServer(azHandler)

	var azClient storage.Client
	var err error

	container := azureTestContainer
	if container == "" {
		// Connect to stub instead of real Azure storage service
		stubURLBase := strings.Split(azStub.URL, "://")[1]
		if azClient, err = storage.NewClient(fakeAccountName, fakeAccountKey, stubURLBase, storage.DefaultAPIVersion, false); err != nil {
			t.Fatal(err)
		}
		container = "fakecontainername"
	} else {
		// Connect to real Azure storage service
		if azClient, err = storage.NewBasicClient(os.Getenv("ARVADOS_TEST_AZURE_ACCOUNT_NAME"), os.Getenv("ARVADOS_TEST_AZURE_ACCOUNT_KEY")); err != nil {
			t.Fatal(err)
		}
	}
	azClient.Sender = &singleSender{}

	bs := azClient.GetBlobService()
	v := &AzureBlobVolume{
		ContainerName:        container,
		WriteRaceInterval:    arvados.Duration(time.Millisecond),
		WriteRacePollTime:    arvados.Duration(time.Nanosecond),
		ListBlobsMaxAttempts: 2,
		ListBlobsRetryDelay:  arvados.Duration(time.Millisecond),
		azClient:             azClient,
		container:            &azureContainer{ctr: bs.GetContainerReference(container)},
		cluster:              cluster,
		volume:               volume,
		logger:               ctxlog.TestLogger(t),
		metrics:              metrics,
	}
	if err = v.check(); err != nil {
		t.Fatal(err)
	}

	return &TestableAzureBlobVolume{
		AzureBlobVolume: v,
		azHandler:       azHandler,
		azStub:          azStub,
		t:               t,
	}
}

var _ = check.Suite(&StubbedAzureBlobSuite{})

type StubbedAzureBlobSuite struct {
	origHTTPTransport http.RoundTripper
}

func (s *StubbedAzureBlobSuite) SetUpTest(c *check.C) {
	s.origHTTPTransport = http.DefaultTransport
	http.DefaultTransport = &http.Transport{
		Dial: (&azStubDialer{logger: ctxlog.TestLogger(c)}).Dial,
	}
}

func (s *StubbedAzureBlobSuite) TearDownTest(c *check.C) {
	http.DefaultTransport = s.origHTTPTransport
}

func (s *StubbedAzureBlobSuite) TestAzureBlobVolumeWithGeneric(c *check.C) {
	DoGenericVolumeTests(c, false, func(t TB, cluster *arvados.Cluster, volume arvados.Volume, logger logrus.FieldLogger, metrics *volumeMetricsVecs) TestableVolume {
		return s.newTestableAzureBlobVolume(t, cluster, volume, metrics)
	})
}

func (s *StubbedAzureBlobSuite) TestAzureBlobVolumeConcurrentRanges(c *check.C) {
	// Test (BlockSize mod azureMaxGetBytes)==0 and !=0 cases
	for _, b := range []int{2 << 22, 2<<22 - 1} {
		DoGenericVolumeTests(c, false, func(t TB, cluster *arvados.Cluster, volume arvados.Volume, logger logrus.FieldLogger, metrics *volumeMetricsVecs) TestableVolume {
			v := s.newTestableAzureBlobVolume(t, cluster, volume, metrics)
			v.MaxGetBytes = b
			return v
		})
	}
}

func (s *StubbedAzureBlobSuite) TestReadonlyAzureBlobVolumeWithGeneric(c *check.C) {
	DoGenericVolumeTests(c, false, func(c TB, cluster *arvados.Cluster, volume arvados.Volume, logger logrus.FieldLogger, metrics *volumeMetricsVecs) TestableVolume {
		return s.newTestableAzureBlobVolume(c, cluster, volume, metrics)
	})
}

func (s *StubbedAzureBlobSuite) TestAzureBlobVolumeRangeFenceposts(c *check.C) {
	v := s.newTestableAzureBlobVolume(c, testCluster(c), arvados.Volume{Replication: 3}, newVolumeMetricsVecs(prometheus.NewRegistry()))
	defer v.Teardown()

	for _, size := range []int{
		2<<22 - 1, // one <max read
		2 << 22,   // one =max read
		2<<22 + 1, // one =max read, one <max
		2 << 23,   // two =max reads
		BlockSize - 1,
		BlockSize,
	} {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte((i + 7) & 0xff)
		}
		hash := fmt.Sprintf("%x", md5.Sum(data))
		err := v.Put(context.Background(), hash, data)
		if err != nil {
			c.Error(err)
		}
		gotData := make([]byte, len(data))
		gotLen, err := v.Get(context.Background(), hash, gotData)
		if err != nil {
			c.Error(err)
		}
		gotHash := fmt.Sprintf("%x", md5.Sum(gotData))
		if gotLen != size {
			c.Errorf("length mismatch: got %d != %d", gotLen, size)
		}
		if gotHash != hash {
			c.Errorf("hash mismatch: got %s != %s", gotHash, hash)
		}
	}
}

func (s *StubbedAzureBlobSuite) TestAzureBlobVolumeCreateBlobRace(c *check.C) {
	v := s.newTestableAzureBlobVolume(c, testCluster(c), arvados.Volume{Replication: 3}, newVolumeMetricsVecs(prometheus.NewRegistry()))
	defer v.Teardown()

	var wg sync.WaitGroup

	v.azHandler.race = make(chan chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := v.Put(context.Background(), TestHash, TestBlock)
		if err != nil {
			c.Error(err)
		}
	}()
	continuePut := make(chan struct{})
	// Wait for the stub's Put to create the empty blob
	v.azHandler.race <- continuePut
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, len(TestBlock))
		_, err := v.Get(context.Background(), TestHash, buf)
		if err != nil {
			c.Error(err)
		}
	}()
	// Wait for the stub's Get to get the empty blob
	close(v.azHandler.race)
	// Allow stub's Put to continue, so the real data is ready
	// when the volume's Get retries
	<-continuePut
	// Wait for Get() and Put() to finish
	wg.Wait()
}

func (s *StubbedAzureBlobSuite) TestAzureBlobVolumeCreateBlobRaceDeadline(c *check.C) {
	v := s.newTestableAzureBlobVolume(c, testCluster(c), arvados.Volume{Replication: 3}, newVolumeMetricsVecs(prometheus.NewRegistry()))
	v.AzureBlobVolume.WriteRaceInterval.Set("2s")
	v.AzureBlobVolume.WriteRacePollTime.Set("5ms")
	defer v.Teardown()

	v.PutRaw(TestHash, nil)

	buf := new(bytes.Buffer)
	v.IndexTo("", buf)
	if buf.Len() != 0 {
		c.Errorf("Index %+q should be empty", buf.Bytes())
	}

	v.TouchWithDate(TestHash, time.Now().Add(-1982*time.Millisecond))

	allDone := make(chan struct{})
	go func() {
		defer close(allDone)
		buf := make([]byte, BlockSize)
		n, err := v.Get(context.Background(), TestHash, buf)
		if err != nil {
			c.Error(err)
			return
		}
		if n != 0 {
			c.Errorf("Got %+q, expected empty buf", buf[:n])
		}
	}()
	select {
	case <-allDone:
	case <-time.After(time.Second):
		c.Error("Get should have stopped waiting for race when block was 2s old")
	}

	buf.Reset()
	v.IndexTo("", buf)
	if !bytes.HasPrefix(buf.Bytes(), []byte(TestHash+"+0")) {
		c.Errorf("Index %+q should have %+q", buf.Bytes(), TestHash+"+0")
	}
}

func (s *StubbedAzureBlobSuite) TestAzureBlobVolumeContextCancelGet(c *check.C) {
	s.testAzureBlobVolumeContextCancel(c, func(ctx context.Context, v *TestableAzureBlobVolume) error {
		v.PutRaw(TestHash, TestBlock)
		_, err := v.Get(ctx, TestHash, make([]byte, BlockSize))
		return err
	})
}

func (s *StubbedAzureBlobSuite) TestAzureBlobVolumeContextCancelPut(c *check.C) {
	s.testAzureBlobVolumeContextCancel(c, func(ctx context.Context, v *TestableAzureBlobVolume) error {
		return v.Put(ctx, TestHash, make([]byte, BlockSize))
	})
}

func (s *StubbedAzureBlobSuite) TestAzureBlobVolumeContextCancelCompare(c *check.C) {
	s.testAzureBlobVolumeContextCancel(c, func(ctx context.Context, v *TestableAzureBlobVolume) error {
		v.PutRaw(TestHash, TestBlock)
		return v.Compare(ctx, TestHash, TestBlock2)
	})
}

func (s *StubbedAzureBlobSuite) testAzureBlobVolumeContextCancel(c *check.C, testFunc func(context.Context, *TestableAzureBlobVolume) error) {
	v := s.newTestableAzureBlobVolume(c, testCluster(c), arvados.Volume{Replication: 3}, newVolumeMetricsVecs(prometheus.NewRegistry()))
	defer v.Teardown()
	v.azHandler.race = make(chan chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	allDone := make(chan struct{})
	go func() {
		defer close(allDone)
		err := testFunc(ctx, v)
		if err != context.Canceled {
			c.Errorf("got %T %q, expected %q", err, err, context.Canceled)
		}
	}()
	releaseHandler := make(chan struct{})
	select {
	case <-allDone:
		c.Error("testFunc finished without waiting for v.azHandler.race")
	case <-time.After(10 * time.Second):
		c.Error("timed out waiting to enter handler")
	case v.azHandler.race <- releaseHandler:
	}

	cancel()

	select {
	case <-time.After(10 * time.Second):
		c.Error("timed out waiting to cancel")
	case <-allDone:
	}

	go func() {
		<-releaseHandler
	}()
}

func (s *StubbedAzureBlobSuite) TestStats(c *check.C) {
	volume := s.newTestableAzureBlobVolume(c, testCluster(c), arvados.Volume{Replication: 3}, newVolumeMetricsVecs(prometheus.NewRegistry()))
	defer volume.Teardown()

	stats := func() string {
		buf, err := json.Marshal(volume.InternalStats())
		c.Check(err, check.IsNil)
		return string(buf)
	}

	c.Check(stats(), check.Matches, `.*"Ops":0,.*`)
	c.Check(stats(), check.Matches, `.*"Errors":0,.*`)

	loc := "acbd18db4cc2f85cedef654fccc4a4d8"
	_, err := volume.Get(context.Background(), loc, make([]byte, 3))
	c.Check(err, check.NotNil)
	c.Check(stats(), check.Matches, `.*"Ops":[^0],.*`)
	c.Check(stats(), check.Matches, `.*"Errors":[^0],.*`)
	c.Check(stats(), check.Matches, `.*"storage\.AzureStorageServiceError 404 \(404 Not Found\)":[^0].*`)
	c.Check(stats(), check.Matches, `.*"InBytes":0,.*`)

	err = volume.Put(context.Background(), loc, []byte("foo"))
	c.Check(err, check.IsNil)
	c.Check(stats(), check.Matches, `.*"OutBytes":3,.*`)
	c.Check(stats(), check.Matches, `.*"CreateOps":1,.*`)

	_, err = volume.Get(context.Background(), loc, make([]byte, 3))
	c.Check(err, check.IsNil)
	_, err = volume.Get(context.Background(), loc, make([]byte, 3))
	c.Check(err, check.IsNil)
	c.Check(stats(), check.Matches, `.*"InBytes":6,.*`)
}

func (v *TestableAzureBlobVolume) PutRaw(locator string, data []byte) {
	v.azHandler.PutRaw(v.ContainerName, locator, data)
}

func (v *TestableAzureBlobVolume) TouchWithDate(locator string, lastPut time.Time) {
	v.azHandler.TouchWithDate(v.ContainerName, locator, lastPut)
}

func (v *TestableAzureBlobVolume) Teardown() {
	v.azStub.Close()
}

func (v *TestableAzureBlobVolume) ReadWriteOperationLabelValues() (r, w string) {
	return "get", "create"
}

func makeEtag() string {
	return fmt.Sprintf("0x%x", rand.Int63())
}
