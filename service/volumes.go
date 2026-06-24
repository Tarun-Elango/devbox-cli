package service

import "fmt"

// DefaultVolumeSizeGB is the root volume size used when no interactive picker is shown.
const DefaultVolumeSizeGB = 16

// MinVolumeSizeGB and MaxVolumeSizeGB bound allowed gp3 root volume sizes (EC2 gp3 limits).
const (
	MinVolumeSizeGB = 8
	MaxVolumeSizeGB = 500
)

// ValidateVolumeSizeGB reports whether sizeGB is within the allowed gp3 range.
func ValidateVolumeSizeGB(sizeGB int) error {
	if sizeGB < MinVolumeSizeGB || sizeGB > MaxVolumeSizeGB {
		return fmt.Errorf("invalid volume size %d GB (allowed: %d–%d GB)", sizeGB, MinVolumeSizeGB, MaxVolumeSizeGB)
	}
	return nil
}
