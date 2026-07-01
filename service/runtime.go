package service

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	awsclient "devbox-cli/service/aws"
	localDb "devbox-cli/service/localDb"
)

// Runtime holds shared DB and AWS clients for a single CLI invocation.
type Runtime struct {
	ctx    context.Context
	cancel context.CancelFunc
	db     *localDb.DB

	ec2Once sync.Once // lazy loading of the ec2 client
	ec2     *ec2.Client
	ec2Err  error
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

// EC2 returns a shared EC2 client, loading AWS config on first call.
func (r *Runtime) EC2() (*ec2.Client, error) {
	// on first call, load the aws config and create the ec2 client
	// on subsequent calls, return the cached client
	r.ec2Once.Do(func() {
		client, err := awsclient.NewClient(r.Context())
		if err != nil {
			r.ec2Err = err
			return
		}
		r.ec2 = ec2.NewFromConfig(client.Config()) // create the ec2 client from the aws config
	})
	if r.ec2Err != nil {
		return nil, r.ec2Err
	}
	return r.ec2, nil
}
