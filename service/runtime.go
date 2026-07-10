package service

import (
	"context"
	"sync"

	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	awsclient "outpost-cli/service/aws"
	localDb "outpost-cli/service/localDb"
)

// Runtime holds shared DB and AWS clients for a single CLI invocation.
type Runtime struct {
	ctx    context.Context
	cancel context.CancelFunc
	db     *localDb.DB

	ec2Mu          sync.Mutex             // mutex for the ec2 clients, so we don't have race conditions when accessing the clients
	ec2ByRegion    map[string]*ec2.Client // map of ec2 clients by region
	ec2ErrByRegion map[string]error       // map of errors by region
}

// Open connects to the local database. AWS clients are created lazily on first use.
// called by helper/runtime.go
func Open(ctx context.Context, cancel context.CancelFunc) (*Runtime, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	db, err := localDb.Open()
	if err != nil {
		if cancel != nil {
			cancel() // cancel the context if the database fails to open, we return anyways
		}
		return nil, err
	}

	// if all good, return the runtime, and nil error
	return &Runtime{ctx: ctx, cancel: cancel, db: db}, nil
}

// Close releases the database connection.
func (r *Runtime) Close() error {
	if r.cancel != nil {
		// if cancel is not nil, cancel the context
		r.cancel()
		r.cancel = nil
	}
	if r.db == nil {
		return nil
	}
	return r.db.Close()
}

// DB returns the shared local database connection.
func (r *Runtime) DB() *localDb.DB {
	return r.db
}

// Context returns the runtime context.
func (r *Runtime) Context() context.Context {
	if r.ctx == nil {
		return context.Background()
	}
	return r.ctx
}

// EC2 returns a shared EC2 client for the configured default region.
func (r *Runtime) EC2() (*ec2.Client, error) {
	cfg, err := awsclient.LoadConfig()
	if err != nil {
		return nil, err
	}
	if cfg.AwsRegion == "" {
		return nil, fmt.Errorf("aws region is required; run: outpost setup")
	}
	return r.EC2ForRegion(cfg.AwsRegion)
}

// EC2ForRegion returns a cached EC2 client for the given AWS region.
func (r *Runtime) EC2ForRegion(region string) (*ec2.Client, error) {
	r.ec2Mu.Lock() // lock the mutex so we don't have race conditions when accessing the clients
	defer r.ec2Mu.Unlock()

	if r.ec2ByRegion == nil {
		r.ec2ByRegion = make(map[string]*ec2.Client)
		r.ec2ErrByRegion = make(map[string]error)
	}
	if client, ok := r.ec2ByRegion[region]; ok {
		if err := r.ec2ErrByRegion[region]; err != nil {
			return nil, err
		}
		return client, nil
	}

	awsClient, err := awsclient.NewClientForRegion(r.Context(), region)
	if err != nil {
		r.ec2ErrByRegion[region] = err
		return nil, err
	}
	client := ec2.NewFromConfig(awsClient.Config())
	r.ec2ByRegion[region] = client
	return client, nil
}
