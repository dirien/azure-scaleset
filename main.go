package main

import (
	"github.com/pulumi/pulumi-azure-native/sdk/go/azure/compute"
	"github.com/pulumi/pulumi-azure-native/sdk/go/azure/managedidentity"
	"github.com/pulumi/pulumi-azure-native/sdk/go/azure/network"
	"github.com/pulumi/pulumi-azure-native/sdk/go/azure/resources"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create an Azure Resource Group
		resourceGroup, err := resources.NewResourceGroup(ctx, "test_rg", nil)
		if err != nil {
			return err
		}

		virtualNetwork, err := network.NewVirtualNetwork(ctx, "test_vnet", &network.VirtualNetworkArgs{
			ResourceGroupName: resourceGroup.Name,
			AddressSpace: &network.AddressSpaceArgs{
				AddressPrefixes: pulumi.StringArray{
					pulumi.String("10.1.0.0/16"),
				},
			},
		})
		if err != nil {
			return err
		}

		subnet, err := network.NewSubnet(ctx, "test_subnet", &network.SubnetArgs{
			ResourceGroupName:  resourceGroup.Name,
			VirtualNetworkName: virtualNetwork.Name,
			AddressPrefix:      pulumi.String("10.1.0.0/24"),
		})
		if err != nil {
			return err
		}

		networkInterface, err := network.NewNetworkInterface(ctx, "test_nic", &network.NetworkInterfaceArgs{
			Location:          resourceGroup.Location,
			ResourceGroupName: resourceGroup.Name,
			IpConfigurations: network.NetworkInterfaceIPConfigurationArray{
				&network.NetworkInterfaceIPConfigurationArgs{
					Name:                      pulumi.String("test_ip_config"),
					PrivateIPAllocationMethod: pulumi.String("Dynamic"),
					Subnet: &network.SubnetTypeArgs{
						Id: subnet.ID(),
					},
				},
			},
		})
		if err != nil {
			return err
		}

		identity, err := managedidentity.NewUserAssignedIdentity(ctx, "test_identity", &managedidentity.UserAssignedIdentityArgs{
			ResourceGroupName: resourceGroup.Name,
			ResourceName:      pulumi.String("test_identity"),
		})
		if err != nil {
			return err
		}

		identityMap := identity.ID().ToIDOutput().ToStringOutput().ApplyT(func(v string) map[string]interface{} {
			m := make(map[string]interface{})
			m[v] = pulumi.ToStringMap(map[string]string{})
			return m
		}).(pulumi.MapOutput)

		compute.NewVirtualMachineScaleSet(ctx, "test_vmss", &compute.VirtualMachineScaleSetArgs{
			ResourceGroupName: resourceGroup.Name,
			Identity: &compute.VirtualMachineScaleSetIdentityArgs{
				Type:                   compute.ResourceIdentityTypeUserAssigned,
				UserAssignedIdentities: identityMap,
			},
			Sku: &compute.SkuArgs{
				Name:     pulumi.String("Standard_D2_v4"),
				Tier:     pulumi.String("Standard"),
				Capacity: pulumi.Float64(1.0),
			},
			UpgradePolicy: &compute.UpgradePolicyArgs{
				Mode: compute.UpgradeModeAutomatic,
			},
			VmScaleSetName: pulumi.String("test_vmss"),
			VirtualMachineProfile: &compute.VirtualMachineScaleSetVMProfileArgs{
				StorageProfile: &compute.VirtualMachineScaleSetStorageProfileArgs{
					OsDisk: &compute.VirtualMachineScaleSetOSDiskArgs{
						CreateOption: pulumi.String("FromImage"),
						ManagedDisk: &compute.VirtualMachineScaleSetManagedDiskParametersArgs{
							StorageAccountType: pulumi.String("Standard_LRS"),
						},
					},
					ImageReference: &compute.ImageReferenceArgs{
						Publisher: pulumi.String("Canonical"),
						Offer:     pulumi.String("0001-com-ubuntu-minimal-jammy-daily"),
						Sku:       pulumi.String("minimal-22_04-daily-lts-gen2"),
						Version:   pulumi.String("latest"),
					},
				},
				NetworkProfile: &compute.VirtualMachineScaleSetNetworkProfileArgs{
					NetworkInterfaceConfigurations: compute.VirtualMachineScaleSetNetworkConfigurationArray{
						&compute.VirtualMachineScaleSetNetworkConfigurationArgs{
							Id:                          networkInterface.ID(),
							Primary:                     pulumi.BoolPtr(true),
							EnableAcceleratedNetworking: pulumi.BoolPtr(true),
							IpConfigurations: compute.VirtualMachineScaleSetIPConfigurationArray{
								&compute.VirtualMachineScaleSetIPConfigurationArgs{
									Name: pulumi.String("test_ip_config"),
									Subnet: compute.ApiEntityReferenceArgs{
										Id: subnet.ID(),
									},
								},
							},
							Name: pulumi.String("test_nic"),
						},
					},
				},
				OsProfile: &compute.VirtualMachineScaleSetOSProfileArgs{
					AdminUsername:      pulumi.String("testadmin"),
					ComputerNamePrefix: pulumi.String("testvm"),
					AdminPassword:      pulumi.String("testPassword1234!"),
					LinuxConfiguration: &compute.LinuxConfigurationArgs{
						DisablePasswordAuthentication: pulumi.Bool(false),
					},
				},
			},
		})

		return nil
	})
}
