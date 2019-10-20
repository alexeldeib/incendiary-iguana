/*
Copyright 2019 Alexander Eldeib.
*/

package vms

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/go-autorest/autorest/to"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/clientutil"
)

type Spec struct {
	internal *compute.VirtualMachine
}

func NewSpec() *Spec {
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
					AdminPassword: to.StringPtr(clientutil.GenerateRandomString(32)),
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

func NewSpecWithRemote(remote *compute.VirtualMachine) *Spec {
	return &Spec{
		internal: remote,
	}
}

func (s *Spec) Set(opts ...func(*Spec)) {
	for _, opt := range opts {
		opt(s)
	}
}

func (s *Spec) Build() compute.VirtualMachine {
	return *s.internal
}

func Name(name string) func(s *Spec) {
	return func(s *Spec) {
		s.internal.Name = &name
	}
}

func Location(location string) func(s *Spec) {
	return func(s *Spec) {
		s.internal.Location = &location
	}
}

// Zones is an array containing the single zone this VM is in.
func Zone(zone string) func(s *Spec) {
	return func(s *Spec) {
		s.internal.Zones = &[]string{zone}
	}
}

func Hostname(hostname string) func(s *Spec) {
	return func(s *Spec) {
		clientutil.Initialize(
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
}

func SKU(sku string) func(s *Spec) {
	return func(s *Spec) {
		clientutil.Initialize(
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
}

func NICs(primaryNic string, secondaryNics *[]string) func(s *Spec) {
	return func(s *Spec) {
		clientutil.Initialize(
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
}

func (s *Spec) NeedsUpdate(local *azurev1alpha1.VM) bool {
	return clientutil.Any([]func() bool{
		func() bool { return s.Name() == nil || local.Spec.Name != *s.Name() },
		func() bool { return s.SKU() == nil || compute.VirtualMachineSizeTypes(local.Spec.SKU) != *s.SKU() },
		// Immutable
		// func() bool { return s.SubscriptionID() == nil || local.Spec.SubscriptionID != *s.SubscriptionID() },
		// func() bool { return s.ResourceGroup() == nil || local.Spec.ResourceGroup != *s.ResourceGroup() },
		// func() bool { return s.CustomData() == nil || local.Spec.CustomData != *s.CustomData() },
		// func() bool { return s.Location() == nil || local.Spec.Location != *s.Location() },
		// func() bool { return local.Spec.Zone != nil && s.Zone() != nil && *local.Spec.Zone != *s.Zone() },
	})
}

func (s *Spec) Name() *string {
	return s.internal.Name
}

func (s *Spec) Location() *string {
	return s.internal.Location
}

func (s *Spec) Zone() *string {
	// VM can only have one zone
	if s.internal.Zones != nil && len(*s.internal.Zones) > 0 {
		return &((*s.internal.Zones)[0]) // TODO(ace): simplify
	}
	return nil
}

func (s *Spec) ID() *string {
	return s.internal.ID
}

func (s *Spec) State() *string {
	if s.internal.VirtualMachineProperties == nil {
		return nil
	}
	return s.internal.VirtualMachineProperties.ProvisioningState
}

func (s *Spec) SKU() *compute.VirtualMachineSizeTypes {
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
