// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"context"
	"flag"
	"log"
	"net/http"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/config"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
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

func GetProvider() (Provider, ImageID, error) {
	if *live != "" {
		cfg := AzureProviderConfig{}
		err := config.LoadFile(&cfg, *live)
		if err != nil {
			return nil, ImageID(""), err
		}
		ap, err := NewAzureProvider(cfg, arvados.Cluster{})
		return ap, ImageID(cfg.Image), err
	} else {
		ap := AzureProvider{
			azconfig: AzureProviderConfig{
				BlobContainer: "vhds",
			},
		}
		ap.vmClient = &VirtualMachinesClientStub{}
		ap.netClient = &InterfacesClientStub{}
		return &ap, ImageID("blob"), nil
	}
}

func (*AzureProviderSuite) TestCreate(c *check.C) {
	ap, img, err := GetProvider()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	inst, err := ap.Create(context.Background(),
		arvados.InstanceType{
			Name:         "tiny",
			ProviderType: "Standard_D1_v2",
			VCPUs:        1,
			RAM:          4000000000,
			Scratch:      10000000000,
			Price:        .02,
			Preemptible:  false,
		},
		img,
		[]InstanceTag{"tag1"})

	c.Assert(err, check.IsNil)

	log.Printf("Result %v %v", inst.String(), inst.Address())
}

func (*AzureProviderSuite) TestListInstances(c *check.C) {
	ap, _, err := GetProvider()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	l, err := ap.Instances(context.Background())

	c.Assert(err, check.IsNil)

	for _, i := range l {
		log.Printf("%v %v", i.String(), i.Address())
	}
}

func (*AzureProviderSuite) TestManageNics(c *check.C) {
	ap, _, err := GetProvider()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	ap.(*AzureProvider).ManageNics(context.Background())
}

func (*AzureProviderSuite) TestManageBlobs(c *check.C) {
	ap, _, err := GetProvider()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	ap.(*AzureProvider).ManageBlobs(context.Background())
}

func (*AzureProviderSuite) TestDestroyInstances(c *check.C) {
	ap, _, err := GetProvider()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	l, err := ap.Instances(context.Background())
	c.Assert(err, check.IsNil)

	for _, i := range l {
		c.Check(i.Destroy(context.Background()), check.IsNil)
	}
}
