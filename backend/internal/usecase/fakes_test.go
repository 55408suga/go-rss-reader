package usecase

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"rss_reader/internal/apperror"
	"rss_reader/internal/domain/model"
)

// This file holds hand-written, programmable test doubles shared across the
// usecase tests. The SaveXxx methods are mutex-guarded because
// FeedJobInteractor.RefreshDueFeeds drives them from multiple goroutines
// (errgroup). Read the captured slices only after the interactor call returns —
// errgroup.Wait establishes the happens-before edge that makes that safe.

// --- repository.FeedRepository ---

type fakeFeedRepo struct {
	mu sync.Mutex

	existsResult bool
	existsErr    error
	getFeed      *model.Feed
	getErr       error
	listFeeds    []*model.Feed
	listErr      error
	saveErr      error
	updateErr    error
	deleteErr    error

	byWebsiteFeed *model.Feed
	byWebsiteErr  error

	checkCalls     int
	byWebsiteCalls int
	gotWebsiteURLs []string
	savedFeeds     []*model.Feed
	updatedFeeds   []*model.Feed
	deletedIDs     []uuid.UUID
	gotListLimit   int
}

func (f *fakeFeedRepo) GetFeedByWebsiteURL(_ context.Context, websiteURLs []string) (*model.Feed, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byWebsiteCalls++
	f.gotWebsiteURLs = websiteURLs
	return f.byWebsiteFeed, f.byWebsiteErr
}

func (f *fakeFeedRepo) CheckFeedExistsByURL(_ context.Context, _ string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.checkCalls++
	return f.existsResult, f.existsErr
}

func (f *fakeFeedRepo) SaveFeed(_ context.Context, feed *model.Feed) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.savedFeeds = append(f.savedFeeds, feed)
	return f.saveErr
}

func (f *fakeFeedRepo) GetFeedByID(_ context.Context, _ uuid.UUID) (*model.Feed, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.getFeed, f.getErr
}

func (f *fakeFeedRepo) ListFeeds(
	_ context.Context, _ *model.PageCursor, limit int,
) ([]*model.Feed, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gotListLimit = limit
	return f.listFeeds, f.listErr
}

func (f *fakeFeedRepo) UpdateFeed(_ context.Context, feed *model.Feed) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updatedFeeds = append(f.updatedFeeds, feed)
	return f.updateErr
}

func (f *fakeFeedRepo) DeleteFeed(_ context.Context, feedID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deletedIDs = append(f.deletedIDs, feedID)
	return f.deleteErr
}

// --- repository.ArticleRepository ---

type fakeArticleRepo struct {
	mu sync.Mutex

	listByFeed    []*model.Article
	listByFeedErr error
	listAll       []*model.Article
	listAllErr    error
	saveErr       error

	savedArticles []*model.Article
	gotListLimit  int
}

func (f *fakeArticleRepo) SaveArticle(_ context.Context, article *model.Article) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.savedArticles = append(f.savedArticles, article)
	return f.saveErr
}

func (f *fakeArticleRepo) GetArticleByID(_ context.Context, _ uuid.UUID) (*model.Article, error) {
	return nil, nil
}

func (f *fakeArticleRepo) ListArticlesByFeedID(
	_ context.Context, _ uuid.UUID, _ *model.PageCursor, limit int,
) ([]*model.Article, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gotListLimit = limit
	return f.listByFeed, f.listByFeedErr
}

func (f *fakeArticleRepo) ListArticles(
	_ context.Context, _ *model.PageCursor, limit int,
) ([]*model.Article, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gotListLimit = limit
	return f.listAll, f.listAllErr
}

func (f *fakeArticleRepo) UpdateArticle(_ context.Context, _ *model.Article) error { return nil }
func (f *fakeArticleRepo) DeleteArticle(_ context.Context, _ uuid.UUID) error      { return nil }

// --- repository.FetchStatusRepository ---

type fakeFetchStatusRepo struct {
	mu sync.Mutex

	dueFeeds []*model.DueFeed
	dueErr   error
	saveErr  error

	savedStatuses []*model.FetchStatus
}

func (f *fakeFetchStatusRepo) SaveFetchStatus(_ context.Context, status *model.FetchStatus) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.savedStatuses = append(f.savedStatuses, status)
	return f.saveErr
}

func (f *fakeFetchStatusRepo) GetFetchStatusByFeedID(
	_ context.Context, _ uuid.UUID,
) (*model.FetchStatus, error) {
	return nil, nil
}

func (f *fakeFetchStatusRepo) GetDueFeeds(
	_ context.Context, _ time.Time, _ int,
) ([]*model.DueFeed, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.dueFeeds, f.dueErr
}

// --- FeedDiscoverer ---

type fakeDiscoverer struct {
	mu sync.Mutex

	candidates []model.FeedCandidate
	err        error

	calls  int
	gotURL string
}

func (f *fakeDiscoverer) DiscoverFeedURLs(
	_ context.Context, websiteURL string,
) ([]model.FeedCandidate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.gotURL = websiteURL
	return f.candidates, f.err
}

// --- RSSFetcher ---

type fakeFetcher struct {
	newFeed     *model.Feed
	newArticles []*model.Article
	newCursor   *model.FeedCursor
	newErr      error

	mu        sync.Mutex
	newCalls  int
	gotNewURL string

	// withCursorFunc lets the job tests vary behaviour per feed URL. It is set
	// before use and never mutated during a call, so it is read without locking.
	withCursorFunc func(
		url string, cursor *model.FeedCursor,
	) (*model.Feed, []*model.Article, *model.FeedCursor, error)
}

func (f *fakeFetcher) FetchNewFeed(
	_ context.Context, feedURL string,
) (*model.Feed, []*model.Article, *model.FeedCursor, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.newCalls++
	f.gotNewURL = feedURL
	return f.newFeed, f.newArticles, f.newCursor, f.newErr
}

func (f *fakeFetcher) FetchFeedWithCursor(
	_ context.Context, url string, cursor *model.FeedCursor,
) (*model.Feed, []*model.Article, *model.FeedCursor, error) {
	if f.withCursorFunc != nil {
		return f.withCursorFunc(url, cursor)
	}
	return nil, nil, nil, nil
}

// --- TransactionManager ---

// fakeTxManager runs fn inline without a real transaction.
type fakeTxManager struct{}

func (fakeTxManager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// assertAppErrorCode fails unless err is a non-nil *apperror.AppError carrying want.
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
