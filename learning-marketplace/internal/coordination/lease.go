package coordination

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	clientv3 "go.etcd.io/etcd/client/v3"

	"learning-marketplace/internal/config"
)

var ErrLeaseAlreadyHeld = errors.New("lease already held")

// LeaseStore owns short-lived resource claims and hands out fencing tokens.
type LeaseStore struct {
	client    *clientv3.Client
	keyPrefix string
}

type LeaseGrant struct {
	Resource     string `json:"resource"`
	Holder       string `json:"holder"`
	LeaseID      int64  `json:"lease_id"`
	FencingToken int64  `json:"fencing_token"`
	TTLSeconds   int64  `json:"ttl_seconds"`
}

// Open connects to etcd so the app can run the lease and fencing lab.
func Open(ctx context.Context, cfg config.EtcdConfig) (*LeaseStore, error) {
	client, err := clientv3.New(clientv3.Config{
		Context:     ctx,
		Endpoints:   cfg.Endpoints,
		DialTimeout: cfg.DialTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("open etcd client: %w", err)
	}

	return &LeaseStore{client: client, keyPrefix: cfg.KeyPrefix}, nil
}

func (s *LeaseStore) Close() error {
	if s == nil || s.client == nil {
		return nil
	}

	return s.client.Close()
}

// Acquire claims a resource if nobody else currently holds it and returns a fencing token.
func (s *LeaseStore) Acquire(ctx context.Context, resource, holder string, ttlSeconds int64) (LeaseGrant, error) {
	leaseResp, err := s.client.Grant(ctx, ttlSeconds)
	if err != nil {
		return LeaseGrant{}, fmt.Errorf("grant etcd lease: %w", err)
	}

	key := s.resourceKey(resource)
	txnResp, err := s.client.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
		Then(clientv3.OpPut(key, holder, clientv3.WithLease(leaseResp.ID))).
		Else(clientv3.OpGet(key)).
		Commit()
	if err != nil {
		_, _ = s.client.Revoke(ctx, leaseResp.ID)
		return LeaseGrant{}, fmt.Errorf("acquire etcd lease: %w", err)
	}

	if !txnResp.Succeeded {
		_, _ = s.client.Revoke(ctx, leaseResp.ID)
		return LeaseGrant{}, ErrLeaseAlreadyHeld
	}

	return LeaseGrant{
		Resource:     resource,
		Holder:       holder,
		LeaseID:      int64(leaseResp.ID),
		FencingToken: txnResp.Header.Revision,
		TTLSeconds:   ttlSeconds,
	}, nil
}

func (s *LeaseStore) Release(ctx context.Context, leaseID int64) error {
	if leaseID == 0 {
		return nil
	}

	_, err := s.client.Revoke(ctx, clientv3.LeaseID(leaseID))
	if err != nil {
		return fmt.Errorf("revoke etcd lease: %w", err)
	}

	return nil
}

// KeepAlive refreshes a granted lease until the caller cancels the context.
func (s *LeaseStore) KeepAlive(ctx context.Context, leaseID int64) error {
	if leaseID == 0 {
		return nil
	}

	ch, err := s.client.KeepAlive(ctx, clientv3.LeaseID(leaseID))
	if err != nil {
		return fmt.Errorf("keep etcd lease alive: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-ch:
				if !ok {
					return
				}
			}
		}
	}()

	return nil
}

func (s *LeaseStore) CurrentHolder(ctx context.Context, resource string) (*LeaseGrant, error) {
	key := s.resourceKey(resource)
	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("get etcd holder: %w", err)
	}
	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	kv := resp.Kvs[0]
	ttlResp, err := s.client.TimeToLive(ctx, clientv3.LeaseID(kv.Lease))
	if err != nil {
		return nil, fmt.Errorf("get etcd ttl: %w", err)
	}

	return &LeaseGrant{
		Resource:     resource,
		Holder:       string(kv.Value),
		LeaseID:      kv.Lease,
		FencingToken: kv.ModRevision,
		TTLSeconds:   ttlResp.TTL,
	}, nil
}

func (s *LeaseStore) resourceKey(resource string) string {
	return path.Join(s.keyPrefix, strings.TrimSpace(resource))
}
