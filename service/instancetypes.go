package service

import "fmt"

// DefaultInstanceType is the instance type used when no interactive picker is shown.
const DefaultInstanceType = "t4g.small"

// InstanceType is an EC2 instance type with a human-readable vCPU/RAM label.
type InstanceType struct {
	ID    string
	Label string
}

// AllInstanceTypes returns every allowed t4g size for the create picker.
func AllInstanceTypes() []InstanceType {
	return []InstanceType{
		{ID: "t4g.nano", Label: "2 vCPU, 0.5 GiB RAM"},
		{ID: "t4g.micro", Label: "2 vCPU, 1 GiB RAM"},
		{ID: "t4g.small", Label: "2 vCPU, 2 GiB RAM"},
		{ID: "t4g.medium", Label: "2 vCPU, 4 GiB RAM"},
		{ID: "t4g.large", Label: "2 vCPU, 8 GiB RAM"},
		{ID: "t4g.xlarge", Label: "4 vCPU, 16 GiB RAM"},
		{ID: "t4g.2xlarge", Label: "8 vCPU, 32 GiB RAM"},
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

// ValidateInstanceType reports whether instanceType is in the allowed t4g list.
func ValidateInstanceType(instanceType string) error {
	for _, t := range AllInstanceTypes() {
		if t.ID == instanceType {
			return nil
		}
	}
	return fmt.Errorf("invalid instance type %q", instanceType)
}
