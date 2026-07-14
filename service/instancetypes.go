package service

import "fmt"

// CPU architectures used by allowed instance types.
const (
	ArchARM64  = "arm64"
	ArchX86_64 = "x86_64"
)

// DefaultInstanceType is the instance type used when no interactive picker is shown.
const DefaultInstanceType = "t4g.medium"

// InstanceType is an EC2 instance type with a human-readable vCPU/RAM/GPU label.
type InstanceType struct {
	ID    string
	Label string
	Arch  string
}

// AllInstanceTypes returns every allowed instance size for the create picker.
func AllInstanceTypes() []InstanceType {
	return []InstanceType{
		{ID: "t4g.nano", Label: "2 vCPU, 0.5 GiB RAM", Arch: ArchARM64},
		{ID: "t4g.micro", Label: "2 vCPU, 1 GiB RAM", Arch: ArchARM64},
		{ID: "t4g.small", Label: "2 vCPU, 2 GiB RAM", Arch: ArchARM64},
		{ID: "t4g.medium", Label: "2 vCPU, 4 GiB RAM", Arch: ArchARM64},
		{ID: "t4g.large", Label: "2 vCPU, 8 GiB RAM", Arch: ArchARM64},
		{ID: "t4g.xlarge", Label: "4 vCPU, 16 GiB RAM", Arch: ArchARM64},
		{ID: "t4g.2xlarge", Label: "8 vCPU, 32 GiB RAM", Arch: ArchARM64},
		{ID: "g4dn.xlarge", Label: "4 vCPU, 16 GiB RAM, 1x NVIDIA T4 (16 GB)", Arch: ArchX86_64},
		{ID: "g6.xlarge", Label: "4 vCPU, 16 GiB RAM, 1x NVIDIA L4 (24 GB)", Arch: ArchX86_64},
		{ID: "g5.xlarge", Label: "4 vCPU, 16 GiB RAM, 1x NVIDIA A10G (24 GB)", Arch: ArchX86_64},
		{ID: "g5.2xlarge", Label: "8 vCPU, 32 GiB RAM, 1x NVIDIA A10G (24 GB)", Arch: ArchX86_64},
	}
}

// DefaultInstanceTypeIndex returns the menu index for DefaultInstanceType.
func DefaultInstanceTypeIndex() int {
	for i, t := range AllInstanceTypes() {
		if t.ID == DefaultInstanceType {
			return i
		}
	}
	return 0
}

// ValidateInstanceType reports whether instanceType is in the allowed list.
func ValidateInstanceType(instanceType string) error {
	for _, t := range AllInstanceTypes() {
		if t.ID == instanceType {
			return nil
		}
	}
	return fmt.Errorf("invalid instance type %q", instanceType)
}

// ArchitectureForInstanceType returns the CPU architecture for a known instance type.
func ArchitectureForInstanceType(instanceType string) (string, error) {
	for _, t := range AllInstanceTypes() {
		if t.ID == instanceType {
			return t.Arch, nil
		}
	}
	return "", fmt.Errorf("invalid instance type %q", instanceType)
}

// InstanceTypesForArch returns allowed instance types matching arch.
func InstanceTypesForArch(arch string) []InstanceType {
	var out []InstanceType
	for _, t := range AllInstanceTypes() {
		if t.Arch == arch {
			out = append(out, t)
		}
	}
	return out
}
