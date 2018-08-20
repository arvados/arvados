package dispatchcloud

import (
	"context"
	"fmt"
	"net/http"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
)

type AzureProviderConfig struct {
	SubscriptionID string
	ClientID       string
	ClientSecret   string
	TenantID       string
	CloudEnv       string
	ResourceGroup  string
	Location       string
	Subnet         string
	StorageAccount string
	BlobContainer  string
}

type VirtualMachinesClientWrapper interface {
	CreateOrUpdate(ctx context.Context,
		resourceGroupName string,
		VMName string,
		parameters compute.VirtualMachine) (result compute.VirtualMachine, err error)
	Delete(ctx context.Context, resourceGroupName string, VMName string) (result *http.Response, err error)
	List(ctx context.Context, resourceGroupName string) (result compute.VirtualMachineListResultPage, err error)
}

type VirtualMachinesClientImpl struct {
	inner compute.VirtualMachinesClient
}

func (cl *VirtualMachinesClientImpl) CreateOrUpdate(ctx context.Context,
	resourceGroupName string,
	VMName string,
	parameters VirtualMachine) (result compute.VirtualMachine, err error) {

	future, err := cl.inner.CreateOrUpdate(ctx, resourceGroupName, VMName, parameters)
	if err != nil {
		return nil, err
	}
	future.WaitForCompletionRef(ctx, cl.inner.Client)
	return future.Result(cl.inner)
}

func (cl *VirtualMachinesClientImpl) Delete(ctx context.Context, resourceGroupName string, VMName string) (result *http.Response, err error) {
	future, err := cl.inner.Delete(ctx, resourceGroupName, VMName)
	if err != nil {
		return nil, err
	}
	future.WaitForCompletionRef(ctx, cl.inner.Client)
	return future.GetResult()
}

func (cl *VirtualMachinesClientImpl) List(ctx context.Context, resourceGroupName string) (result compute.VirtualMachineListResultPage, err error) {
	return cl.inner.List(ctx, resourceGroupName)
}

type InterfacesClientWrapper interface {
	CreateOrUpdate(ctx context.Context,
		resourceGroupName string,
		networkInterfaceName string,
		parameters network.Interface) (result network.Interface, err error)
	Delete(ctx context.Context, resourceGroupName string, networkInterfaceName string) (result *http.Response, err error)
	List(ctx context.Context, resourceGroupName string) (result network.InterfaceListResultPage, err error)
}

type InterfacesClientImpl struct {
	inner network.InterfacesClient
}

func (cl *InterfacesClientImpl) Delete(ctx context.Context, resourceGroupName string, VMName string) (result *http.Response, err error) {
	future, err := cl.inner.Delete(ctx, resourceGroupName, VMName)
	if err != nil {
		return nil, err
	}
	future.WaitForCompletionRef(ctx, cl.inner.Client)
	return future.GetResult()
}

func (cl *InterfacesClientImpl) CreateOrUpdate(ctx context.Context,
	resourceGroupName string,
	networkInterfaceName string,
	parameters network.Interface) (result compute.VirtualMachine, err error) {

	future, err := cl.inner.CreateOrUpdate(ctx, resourceGroupName, networkInterfaceName, parameters)
	if err != nil {
		return nil, err
	}
	future.WaitForCompletionRef(ctx, cl.inner.Client)
	return future.Result(cl.inner)
}

func (cl *InterfacesClientImpl) List(ctx context.Context, resourceGroupName string) (result compute.InterfaceListResultPage, err error) {
	return cl.inner.List(ctx, resourceGroupName)
}

type AzureProvider struct {
	config    AzureProviderConfig
	vmClient  VirtualMachinesClientWrapper
	netClient InterfacesClientWrapper
	azureEnv  auth.Environment
}

func (az *AzureProvider) Init(cfg AzureProviderConfig) error {
	az.config = cfg
	vmClient := compute.NewVirtualMachinesClient(az.config.SubscriptionId)
	netClient := network.NewInterfacesClient(az.config.SubscriptionId)

	az.azureEnv, err = azure.EnvironmentFromName(az.config.CloudEnv)
	if err != nil {
		return err
	}

	authorizer, err := auth.ClientCredentialsConfig{
		ClientID:     az.config.ClientID,
		ClientSecret: az.config.ClientSecret,
		TenantID:     az.config.TenantID,
		Resource:     env.ResourceManagerEndpoint,
		AADEndpoint:  env.ActiveDirectoryEndpoint,
	}.Authorizer()
	if err != nil {
		return err
	}

	vmClient.Authorizer = authorizer
	netClient.Authorizer = authorizer

	az.vmClient = VirtualMachinesClientImpl{vmClient}
	az.netClient = InterfacesClientImpl{netClient}

	return nil
}

func (az *AzureProvider) Create(ctx context.Context,
	instanceType arvados.InstanceType,
	imageId ImageID,
	instanceTag []InstanceTag) (Instance, error) {

	name := "randomname"

	nicParameters := network.Interface{
		Location: az.config.Location,
		Tags: []map[string]string{
			"arvados-class":   "dynamic-compute",
			"arvados-cluster": "",
		},
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &[]network.InterfaceIPConfiguration{
				network.InterfaceIPConfiguration{},
				Name: "ip1",
				InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
					Subnet: &network.Subnet{
						ID: az.config.Subnet,
					},
					PrivateIPAllocationMethod: network.Dynamic,
				},
			},
		},
	}
	nic, err := az.netClient.CreateOrUpdate(ctx, az.config.ResourceGroup, name+"-nic", nicParameters)
	if err != nil {
		return nil, err
	}

	instance_vhd = fmt.Sprintf("https://%s.blob.%s/%s/%s-os.vhd",
		az.config.StorageAccount,
		az.azureEnv.StorageEndpointSuffix,
		az.config.BlobContainer,
		name)

	vmParameters := compute.VirtualMachine{
		Location: az.config.Location,
		Tags: []map[string]string{
			"arvados-class":   "dynamic-compute",
			"arvados-cluster": "",
		},
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			HardwareProfile: &compute.HardwareProfile{
				VMSize: instanceType.ProviderType,
			},
			StorageProfile: &compute.StorageProfile{
				OsDisk: &compute.OSDisk{
					OsType:       compute.Linux,
					Name:         "",
					CreateOption: compute.FromImage,
					Image: &compute.VirtualHardDisk{
						URI: imageId,
					},
					Vhd: &compute.VirtualHardDisk{
						URI: vhd,
					},
				},
			},
			NetworkProfile: &compute.NetworkProfile{
				NetworkInterfaces: &[]compute.NetworkInterfaceReference{
					compute.NetworkInterfaceReference{
						ID: "",
						NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
							Primary: to.BoolPtr(true),
						},
					},
				},
			},
			OsProfile: &compute.OSProfile{
				LinuxConfiguration: &compute.LinuxConfiguration{
					DisablePasswordAuthentication: true,
					SSH: &compute.SSHConfiguration{
						PublicKeys: &[]compute.SSHPublicKey{
							compute.SSHPublicKey{
								Path:    "",
								KeyData: "",
							},
						},
					},
				},
			},
		},
	}

	vm, err := az.vmClient.CreateOrUpdate(ctx, az.config.ResourceGroup, name+"-compute", vmParameters)
	if err != nil {
		return nil, err
	}

	return &AzureInstance{
		instanceType: instanceType,
		provider:     az,
		nic:          nic,
		vm:           vm,
	}
}

func (az *AzureProvider) Instances(ctx context.Context) ([]Instance, error) {
	result, err := az.vmClient.List(ctx, az.config.ResourceGroup)
	if err != nil {
		return nil, err
	}
	instances := make([]Instance)
	for {
		if result.NotDone() {
			result.Next()
		} else {
			return instances, nil
		}
		result.Values
	}
}

type AzureInstance struct {
	instanceType arvados.InstanceType
	provider     *AzureProvider
	nic          network.Interface
	vm           compute.VirtualMachine
}

func (ai *AzureInstance) String() string {
	return ai.vm.ID
}

func (ai *AzureInstance) ProviderType() string {
	return ai.vm.VirtualMachineProperties.HardwareProfile.VMSize
}

func (ai *AzureInstance) InstanceType() arvados.InstanceType {
	return ai.instanceType
}

func (ai *AzureInstance) SetTags([]InstanceTag) error {
	return nil
}

func (ai *AzureInstance) Destroy(ctx context.Context) error {
	response, err := ai.provider.vm.Delete(ctx, ai.provider.config.ResourceGroup, ai.vm.Name)
	// check response code
	return err
}

func (ai *AzureInstance) Address() string {
	return ai.nic.IPConfigurations[0].PrivateIPAddress
}
