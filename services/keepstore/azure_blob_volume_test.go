package main

import (
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
)

const (
	// The same fake credentials used by Microsoft's Azure emulator
	emulatorAccountName = "devstoreaccount1"
	emulatorAccountKey = "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
)

type azStubHandler struct {}

func (azStubHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
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
		log.Println("custom dialer: dial", hp, "instead of", address)
		address = hp
	}
	return d.Dialer.Dial(network, address)
}

type TestableAzureBlobVolume struct {
	*AzureBlobVolume
	azStub *httptest.Server
	t      *testing.T
}

func NewTestableAzureBlobVolume(t *testing.T, readonly bool) *TestableAzureBlobVolume {
	azStub := httptest.NewServer(azStubHandler{})

	stubURLBase := strings.Split(azStub.URL, "://")[1]
	azClient, err := storage.NewClient(emulatorAccountName, emulatorAccountKey, stubURLBase, storage.DefaultAPIVersion, false)
	if err != nil {
		t.Fatal(err)
	}

	v := NewAzureBlobVolume(azClient, "fakecontainername", readonly)

	return &TestableAzureBlobVolume{
		AzureBlobVolume: v,
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
		return NewTestableAzureBlobVolume(t, false)
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
		return NewTestableAzureBlobVolume(t, true)
	})
}

func (v *TestableAzureBlobVolume) PutRaw(locator string, data []byte) {
	v.Put(locator, data)
}

func (v *TestableAzureBlobVolume) TouchWithDate(locator string, lastPut time.Time) {
}

func (v *TestableAzureBlobVolume) Teardown() {
	v.azStub.Close()
}
