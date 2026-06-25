package service

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	localDb "devbox-cli/service/localDb"
)

type fakeCreateTagsClient struct {
	input *ec2.CreateTagsInput
}

func (f *fakeCreateTagsClient) CreateTags(ctx context.Context, input *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error) {
	f.input = input
	return &ec2.CreateTagsOutput{}, nil
}

func TestCreateInstanceRejectsDuplicateNameBeforeAWS(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	db, err := localDb.Open()
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := db.InsertInstance("box-1", "i-1234567890abcdef0", "alpha", LocalUserID, "running", DefaultInstanceType); err != nil {
		t.Fatalf("insert existing instance: %v", err)
	}

	rt := &Runtime{ctx: context.Background(), db: db}
	_, err = rt.CreateInstance("alpha", "", "", LocalUserID, DefaultInstanceType, DefaultVolumeSizeGB)
	if err == nil {
		t.Fatal("expected duplicate name error")
	}
	if !strings.Contains(err.Error(), "box name already exists: alpha") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateInstanceNameTagCreatesNameTag(t *testing.T) {
	client := &fakeCreateTagsClient{}

	if err := updateInstanceNameTag(context.Background(), client, "i-1234567890abcdef0", "beta"); err != nil {
		t.Fatalf("update instance name tag: %v", err)
	}
	if client.input == nil {
		t.Fatal("CreateTags was not called")
	}
	if len(client.input.Resources) != 1 || client.input.Resources[0] != "i-1234567890abcdef0" {
		t.Fatalf("resources = %v, want instance id", client.input.Resources)
	}
	if len(client.input.Tags) != 1 {
		t.Fatalf("tags = %v, want one Name tag", client.input.Tags)
	}
	if aws.ToString(client.input.Tags[0].Key) != "Name" || aws.ToString(client.input.Tags[0].Value) != "beta" {
		t.Fatalf("tag = %v, want Name=beta", client.input.Tags[0])
	}
}
