// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	storageacct "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/jmcvetta/randutil"
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
	AuthorizedKey                string  `json:"authorized_key"`
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
		return compute.VirtualMachine{}, err
	}
	future.WaitForCompletionRef(ctx, cl.inner.Client)
	return future.Result(cl.inner)
}

func (cl *VirtualMachinesClientImpl) Delete(ctx context.Context, resourceGroupName string, VMName string) (result *http.Response, err error) {
	future, err := cl.inner.Delete(ctx, resourceGroupName, VMName)
	if err != nil {
		return nil, err
	}
	err = future.WaitForCompletionRef(ctx, cl.inner.Client)
	return future.Response(), err
}

func (cl *VirtualMachinesClientImpl) ListComplete(ctx context.Context, resourceGroupName string) (result compute.VirtualMachineListResultIterator, err error) {
	return cl.inner.ListComplete(ctx, resourceGroupName)
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
		return nil, err
	}
	err = future.WaitForCompletionRef(ctx, cl.inner.Client)
	return future.Response(), err
}

func (cl *InterfacesClientImpl) CreateOrUpdate(ctx context.Context,
	resourceGroupName string,
	networkInterfaceName string,
	parameters network.Interface) (result network.Interface, err error) {

	future, err := cl.inner.CreateOrUpdate(ctx, resourceGroupName, networkInterfaceName, parameters)
	if err != nil {
		return network.Interface{}, err
	}
	future.WaitForCompletionRef(ctx, cl.inner.Client)
	return future.Result(cl.inner)
}

func (cl *InterfacesClientImpl) ListComplete(ctx context.Context, resourceGroupName string) (result network.InterfaceListResultIterator, err error) {
	return cl.inner.ListComplete(ctx, resourceGroupName)
}

type AzureProvider struct {
	azconfig          AzureProviderConfig
	arvconfig         arvados.Cluster
	vmClient          VirtualMachinesClientWrapper
	netClient         InterfacesClientWrapper
	storageAcctClient storageacct.AccountsClient
	azureEnv          azure.Environment
	interfaces        map[string]network.Interface
}

func NewAzureProvider(azcfg AzureProviderConfig, arvcfg arvados.Cluster) (prv Provider, err error) {
	ap := AzureProvider{}
	err = ap.setup(azcfg, arvcfg)
	if err != nil {
		return nil, err
	}
	return &ap, nil
}

func (az *AzureProvider) setup(azcfg AzureProviderConfig, arvcfg arvados.Cluster) (err error) {
	az.azconfig = azcfg
	az.arvconfig = arvcfg
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

	return nil
}

func (az *AzureProvider) Create(ctx context.Context,
	instanceType arvados.InstanceType,
	imageId ImageID,
	instanceTag []InstanceTag) (Instance, error) {

	name, err := randutil.String(15, "abcdefghijklmnopqrstuvwxyz0123456789")
	if err != nil {
		return nil, err
	}

	name = "compute-" + name
	log.Printf("name is %v", name)

	timestamp := time.Now().Format(time.RFC3339Nano)

	nicParameters := network.Interface{
		Location: &az.azconfig.Location,
		Tags: map[string]*string{
			"arvados-class":   to.StringPtr("crunch-dynamic-compute"),
			"arvados-cluster": &az.arvconfig.ClusterID,
			"created-at":      &timestamp,
		},
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
		return nil, err
	}

	log.Printf("Created NIC %v", *nic.ID)

	instance_vhd := fmt.Sprintf("https://%s.blob.%s/%s/%s-os.vhd",
		az.azconfig.StorageAccount,
		az.azureEnv.StorageEndpointSuffix,
		az.azconfig.BlobContainer,
		name)

	log.Printf("URI instance vhd %v", instance_vhd)

	vmParameters := compute.VirtualMachine{
		Location: &az.azconfig.Location,
		Tags: map[string]*string{
			"arvados-class":   to.StringPtr("crunch-dynamic-compute"),
			"arvados-cluster": &az.arvconfig.ClusterID,
			"created-at":      &timestamp,
		},
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.VirtualMachineSizeTypes(instanceType.ProviderType),
			},
			StorageProfile: &compute.StorageProfile{
				OsDisk: &compute.OSDisk{
					OsType:       compute.Linux,
					Name:         to.StringPtr(fmt.Sprintf("%v-os", name)),
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
				AdminUsername: to.StringPtr("arvados"),
				LinuxConfiguration: &compute.LinuxConfiguration{
					DisablePasswordAuthentication: to.BoolPtr(true),
					SSH: &compute.SSHConfiguration{
						PublicKeys: &[]compute.SSHPublicKey{
							compute.SSHPublicKey{
								Path:    to.StringPtr("/home/arvados/.ssh/authorized_keys"),
								KeyData: to.StringPtr(az.azconfig.AuthorizedKey),
							},
						},
					},
				},
				//CustomData: to.StringPtr(""),
			},
		},
	}

	vm, err := az.vmClient.CreateOrUpdate(ctx, az.azconfig.ResourceGroup, name, vmParameters)
	if err != nil {
		return nil, err
	}

	return &AzureInstance{
		instanceType: instanceType,
		provider:     az,
		nic:          nic,
		vm:           vm,
	}, nil
}

func (az *AzureProvider) Instances(ctx context.Context) ([]Instance, error) {
	interfaces, err := az.ManageNics(ctx)
	if err != nil {
		return nil, err
	}

	result, err := az.vmClient.ListComplete(ctx, az.azconfig.ResourceGroup)
	if err != nil {
		return nil, err
	}

	instances := make([]Instance, 0)

	for ; result.NotDone(); err = result.Next() {
		if err != nil {
			return nil, err
		}
		if result.Value().Tags["arvados-class"] != nil &&
			(*result.Value().Tags["arvados-class"]) == "crunch-dynamic-compute" {
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
		return nil, err
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
		if result.Value().Tags["arvados-class"] != nil &&
			(*result.Value().Tags["arvados-class"]) == "crunch-dynamic-compute" {

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

	page := storage.ListBlobsParameters{Prefix: "compute-"}

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

type AzureInstance struct {
	instanceType arvados.InstanceType
	provider     *AzureProvider
	nic          network.Interface
	vm           compute.VirtualMachine
}

func (ai *AzureInstance) String() string {
	return *ai.vm.Name
}

func (ai *AzureInstance) ProviderType() string {
	return string(ai.vm.VirtualMachineProperties.HardwareProfile.VMSize)
}

func (ai *AzureInstance) InstanceType() arvados.InstanceType {
	return ai.instanceType
}

func (ai *AzureInstance) SetTags([]InstanceTag) error {
	return nil
}

func (ai *AzureInstance) GetTags() ([]InstanceTag, error) {
	return nil, nil
}

func (ai *AzureInstance) Destroy(ctx context.Context) error {
	_, err := ai.provider.vmClient.Delete(ctx, ai.provider.azconfig.ResourceGroup, *ai.vm.Name)
	// check response code?
	return err
}

func (ai *AzureInstance) Address() string {
	return *(*ai.nic.IPConfigurations)[0].PrivateIPAddress
}
