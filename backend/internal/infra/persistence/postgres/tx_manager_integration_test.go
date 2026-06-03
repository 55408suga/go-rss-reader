//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"

	"rss_reader/internal/apperror"
)

func TestTransactionManagerCommit(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	tm := NewPgTransactionManager(testPool, quietLogger())
	feedRepo := NewFeedRepository(testPool, quietLogger())

	feed := makeFeed(t, "commit")
	err := tm.WithTransaction(ctx, func(txCtx context.Context) error {
		return feedRepo.SaveFeed(txCtx, feed)
	})
	if err != nil {
		t.Fatalf("WithTransaction: %v", err)
	}

	// Committed -> visible through the pool (outside the transaction).
	if _, err := feedRepo.GetFeedByID(ctx, feed.ID); err != nil {
		t.Errorf("expected committed feed to be visible: %v", err)
	}
}

func TestTransactionManagerRollback(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	tm := NewPgTransactionManager(testPool, quietLogger())
	feedRepo := NewFeedRepository(testPool, quietLogger())

	feed := makeFeed(t, "rollback")
	sentinel := errors.New("force rollback")
	err := tm.WithTransaction(ctx, func(txCtx context.Context) error {
		if saveErr := feedRepo.SaveFeed(txCtx, feed); saveErr != nil {
			return saveErr
		}
		return sentinel // returning an error must roll back the SaveFeed above
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("WithTransaction error = %v, want sentinel", err)
	}

	// Rolled back -> the feed must not exist.
	_, getErr := feedRepo.GetFeedByID(ctx, feed.ID)
	assertAppErrorCode(t, getErr, apperror.CodeNotFound)
}

func TestTransactionManagerRoutesRepoThroughTx(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	tm := NewPgTransactionManager(testPool, quietLogger())
	feedRepo := NewFeedRepository(testPool, quietLogger())

	feed := makeFeed(t, "routing")
	err := tm.WithTransaction(ctx, func(txCtx context.Context) error {
		if saveErr := feedRepo.SaveFeed(txCtx, feed); saveErr != nil {
			return saveErr
		}
		// Inside the same tx (txCtx), the not-yet-committed feed is visible.
		if _, visErr := feedRepo.GetFeedByID(txCtx, feed.ID); visErr != nil {
			t.Errorf("feed not visible within its own transaction: %v", visErr)
		}
		// Through the pool (outer ctx, a different connection) it must NOT be
		// visible yet under READ COMMITTED isolation.
		if _, outerErr := feedRepo.GetFeedByID(ctx, feed.ID); outerErr == nil {
			t.Error("uncommitted feed unexpectedly visible outside the transaction")
		} else {
			assertAppErrorCode(t, outerErr, apperror.CodeNotFound)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTransaction: %v", err)
	}
}
