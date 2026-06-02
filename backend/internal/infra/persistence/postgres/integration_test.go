//go:build integration

package postgres

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
)

// testPool is the shared pgx pool against a throwaway Postgres container,
// initialised once in TestMain. These tests run serially (no t.Parallel) and
// reset the database between cases via resetDB.
//
// Run with: go test -tags=integration ./... (requires a running Docker daemon).
var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:18-alpine",
		postgres.WithDatabase("rss_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.WithInitScripts(filepath.Join("sql", "schema.sql")),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(90*time.Second),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start postgres container: %v\n", err)
		os.Exit(1)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build connection string: %v\n", err)
		os.Exit(1)
	}

	testPool, err = pgxpool.New(ctx, connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create pgx pool: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	testPool.Close()
	if termErr := container.Terminate(ctx); termErr != nil {
		fmt.Fprintf(os.Stderr, "failed to terminate container: %v\n", termErr)
	}
	os.Exit(code)
}

// resetDB truncates every table so each test starts from a clean slate.
func resetDB(t *testing.T) {
	t.Helper()
	_, err := testPool.Exec(context.Background(), "TRUNCATE feed_fetch_status, articles, feeds CASCADE")
	if err != nil {
		t.Fatalf("failed to reset database: %v", err)
	}
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func assertAppErrorCode(t *testing.T, err error, want apperror.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %q, got nil", want)
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.AppError, got %T: %v", err, err)
	}
	if appErr.Code != want {
		t.Errorf("error code = %q, want %q", appErr.Code, want)
	}
}

// makeFeed builds a feed with URLs made unique by suffix (feeds.feed_url and
// feeds.website_url are both UNIQUE).
func makeFeed(t *testing.T, suffix string) *model.Feed {
	t.Helper()
	feed, err := model.NewFeed(
		"Feed "+suffix,
		"https://example.com/"+suffix+"/feed.xml",
		"https://example.com/"+suffix,
		"description",
		"en",
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("NewFeed: %v", err)
	}
	return feed
}

// makeArticle builds an article parented to feedID with the given external ID.
func makeArticle(t *testing.T, feedID uuid.UUID, externalID string) *model.Article {
	t.Helper()
	article, err := model.NewArticle(
		"Article "+externalID,
		"description",
		"content",
		"https://example.com/posts/"+externalID,
		time.Now().UTC(),
		feedID,
		externalID,
	)
	if err != nil {
		t.Fatalf("NewArticle: %v", err)
	}
	return article
}

// saveParentFeed persists and returns a parent feed for FK-constrained tests.
func saveParentFeed(t *testing.T, suffix string) *model.Feed {
	t.Helper()
	feed := makeFeed(t, suffix)
	if err := NewFeedRepository(testPool, quietLogger()).SaveFeed(context.Background(), feed); err != nil {
		t.Fatalf("save parent feed: %v", err)
	}
	return feed
}
