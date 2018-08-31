// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"context"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/config"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"golang.org/x/crypto/ssh"
	check "gopkg.in/check.v1"
)

type AzureProviderSuite struct{}

var _ = check.Suite(&AzureProviderSuite{})

type VirtualMachinesClientStub struct{}

func (*VirtualMachinesClientStub) CreateOrUpdate(ctx context.Context,
	resourceGroupName string,
	VMName string,
	parameters compute.VirtualMachine) (result compute.VirtualMachine, err error) {
	parameters.ID = &VMName
	parameters.Name = &VMName
	return parameters, nil
}

func (*VirtualMachinesClientStub) Delete(ctx context.Context, resourceGroupName string, VMName string) (result *http.Response, err error) {
	return nil, nil
}

func (*VirtualMachinesClientStub) ListComplete(ctx context.Context, resourceGroupName string) (result compute.VirtualMachineListResultIterator, err error) {
	return compute.VirtualMachineListResultIterator{}, nil
}

type InterfacesClientStub struct{}

func (*InterfacesClientStub) CreateOrUpdate(ctx context.Context,
	resourceGroupName string,
	nicName string,
	parameters network.Interface) (result network.Interface, err error) {
	parameters.ID = to.StringPtr(nicName)
	(*parameters.IPConfigurations)[0].PrivateIPAddress = to.StringPtr("192.168.5.5")
	return parameters, nil
}

func (*InterfacesClientStub) Delete(ctx context.Context, resourceGroupName string, VMName string) (result *http.Response, err error) {
	return nil, nil
}

func (*InterfacesClientStub) ListComplete(ctx context.Context, resourceGroupName string) (result network.InterfaceListResultIterator, err error) {
	return network.InterfaceListResultIterator{}, nil
}

var live = flag.String("live-azure-cfg", "", "Test with real azure API, provide config file")

func GetProvider() (InstanceProvider, ImageID, arvados.Cluster, error) {
	cluster := arvados.Cluster{
		InstanceTypes: arvados.InstanceTypeMap(map[string]arvados.InstanceType{
			"tiny": arvados.InstanceType{
				Name:         "tiny",
				ProviderType: "Standard_D1_v2",
				VCPUs:        1,
				RAM:          4000000000,
				Scratch:      10000000000,
				Price:        .02,
				Preemptible:  false,
			},
		})}
	if *live != "" {
		cfg := AzureProviderConfig{}
		err := config.LoadFile(&cfg, *live)
		if err != nil {
			return nil, ImageID(""), cluster, err
		}
		ap, err := NewAzureProvider(cfg, "test123")
		return ap, ImageID(cfg.Image), cluster, err
	} else {
		ap := AzureProvider{
			azconfig: AzureProviderConfig{
				BlobContainer: "vhds",
			},
			dispatcherID: "test123",
			namePrefix:   "compute-test123-",
		}
		ap.vmClient = &VirtualMachinesClientStub{}
		ap.netClient = &InterfacesClientStub{}
		return &ap, ImageID("blob"), cluster, nil
	}
}

func (*AzureProviderSuite) TestCreate(c *check.C) {
	ap, img, cluster, err := GetProvider()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	f, err := os.Open("azconfig_sshkey.pub")
	c.Assert(err, check.IsNil)

	keybytes, err := ioutil.ReadAll(f)
	c.Assert(err, check.IsNil)

	pk, _, _, _, err := ssh.ParseAuthorizedKey(keybytes)
	c.Assert(err, check.IsNil)

	inst, err := ap.Create(context.Background(),
		cluster.InstanceTypes["tiny"],
		img, map[string]string{"tag1": "bleep"},
		pk)

	c.Assert(err, check.IsNil)

	log.Printf("Result %v %v", inst.String(), inst.Address())

}

func (*AzureProviderSuite) TestListInstances(c *check.C) {
	ap, _, _, err := GetProvider()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	l, err := ap.Instances(context.Background())

	c.Assert(err, check.IsNil)

	for _, i := range l {
		tg, _ := i.Tags(context.Background())
		log.Printf("%v %v %v", i.String(), i.Address(), tg)
	}
}

func (*AzureProviderSuite) TestManageNics(c *check.C) {
	ap, _, _, err := GetProvider()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	ap.(*AzureProvider).ManageNics(context.Background())
}

func (*AzureProviderSuite) TestManageBlobs(c *check.C) {
	ap, _, _, err := GetProvider()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	ap.(*AzureProvider).ManageBlobs(context.Background())
}

func (*AzureProviderSuite) TestDestroyInstances(c *check.C) {
	ap, _, _, err := GetProvider()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	l, err := ap.Instances(context.Background())
	c.Assert(err, check.IsNil)

	for _, i := range l {
		c.Check(i.Destroy(context.Background()), check.IsNil)
	}
}

func (*AzureProviderSuite) TestDeleteFake(c *check.C) {
	ap, _, _, err := GetProvider()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	_, err = ap.(*AzureProvider).netClient.Delete(context.Background(), "fakefakefake", "fakefakefake")

	de, ok := err.(autorest.DetailedError)
	if ok {
		rq := de.Original.(*azure.RequestError)

		log.Printf("%v %q %q", rq.Response.StatusCode, rq.ServiceError.Code, rq.ServiceError.Message)
	}
}

func (*AzureProviderSuite) TestWrapError(c *check.C) {
	retryError := autorest.DetailedError{
		Original: &azure.RequestError{
			DetailedError: autorest.DetailedError{
				Response: &http.Response{
					StatusCode: 429,
					Header:     map[string][]string{"Retry-After": []string{"123"}},
				},
			},
			ServiceError: &azure.ServiceError{},
		},
	}
	wrapped := WrapAzureError(retryError)
	_, ok := wrapped.(RateLimitError)
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
	wrapped = WrapAzureError(quotaError)
	_, ok = wrapped.(QuotaError)
	c.Check(ok, check.Equals, true)
}

func (*AzureProviderSuite) TestSetTags(c *check.C) {
	ap, _, _, err := GetProvider()
	if err != nil {
		c.Fatal("Error making provider", err)
	}
	l, err := ap.Instances(context.Background())
	c.Assert(err, check.IsNil)

	if len(l) > 0 {
		err = l[0].SetTags(context.Background(), map[string]string{"foo": "bar"})
		if err != nil {
			c.Fatal("Error setting tags", err)
		}
	}
	l, err = ap.Instances(context.Background())
	c.Assert(err, check.IsNil)

	if len(l) > 0 {
		tg, _ := l[0].Tags(context.Background())
		log.Printf("tags are %v", tg)
	}
}

func (*AzureProviderSuite) TestSSH(c *check.C) {
	ap, _, _, err := GetProvider()
	if err != nil {
		c.Fatal("Error making provider", err)
	}
	l, err := ap.Instances(context.Background())
	c.Assert(err, check.IsNil)

	if len(l) > 0 {

		sshclient, err := SetupSSHClient(c, l[0].Address()+":2222")
		c.Assert(err, check.IsNil)

		sess, err := sshclient.NewSession()
		c.Assert(err, check.IsNil)

		out, err := sess.Output("ls /")
		c.Assert(err, check.IsNil)

		log.Printf("%v", out)

		sshclient.Conn.Close()
	}
}

func SetupSSHClient(c *check.C, addr string) (*ssh.Client, error) {
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

	/*if wkr.publicKey == nil || !bytes.Equal(wkr.publicKey.Marshal(), receivedKey.Marshal()) {
		err = wkr.instance.VerifyPublicKey(receivedKey, client)
		if err != nil {
			return nil, err
		}
		wkr.publicKey = receivedKey
	}*/
	return client, nil
}
