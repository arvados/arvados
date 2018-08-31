// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	storageacct "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/jmcvetta/randutil"
	"golang.org/x/crypto/ssh"
)

type AzureProviderConfig struct {
	SubscriptionID               string  `json:"subscription_id"`
	ClientID                     string  `json:"key"`
	ClientSecret                 string  `json:"secret"`
	TenantID                     string  `json:"tenant_id"`
	CloudEnv                     string  `json:"cloud_environment"`
	ResourceGroup                string  `json:"resource_group"`
	Location                     string  `json:"region"`
	Network                      string  `json:"network"`
	Subnet                       string  `json:"subnet"`
	StorageAccount               string  `json:"storage_account"`
	BlobContainer                string  `json:"blob_container"`
	Image                        string  `json:"image"`
	DeleteDanglingResourcesAfter float64 `json:"delete_dangling_resources_after"`
}

type VirtualMachinesClientWrapper interface {
	CreateOrUpdate(ctx context.Context,
		resourceGroupName string,
		VMName string,
		parameters compute.VirtualMachine) (result compute.VirtualMachine, err error)
	Delete(ctx context.Context, resourceGroupName string, VMName string) (result *http.Response, err error)
	ListComplete(ctx context.Context, resourceGroupName string) (result compute.VirtualMachineListResultIterator, err error)
}

type VirtualMachinesClientImpl struct {
	inner compute.VirtualMachinesClient
}

func (cl *VirtualMachinesClientImpl) CreateOrUpdate(ctx context.Context,
	resourceGroupName string,
	VMName string,
	parameters compute.VirtualMachine) (result compute.VirtualMachine, err error) {

	future, err := cl.inner.CreateOrUpdate(ctx, resourceGroupName, VMName, parameters)
	if err != nil {
		return compute.VirtualMachine{}, WrapAzureError(err)
	}
	future.WaitForCompletionRef(ctx, cl.inner.Client)
	r, err := future.Result(cl.inner)
	return r, WrapAzureError(err)
}

func (cl *VirtualMachinesClientImpl) Delete(ctx context.Context, resourceGroupName string, VMName string) (result *http.Response, err error) {
	future, err := cl.inner.Delete(ctx, resourceGroupName, VMName)
	if err != nil {
		return nil, WrapAzureError(err)
	}
	err = future.WaitForCompletionRef(ctx, cl.inner.Client)
	return future.Response(), WrapAzureError(err)
}

func (cl *VirtualMachinesClientImpl) ListComplete(ctx context.Context, resourceGroupName string) (result compute.VirtualMachineListResultIterator, err error) {
	r, err := cl.inner.ListComplete(ctx, resourceGroupName)
	return r, WrapAzureError(err)
}

type InterfacesClientWrapper interface {
	CreateOrUpdate(ctx context.Context,
		resourceGroupName string,
		networkInterfaceName string,
		parameters network.Interface) (result network.Interface, err error)
	Delete(ctx context.Context, resourceGroupName string, networkInterfaceName string) (result *http.Response, err error)
	ListComplete(ctx context.Context, resourceGroupName string) (result network.InterfaceListResultIterator, err error)
}

type InterfacesClientImpl struct {
	inner network.InterfacesClient
}

func (cl *InterfacesClientImpl) Delete(ctx context.Context, resourceGroupName string, VMName string) (result *http.Response, err error) {
	future, err := cl.inner.Delete(ctx, resourceGroupName, VMName)
	if err != nil {
		return nil, WrapAzureError(err)
	}
	err = future.WaitForCompletionRef(ctx, cl.inner.Client)
	return future.Response(), WrapAzureError(err)
}

func (cl *InterfacesClientImpl) CreateOrUpdate(ctx context.Context,
	resourceGroupName string,
	networkInterfaceName string,
	parameters network.Interface) (result network.Interface, err error) {

	future, err := cl.inner.CreateOrUpdate(ctx, resourceGroupName, networkInterfaceName, parameters)
	if err != nil {
		return network.Interface{}, WrapAzureError(err)
	}
	future.WaitForCompletionRef(ctx, cl.inner.Client)
	r, err := future.Result(cl.inner)
	return r, WrapAzureError(err)
}

func (cl *InterfacesClientImpl) ListComplete(ctx context.Context, resourceGroupName string) (result network.InterfaceListResultIterator, err error) {
	r, err := cl.inner.ListComplete(ctx, resourceGroupName)
	return r, WrapAzureError(err)
}

var quotaRe = regexp.MustCompile(`(?i:exceed|quota|limit)`)

type AzureRateLimitError struct {
	azure.RequestError
	earliestRetry time.Time
}

func (ar *AzureRateLimitError) EarliestRetry() time.Time {
	return ar.earliestRetry
}

type AzureQuotaError struct {
	azure.RequestError
}

func (ar *AzureQuotaError) IsQuotaError() bool {
	return true
}

func WrapAzureError(err error) error {
	de, ok := err.(autorest.DetailedError)
	if !ok {
		return err
	}
	rq, ok := de.Original.(*azure.RequestError)
	if !ok {
		return err
	}
	if rq.Response == nil {
		return err
	}
	if rq.Response.StatusCode == 429 || len(rq.Response.Header["Retry-After"]) >= 1 {
		// API throttling
		ra := rq.Response.Header["Retry-After"][0]
		earliestRetry, parseErr := http.ParseTime(ra)
		if parseErr != nil {
			// Could not parse as a timestamp, must be number of seconds
			dur, parseErr := strconv.ParseInt(ra, 10, 64)
			if parseErr != nil {
				earliestRetry = time.Now().Add(time.Duration(dur) * time.Second)
			}
		}
		if parseErr != nil {
			// Couldn't make sense of retry-after,
			// so set retry to 20 seconds
			earliestRetry = time.Now().Add(20 * time.Second)
		}
		return &AzureRateLimitError{*rq, earliestRetry}
	}
	if rq.ServiceError == nil {
		return err
	}
	if quotaRe.FindString(rq.ServiceError.Code) != "" || quotaRe.FindString(rq.ServiceError.Message) != "" {
		return &AzureQuotaError{*rq}
	}
	return err
}

type AzureProvider struct {
	azconfig          AzureProviderConfig
	vmClient          VirtualMachinesClientWrapper
	netClient         InterfacesClientWrapper
	storageAcctClient storageacct.AccountsClient
	azureEnv          azure.Environment
	interfaces        map[string]network.Interface
	dispatcherID      string
	namePrefix        string
}

func NewAzureProvider(azcfg AzureProviderConfig, dispatcherID string) (prv InstanceProvider, err error) {
	ap := AzureProvider{}
	err = ap.setup(azcfg, dispatcherID)
	if err != nil {
		return nil, err
	}
	return &ap, nil
}

func (az *AzureProvider) setup(azcfg AzureProviderConfig, dispatcherID string) (err error) {
	az.azconfig = azcfg
	vmClient := compute.NewVirtualMachinesClient(az.azconfig.SubscriptionID)
	netClient := network.NewInterfacesClient(az.azconfig.SubscriptionID)
	storageAcctClient := storageacct.NewAccountsClient(az.azconfig.SubscriptionID)

	az.azureEnv, err = azure.EnvironmentFromName(az.azconfig.CloudEnv)
	if err != nil {
		return err
	}

	authorizer, err := auth.ClientCredentialsConfig{
		ClientID:     az.azconfig.ClientID,
		ClientSecret: az.azconfig.ClientSecret,
		TenantID:     az.azconfig.TenantID,
		Resource:     az.azureEnv.ResourceManagerEndpoint,
		AADEndpoint:  az.azureEnv.ActiveDirectoryEndpoint,
	}.Authorizer()
	if err != nil {
		return err
	}

	vmClient.Authorizer = authorizer
	netClient.Authorizer = authorizer
	storageAcctClient.Authorizer = authorizer

	az.vmClient = &VirtualMachinesClientImpl{vmClient}
	az.netClient = &InterfacesClientImpl{netClient}
	az.storageAcctClient = storageAcctClient

	az.dispatcherID = dispatcherID
	az.namePrefix = fmt.Sprintf("compute-%s-", az.dispatcherID)

	return nil
}

func (az *AzureProvider) Create(ctx context.Context,
	instanceType arvados.InstanceType,
	imageId ImageID,
	newTags InstanceTags,
	publicKey ssh.PublicKey) (Instance, error) {

	if len(newTags["node-token"]) == 0 {
		return nil, fmt.Errorf("Must provide tag 'node-token'")
	}

	name, err := randutil.String(15, "abcdefghijklmnopqrstuvwxyz0123456789")
	if err != nil {
		return nil, err
	}

	name = az.namePrefix + name
	log.Printf("name is %v", name)

	timestamp := time.Now().Format(time.RFC3339Nano)

	newTags["instance-type"] = instanceType.Name

	tags := make(map[string]*string)
	tags["created-at"] = &timestamp
	for k, v := range newTags {
		tags["dispatch-"+k] = &v
	}

	nicParameters := network.Interface{
		Location: &az.azconfig.Location,
		Tags:     tags,
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &[]network.InterfaceIPConfiguration{
				network.InterfaceIPConfiguration{
					Name: to.StringPtr("ip1"),
					InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
						Subnet: &network.Subnet{
							ID: to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers"+
								"/Microsoft.Network/virtualnetworks/%s/subnets/%s",
								az.azconfig.SubscriptionID,
								az.azconfig.ResourceGroup,
								az.azconfig.Network,
								az.azconfig.Subnet)),
						},
						PrivateIPAllocationMethod: network.Dynamic,
					},
				},
			},
		},
	}
	nic, err := az.netClient.CreateOrUpdate(ctx, az.azconfig.ResourceGroup, name+"-nic", nicParameters)
	if err != nil {
		return nil, WrapAzureError(err)
	}

	log.Printf("Created NIC %v", *nic.ID)

	instance_vhd := fmt.Sprintf("https://%s.blob.%s/%s/%s-os.vhd",
		az.azconfig.StorageAccount,
		az.azureEnv.StorageEndpointSuffix,
		az.azconfig.BlobContainer,
		name)

	log.Printf("URI instance vhd %v", instance_vhd)

	customData := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(`#!/bin/sh
echo '%s-%s' > /home/crunch/node-token`, name, newTags["node-token"])))

	vmParameters := compute.VirtualMachine{
		Location: &az.azconfig.Location,
		Tags:     tags,
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.VirtualMachineSizeTypes(instanceType.ProviderType),
			},
			StorageProfile: &compute.StorageProfile{
				OsDisk: &compute.OSDisk{
					OsType:       compute.Linux,
					Name:         to.StringPtr(name + "-os"),
					CreateOption: compute.FromImage,
					Image: &compute.VirtualHardDisk{
						URI: to.StringPtr(string(imageId)),
					},
					Vhd: &compute.VirtualHardDisk{
						URI: &instance_vhd,
					},
				},
			},
			NetworkProfile: &compute.NetworkProfile{
				NetworkInterfaces: &[]compute.NetworkInterfaceReference{
					compute.NetworkInterfaceReference{
						ID: nic.ID,
						NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
							Primary: to.BoolPtr(true),
						},
					},
				},
			},
			OsProfile: &compute.OSProfile{
				ComputerName:  &name,
				AdminUsername: to.StringPtr("crunch"),
				LinuxConfiguration: &compute.LinuxConfiguration{
					DisablePasswordAuthentication: to.BoolPtr(true),
					SSH: &compute.SSHConfiguration{
						PublicKeys: &[]compute.SSHPublicKey{
							compute.SSHPublicKey{
								Path:    to.StringPtr("/home/crunch/.ssh/authorized_keys"),
								KeyData: to.StringPtr(string(ssh.MarshalAuthorizedKey(publicKey))),
							},
						},
					},
				},
				CustomData: &customData,
			},
		},
	}

	vm, err := az.vmClient.CreateOrUpdate(ctx, az.azconfig.ResourceGroup, name, vmParameters)
	if err != nil {
		return nil, WrapAzureError(err)
	}

	return &AzureInstance{
		provider: az,
		nic:      nic,
		vm:       vm,
	}, nil
}

func (az *AzureProvider) Instances(ctx context.Context) ([]Instance, error) {
	interfaces, err := az.ManageNics(ctx)
	if err != nil {
		return nil, err
	}

	result, err := az.vmClient.ListComplete(ctx, az.azconfig.ResourceGroup)
	if err != nil {
		return nil, WrapAzureError(err)
	}

	instances := make([]Instance, 0)

	for ; result.NotDone(); err = result.Next() {
		if err != nil {
			return nil, WrapAzureError(err)
		}
		if strings.HasPrefix(*result.Value().Name, az.namePrefix) {
			instances = append(instances, &AzureInstance{
				provider: az,
				vm:       result.Value(),
				nic:      interfaces[*(*result.Value().NetworkProfile.NetworkInterfaces)[0].ID]})
		}
	}
	return instances, nil
}

func (az *AzureProvider) ManageNics(ctx context.Context) (map[string]network.Interface, error) {
	result, err := az.netClient.ListComplete(ctx, az.azconfig.ResourceGroup)
	if err != nil {
		return nil, WrapAzureError(err)
	}

	interfaces := make(map[string]network.Interface)

	timestamp := time.Now()
	wg := sync.WaitGroup{}
	deletechannel := make(chan string, 20)
	defer func() {
		wg.Wait()
		close(deletechannel)
	}()
	for i := 0; i < 4; i += 1 {
		go func() {
			for {
				nicname, ok := <-deletechannel
				if !ok {
					return
				}
				_, delerr := az.netClient.Delete(context.Background(), az.azconfig.ResourceGroup, nicname)
				if delerr != nil {
					log.Printf("Error deleting %v: %v", nicname, delerr)
				} else {
					log.Printf("Deleted %v", nicname)
				}
				wg.Done()
			}
		}()
	}

	for ; result.NotDone(); err = result.Next() {
		if err != nil {
			log.Printf("Error listing nics: %v", err)
			return interfaces, nil
		}
		if strings.HasPrefix(*result.Value().Name, az.namePrefix) {
			if result.Value().VirtualMachine != nil {
				interfaces[*result.Value().ID] = result.Value()
			} else {
				if result.Value().Tags["created-at"] != nil {
					created_at, err := time.Parse(time.RFC3339Nano, *result.Value().Tags["created-at"])
					if err == nil {
						//log.Printf("found dangling NIC %v created %v seconds ago", *result.Value().Name, timestamp.Sub(created_at).Seconds())
						if timestamp.Sub(created_at).Seconds() > az.azconfig.DeleteDanglingResourcesAfter {
							log.Printf("Will delete %v because it is older than %v s", *result.Value().Name, az.azconfig.DeleteDanglingResourcesAfter)
							wg.Add(1)
							deletechannel <- *result.Value().Name
						}
					}
				}
			}
		}
	}
	return interfaces, nil
}

func (az *AzureProvider) ManageBlobs(ctx context.Context) {
	result, err := az.storageAcctClient.ListKeys(ctx, az.azconfig.ResourceGroup, az.azconfig.StorageAccount)
	if err != nil {
		log.Printf("Couldn't get account keys %v", err)
		return
	}

	key1 := *(*result.Keys)[0].Value
	client, err := storage.NewBasicClientOnSovereignCloud(az.azconfig.StorageAccount, key1, az.azureEnv)
	if err != nil {
		log.Printf("Couldn't make client %v", err)
		return
	}

	blobsvc := client.GetBlobService()
	blobcont := blobsvc.GetContainerReference(az.azconfig.BlobContainer)

	timestamp := time.Now()
	wg := sync.WaitGroup{}
	deletechannel := make(chan storage.Blob, 20)
	defer func() {
		wg.Wait()
		close(deletechannel)
	}()
	for i := 0; i < 4; i += 1 {
		go func() {
			for {
				blob, ok := <-deletechannel
				if !ok {
					return
				}
				err := blob.Delete(nil)
				if err != nil {
					log.Printf("error deleting %v: %v", blob.Name, err)
				} else {
					log.Printf("Deleted blob %v", blob.Name)
				}
				wg.Done()
			}
		}()
	}

	page := storage.ListBlobsParameters{Prefix: az.namePrefix}

	for {
		response, err := blobcont.ListBlobs(page)
		if err != nil {
			log.Printf("Error listing blobs %v", err)
			return
		}
		for _, b := range response.Blobs {
			age := timestamp.Sub(time.Time(b.Properties.LastModified))
			if b.Properties.BlobType == storage.BlobTypePage &&
				b.Properties.LeaseState == "available" &&
				b.Properties.LeaseStatus == "unlocked" &&
				age.Seconds() > az.azconfig.DeleteDanglingResourcesAfter {

				log.Printf("Blob %v is unlocked and not modified for %v seconds, will delete", b.Name, age.Seconds())
				wg.Add(1)
				deletechannel <- b
			}
		}
		if response.NextMarker != "" {
			page.Marker = response.NextMarker
		} else {
			break
		}
	}
}

func (az *AzureProvider) Stop() {
}

type AzureInstance struct {
	provider *AzureProvider
	nic      network.Interface
	vm       compute.VirtualMachine
}

func (ai *AzureInstance) ID() InstanceID {
	return InstanceID(*ai.vm.ID)
}

func (ai *AzureInstance) String() string {
	return *ai.vm.Name
}

func (ai *AzureInstance) SetTags(ctx context.Context, newTags InstanceTags) error {
	tags := make(map[string]*string)

	for k, v := range ai.vm.Tags {
		if !strings.HasPrefix(k, "dispatch-") {
			tags[k] = v
		}
	}
	for k, v := range newTags {
		tags["dispatch-"+k] = &v
	}

	vmParameters := compute.VirtualMachine{
		Location: &ai.provider.azconfig.Location,
		Tags:     tags,
	}
	vm, err := ai.provider.vmClient.CreateOrUpdate(ctx, ai.provider.azconfig.ResourceGroup, *ai.vm.Name, vmParameters)
	if err != nil {
		return WrapAzureError(err)
	}
	ai.vm = vm

	return nil
}

func (ai *AzureInstance) Tags(ctx context.Context) (InstanceTags, error) {
	tags := make(map[string]string)

	for k, v := range ai.vm.Tags {
		if strings.HasPrefix(k, "dispatch-") {
			tags[k[9:]] = *v
		}
	}

	return tags, nil
}

func (ai *AzureInstance) Destroy(ctx context.Context) error {
	_, err := ai.provider.vmClient.Delete(ctx, ai.provider.azconfig.ResourceGroup, *ai.vm.Name)
	return WrapAzureError(err)
}

func (ai *AzureInstance) Address() string {
	return *(*ai.nic.IPConfigurations)[0].PrivateIPAddress
}
