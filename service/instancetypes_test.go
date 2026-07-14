package service

import (
	"strings"
	"testing"
)

func TestValidateInstanceType(t *testing.T) {
	for _, id := range []string{
		"t4g.medium",
		"g4dn.xlarge",
		"g6.xlarge",
		"g5.xlarge",
		"g5.2xlarge",
	} {
		if err := ValidateInstanceType(id); err != nil {
			t.Fatalf("ValidateInstanceType(%q) = %v", id, err)
		}
	}
	if err := ValidateInstanceType("t3.micro"); err == nil {
		t.Fatal("expected error for t3.micro")
	}
}

func TestArchitectureForInstanceType(t *testing.T) {
	cases := map[string]string{
		"t4g.nano":    ArchARM64,
		"t4g.medium":  ArchARM64,
		"t4g.2xlarge": ArchARM64,
		"g4dn.xlarge": ArchX86_64,
		"g6.xlarge":   ArchX86_64,
		"g5.xlarge":   ArchX86_64,
		"g5.2xlarge":  ArchX86_64,
	}
	for id, want := range cases {
		got, err := ArchitectureForInstanceType(id)
		if err != nil {
			t.Fatalf("ArchitectureForInstanceType(%q): %v", id, err)
		}
		if got != want {
			t.Fatalf("ArchitectureForInstanceType(%q) = %q, want %q", id, got, want)
		}
	}
	if _, err := ArchitectureForInstanceType("m5.large"); err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestInstanceTypesForArch(t *testing.T) {
	arm := InstanceTypesForArch(ArchARM64)
	if len(arm) != 7 {
		t.Fatalf("arm64 count = %d, want 7", len(arm))
	}
	for _, tpe := range arm {
		if tpe.Arch != ArchARM64 {
			t.Fatalf("%s has arch %q, want %s", tpe.ID, tpe.Arch, ArchARM64)
		}
	}

	x86 := InstanceTypesForArch(ArchX86_64)
	if len(x86) != 4 {
		t.Fatalf("x86_64 count = %d, want 4", len(x86))
	}
	wantIDs := []string{"g4dn.xlarge", "g6.xlarge", "g5.xlarge", "g5.2xlarge"}
	for i, want := range wantIDs {
		if x86[i].ID != want {
			t.Fatalf("x86[%d] = %q, want %q", i, x86[i].ID, want)
		}
		if !strings.Contains(x86[i].Label, "NVIDIA") {
			t.Fatalf("%s label %q missing GPU spec", x86[i].ID, x86[i].Label)
		}
	}
}
