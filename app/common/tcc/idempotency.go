package tcc

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Phase represents a TCC step name
type Phase string

const (
	PhaseTry     Phase = "try"
	PhaseConfirm Phase = "confirm"
	PhaseCancel  Phase = "cancel"
)

// IdempotencyStore provides strong idempotency per (txID, componentID, phase)
type IdempotencyStore struct {
	redis *redis.Client
}

func NewIdempotencyStore(redis *redis.Client) *IdempotencyStore {
	return &IdempotencyStore{redis: redis}
}

func (s *IdempotencyStore) key(txID, componentID string, phase Phase) string {
	return fmt.Sprintf("tcc:idem:%s:%s:%s", txID, componentID, phase)
}

// CheckAndMark returns true if this (tx,component,phase) was already processed; if not, it marks it and returns false.
// ttl bounds how long idempotency records live to avoid unbounded growth.
func (s *IdempotencyStore) CheckAndMark(ctx context.Context, txID, componentID string, phase Phase, ttl time.Duration) (already bool, err error) {
	key := s.key(txID, componentID, phase)
	res := s.redis.SetNX(ctx, key, "done", ttl)
	if res.Err() != nil {
		return false, res.Err()
	}
	// SetNX returns true if the key was set (i.e., not existing). If false, it already existed.
	return !res.Val(), nil
}

// MarkExplicit marks a phase as completed explicitly (used when short-circuiting empty rollback)
func (s *IdempotencyStore) MarkExplicit(ctx context.Context, txID, componentID string, phase Phase, ttl time.Duration) error {
	key := s.key(txID, componentID, phase)
	return s.redis.Set(ctx, key, "done", ttl).Err()
}
