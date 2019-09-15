/*
Copyright 2019 Alexander Eldeib.
*/

package vmspec

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

type Spec struct {
	internal *compute.VirtualMachine
}

func New() *Spec {
	return &Spec{
		internal: &compute.VirtualMachine{
			VirtualMachineProperties: &compute.VirtualMachineProperties{
				HardwareProfile: &compute.HardwareProfile{},
				StorageProfile: &compute.StorageProfile{
					ImageReference: &compute.ImageReference{
						Publisher: to.StringPtr("Canonical"),
						Offer:     to.StringPtr("UbuntuServer"),
						Sku:       to.StringPtr("18.04-LTS"),
						Version:   to.StringPtr("latest"),
					},
					OsDisk: &compute.OSDisk{
						CreateOption: compute.DiskCreateOptionTypesFromImage,
						DiskSizeGB:   to.Int32Ptr(100),
						// ManagedDisk: &compute.ManagedDiskParameters{
						// 	StorageAccountType: compute.StorageAccountTypes(vmSpec.OSDisk.ManagedDisk.StorageAccountType),
						// },
					},
				},
				NetworkProfile: &compute.NetworkProfile{
					NetworkInterfaces: &[]compute.NetworkInterfaceReference{},
				},
				OsProfile: &compute.OSProfile{
					AdminUsername: to.StringPtr("azureuser"),
					AdminPassword: to.StringPtr(GenerateRandomString(32)),
					LinuxConfiguration: &compute.LinuxConfiguration{
						DisablePasswordAuthentication: to.BoolPtr(false),
						SSH: &compute.SSHConfiguration{
							PublicKeys: &[]compute.SSHPublicKey{},
						},
					},
				},
			},
			Zones: &[]string{},
		},
	}
}

func NewFromExisting(remote *compute.VirtualMachine) *Spec {
	return &Spec{
		internal: remote,
	}
}

func (s *Spec) Build() compute.VirtualMachine {
	return *s.internal
}

func (s *Spec) Name(name string) {
	s.internal.Name = &name
}

func (s *Spec) Location(location string) {
	s.internal.Location = &location
}

// Zones is an array containing the single zone this VM is in.
func (s *Spec) Zone(zone string) {
	s.internal.Zones = &[]string{zone}
}

func (s *Spec) Hostname(hostname string) {
	s.initialize(
		[]func() bool{
			s.checkProperties,
			s.checkOSProfile,
		},
		[]func(){
			s.initProperties,
			s.initOSProfile,
		},
	)
	s.internal.VirtualMachineProperties.OsProfile.ComputerName = &hostname
}

func (s *Spec) SKU(sku string) {
	s.initialize(
		[]func() bool{
			s.checkProperties,
			s.checkHardwareProfile,
		},
		[]func(){
			s.initProperties,
			s.initHardwareProfile,
		},
	)
	s.internal.VirtualMachineProperties.HardwareProfile.VMSize = compute.VirtualMachineSizeTypes(sku)
}

func (s *Spec) NICs(primaryNic string, secondaryNics *[]string) {
	s.initialize(
		[]func() bool{
			s.checkProperties,
			s.checkNetworkProfile,
		},
		[]func(){
			s.initProperties,
			s.initNetworkProfile,
		},
	)

	s.internal.VirtualMachineProperties.NetworkProfile.NetworkInterfaces = &[]compute.NetworkInterfaceReference{
		{
			ID: &primaryNic,
			NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
				Primary: to.BoolPtr(true),
			},
		},
	}

	if secondaryNics != nil {
		for _, nic := range *secondaryNics {
			*s.internal.VirtualMachineProperties.NetworkProfile.NetworkInterfaces = append(
				*s.internal.VirtualMachineProperties.NetworkProfile.NetworkInterfaces,
				compute.NetworkInterfaceReference{
					ID: &nic,
					NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
						Primary: to.BoolPtr(false),
					},
				},
			)
		}
	}
}

func (s *Spec) NeedsUpdate(local *azurev1alpha1.VM) bool {
	return any([]func() bool{
		func() bool { return Name(s) == nil || local.Spec.Name != *Name(s) },
		func() bool { return SKU(s) == nil || compute.VirtualMachineSizeTypes(local.Spec.SKU) != *SKU(s) },
		// Immutable
		// func() bool { return SubscriptionID(s) == nil || local.Spec.SubscriptionID != *SubscriptionID(s) },
		// func() bool { return ResourceGroup(s) == nil || local.Spec.ResourceGroup != *ResourceGroup(s) },
		// func() bool { return CustomData(s) == nil || local.Spec.CustomData != *CustomData(s) },
		// func() bool { return Location(s) == nil || local.Spec.Location != *Location(s) },
		// func() bool { return local.Spec.Zone != nil && Zone(s) != nil && *local.Spec.Zone != *Zone(s) },
	})
}

func (s *Spec) initialize(detectors []func() bool, remediators []func()) {
	for idx, f := range detectors {
		if f() {
			remediators[idx]()
		}
	}
}

func Name(s *Spec) *string {
	return s.internal.Name
}

func Location(s *Spec) *string {
	return s.internal.Location
}

// VM can only have one zone
func Zone(s *Spec) *string {
	if s.internal.Zones != nil && len(*s.internal.Zones) > 0 {
		return &((*s.internal.Zones)[0]) // TODO(ace): simplify
	}
	return nil
}

func ID(s *Spec) *string {
	return s.internal.ID
}

func State(s *Spec) *string {
	if s.internal.VirtualMachineProperties == nil {
		return nil
	}
	return s.internal.VirtualMachineProperties.ProvisioningState
}

func SKU(s *Spec) *compute.VirtualMachineSizeTypes {
	if s.internal.VirtualMachineProperties == nil || s.internal.VirtualMachineProperties.HardwareProfile == nil {
		return nil
	}
	return &s.internal.VirtualMachineProperties.HardwareProfile.VMSize
}

func (s *Spec) checkProperties() bool {
	return s.internal.VirtualMachineProperties == nil
}
func (s *Spec) initProperties() {
	s.internal.VirtualMachineProperties = &compute.VirtualMachineProperties{}
}

func (s *Spec) checkHardwareProfile() bool {
	return s.internal.VirtualMachineProperties.HardwareProfile == nil
}
func (s *Spec) initHardwareProfile() {
	s.internal.VirtualMachineProperties.HardwareProfile = &compute.HardwareProfile{}
}

func (s *Spec) checkNetworkProfile() bool {
	return s.internal.VirtualMachineProperties.NetworkProfile == nil
}
func (s *Spec) initNetworkProfile() {
	s.internal.VirtualMachineProperties.NetworkProfile = &compute.NetworkProfile{}
}

func (s *Spec) checkOSProfile() bool {
	return s.internal.VirtualMachineProperties.OsProfile == nil
}
func (s *Spec) initOSProfile() {
	s.internal.VirtualMachineProperties.OsProfile = &compute.OSProfile{}
}

func any(funcs []func() bool) bool {
	for _, f := range funcs {
		if f() {
			return true
		}
	}
	return false
}

// https://github.com/kubernetes-sigs/cluster-api-provider-azure/blob/60b7c6058550ae694935fb03103460a2efa4e332/pkg/cloud/azure/services/virtualmachines/virtualmachines.go#L215
func GenerateRandomString(n int) string {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		fmt.Printf("error in generate random: %+#v", err.Error())
	}
	return base64.URLEncoding.EncodeToString(b) //, err
}
