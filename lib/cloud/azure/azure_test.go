// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
//
//
// How to manually run individual tests against the real cloud:
//
// $ go test -v git.arvados.org/arvados.git/lib/cloud/azure -live-azure-cfg azconfig.yml -check.f=TestCreate
//
// Tests should be run individually and in the order they are listed in the file:
//
// Example azconfig.yml:
//
// ImageIDForTestSuite: "https://example.blob.core.windows.net/system/Microsoft.Compute/Images/images/zzzzz-compute-osDisk.XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX.vhd"
// DriverParameters:
// 	 SubscriptionID: XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX
// 	 ClientID: XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX
// 	 Location: centralus
// 	 CloudEnvironment: AzurePublicCloud
// 	 ClientSecret: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
// 	 TenantId: XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX
// 	 ResourceGroup: zzzzz
// 	 Network: zzzzz
// 	 Subnet: zzzzz-subnet-private
// 	 StorageAccount: example
// 	 BlobContainer: vhds
// 	 DeleteDanglingResourcesAfter: 20s
//	 AdminUsername: crunch

package azure

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/lib/dispatchcloud/test"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/config"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

type AzureInstanceSetSuite struct{}

var _ = check.Suite(&AzureInstanceSetSuite{})

const testNamePrefix = "compute-test123-"

type VirtualMachinesClientStub struct{}

func (*VirtualMachinesClientStub) createOrUpdate(ctx context.Context,
	resourceGroupName string,
	VMName string,
	parameters compute.VirtualMachine) (result compute.VirtualMachine, err error) {
	parameters.ID = &VMName
	parameters.Name = &VMName
	return parameters, nil
}

func (*VirtualMachinesClientStub) delete(ctx context.Context, resourceGroupName string, VMName string) (result *http.Response, err error) {
	return nil, nil
}

func (*VirtualMachinesClientStub) listComplete(ctx context.Context, resourceGroupName string) (result compute.VirtualMachineListResultIterator, err error) {
	return compute.VirtualMachineListResultIterator{}, nil
}

type InterfacesClientStub struct{}

func (*InterfacesClientStub) createOrUpdate(ctx context.Context,
	resourceGroupName string,
	nicName string,
	parameters network.Interface) (result network.Interface, err error) {
	parameters.ID = to.StringPtr(nicName)
	(*parameters.IPConfigurations)[0].PrivateIPAddress = to.StringPtr("192.168.5.5")
	return parameters, nil
}

func (*InterfacesClientStub) delete(ctx context.Context, resourceGroupName string, VMName string) (result *http.Response, err error) {
	return nil, nil
}

func (*InterfacesClientStub) listComplete(ctx context.Context, resourceGroupName string) (result network.InterfaceListResultIterator, err error) {
	return network.InterfaceListResultIterator{}, nil
}

type BlobContainerStub struct{}

func (*BlobContainerStub) GetBlobReference(name string) *storage.Blob {
	return nil
}

func (*BlobContainerStub) ListBlobs(params storage.ListBlobsParameters) (storage.BlobListResponse, error) {
	return storage.BlobListResponse{}, nil
}

type testConfig struct {
	ImageIDForTestSuite string
	DriverParameters    json.RawMessage
}

var live = flag.String("live-azure-cfg", "", "Test with real azure API, provide config file")

func GetInstanceSet() (cloud.InstanceSet, cloud.ImageID, arvados.Cluster, error) {
	cluster := arvados.Cluster{
		InstanceTypes: arvados.InstanceTypeMap(map[string]arvados.InstanceType{
			"tiny": {
				Name:         "tiny",
				ProviderType: "Standard_D1_v2",
				VCPUs:        1,
				RAM:          4000000000,
				Scratch:      10000000000,
				Price:        .02,
				Preemptible:  false,
			},
			"tinyp": {
				Name:         "tiny",
				ProviderType: "Standard_D1_v2",
				VCPUs:        1,
				RAM:          4000000000,
				Scratch:      10000000000,
				Price:        .002,
				Preemptible:  true,
			},
		})}
	if *live != "" {
		var exampleCfg testConfig
		err := config.LoadFile(&exampleCfg, *live)
		if err != nil {
			return nil, cloud.ImageID(""), cluster, err
		}

		ap, err := newAzureInstanceSet(exampleCfg.DriverParameters, "test123", nil, logrus.StandardLogger())
		return ap, cloud.ImageID(exampleCfg.ImageIDForTestSuite), cluster, err
	}
	ap := azureInstanceSet{
		azconfig: azureInstanceSetConfig{
			BlobContainer: "vhds",
		},
		dispatcherID: "test123",
		namePrefix:   testNamePrefix,
		logger:       logrus.StandardLogger(),
		deleteNIC:    make(chan string),
		deleteBlob:   make(chan storage.Blob),
		deleteDisk:   make(chan compute.Disk),
	}
	ap.ctx, ap.stopFunc = context.WithCancel(context.Background())
	ap.vmClient = &VirtualMachinesClientStub{}
	ap.netClient = &InterfacesClientStub{}
	ap.blobcont = &BlobContainerStub{}
	return &ap, cloud.ImageID("blob"), cluster, nil
}

func (*AzureInstanceSetSuite) TestCreate(c *check.C) {
	ap, img, cluster, err := GetInstanceSet()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	pk, _ := test.LoadTestKey(c, "../../dispatchcloud/test/sshkey_dispatch")
	c.Assert(err, check.IsNil)

	inst, err := ap.Create(cluster.InstanceTypes["tiny"],
		img, map[string]string{
			"TestTagName": "test tag value",
		}, "umask 0600; echo -n test-file-data >/var/run/test-file", pk)

	c.Assert(err, check.IsNil)

	tags := inst.Tags()
	c.Check(tags["TestTagName"], check.Equals, "test tag value")
	c.Logf("inst.String()=%v Address()=%v Tags()=%v", inst.String(), inst.Address(), tags)

	instPreemptable, err := ap.Create(cluster.InstanceTypes["tinyp"],
		img, map[string]string{
			"TestTagName": "test tag value",
		}, "umask 0600; echo -n test-file-data >/var/run/test-file", pk)

	c.Assert(err, check.IsNil)

	tags = instPreemptable.Tags()
	c.Check(tags["TestTagName"], check.Equals, "test tag value")
	c.Logf("instPreemptable.String()=%v Address()=%v Tags()=%v", instPreemptable.String(), instPreemptable.Address(), tags)

}

func (*AzureInstanceSetSuite) TestListInstances(c *check.C) {
	ap, _, _, err := GetInstanceSet()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	l, err := ap.Instances(nil)

	c.Assert(err, check.IsNil)

	for _, i := range l {
		tg := i.Tags()
		log.Printf("%v %v %v", i.String(), i.Address(), tg)
	}
}

func (*AzureInstanceSetSuite) TestManageNics(c *check.C) {
	ap, _, _, err := GetInstanceSet()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	ap.(*azureInstanceSet).manageNics()
	ap.Stop()
}

func (*AzureInstanceSetSuite) TestManageBlobs(c *check.C) {
	ap, _, _, err := GetInstanceSet()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	ap.(*azureInstanceSet).manageBlobs()
	ap.Stop()
}

func (*AzureInstanceSetSuite) TestDestroyInstances(c *check.C) {
	ap, _, _, err := GetInstanceSet()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	l, err := ap.Instances(nil)
	c.Assert(err, check.IsNil)

	for _, i := range filterInstances(c, l) {
		c.Check(i.Destroy(), check.IsNil)
	}
}

func (*AzureInstanceSetSuite) TestDeleteFake(c *check.C) {
	ap, _, _, err := GetInstanceSet()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	_, err = ap.(*azureInstanceSet).netClient.delete(context.Background(), "fakefakefake", "fakefakefake")

	de, ok := err.(autorest.DetailedError)
	if ok {
		rq := de.Original.(*azure.RequestError)

		log.Printf("%v %q %q", rq.Response.StatusCode, rq.ServiceError.Code, rq.ServiceError.Message)
	}
}

func (*AzureInstanceSetSuite) TestWrapError(c *check.C) {
	retryError := autorest.DetailedError{
		Original: &azure.RequestError{
			DetailedError: autorest.DetailedError{
				Response: &http.Response{
					StatusCode: 429,
					Header:     map[string][]string{"Retry-After": {"123"}},
				},
			},
			ServiceError: &azure.ServiceError{},
		},
	}
	wrapped := wrapAzureError(retryError)
	_, ok := wrapped.(cloud.RateLimitError)
	c.Check(ok, check.Equals, true)

	quotaError := autorest.DetailedError{
		Original: &azure.RequestError{
			DetailedError: autorest.DetailedError{
				Response: &http.Response{
					StatusCode: 503,
				},
			},
			ServiceError: &azure.ServiceError{
				Message: "No more quota",
			},
		},
	}
	wrapped = wrapAzureError(quotaError)
	_, ok = wrapped.(cloud.QuotaError)
	c.Check(ok, check.Equals, true)
}

func (*AzureInstanceSetSuite) TestSetTags(c *check.C) {
	ap, _, _, err := GetInstanceSet()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	l, err := ap.Instances(nil)
	c.Assert(err, check.IsNil)
	l = filterInstances(c, l)
	if len(l) > 0 {
		err = l[0].SetTags(map[string]string{"foo": "bar"})
		if err != nil {
			c.Fatal("Error setting tags", err)
		}
	}

	l, err = ap.Instances(nil)
	c.Assert(err, check.IsNil)
	l = filterInstances(c, l)

	if len(l) > 0 {
		tg := l[0].Tags()
		log.Printf("tags are %v", tg)
	}
}

func (*AzureInstanceSetSuite) TestSSH(c *check.C) {
	ap, _, _, err := GetInstanceSet()
	if err != nil {
		c.Fatal("Error making provider", err)
	}
	l, err := ap.Instances(nil)
	c.Assert(err, check.IsNil)
	l = filterInstances(c, l)

	if len(l) > 0 {
		sshclient, err := SetupSSHClient(c, l[0])
		c.Assert(err, check.IsNil)
		defer sshclient.Conn.Close()

		sess, err := sshclient.NewSession()
		c.Assert(err, check.IsNil)
		defer sess.Close()
		_, err = sess.Output("find /var/run/test-file -maxdepth 0 -user root -perm 0600")
		c.Assert(err, check.IsNil)

		sess, err = sshclient.NewSession()
		c.Assert(err, check.IsNil)
		defer sess.Close()
		out, err := sess.Output("sudo cat /var/run/test-file")
		c.Assert(err, check.IsNil)
		c.Check(string(out), check.Equals, "test-file-data")
	}
}

func SetupSSHClient(c *check.C, inst cloud.Instance) (*ssh.Client, error) {
	addr := inst.Address() + ":2222"
	if addr == "" {
		return nil, errors.New("instance has no address")
	}

	f, err := os.Open("azconfig_sshkey")
	c.Assert(err, check.IsNil)

	keybytes, err := ioutil.ReadAll(f)
	c.Assert(err, check.IsNil)

	priv, err := ssh.ParsePrivateKey(keybytes)
	c.Assert(err, check.IsNil)

	var receivedKey ssh.PublicKey
	client, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{
		User: "crunch",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(priv),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			receivedKey = key
			return nil
		},
		Timeout: time.Minute,
	})

	if err != nil {
		return nil, err
	} else if receivedKey == nil {
		return nil, errors.New("BUG: key was never provided to HostKeyCallback")
	}

	err = inst.VerifyHostKey(receivedKey, client)
	c.Assert(err, check.IsNil)

	return client, nil
}

func filterInstances(c *check.C, instances []cloud.Instance) []cloud.Instance {
	var r []cloud.Instance
	for _, i := range instances {
		if !strings.HasPrefix(i.String(), testNamePrefix) {
			c.Logf("ignoring instance %s", i)
			continue
		}
		r = append(r, i)
	}
	return r
}
