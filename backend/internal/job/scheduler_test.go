package job

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"rss_reader/internal/apperror"
)

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

func TestNewJobSchedulerDefaultsLogger(t *testing.T) {
	t.Parallel()
	if NewJobScheduler(nil) == nil {
		t.Error("NewJobScheduler(nil) = nil, want a scheduler")
	}
}

func TestAddRegistersJobs(t *testing.T) {
	t.Parallel()
	s := NewJobScheduler(quietLogger())
	s.Add(Job{Name: "a"})
	s.Add(Job{Name: "b"})
	if len(s.jobs) != 2 {
		t.Errorf("len(jobs) = %d, want 2", len(s.jobs))
	}
}

func TestExecuteJobRunsFuncWithDeadline(t *testing.T) {
	t.Parallel()
	s := NewJobScheduler(quietLogger())

	ran := false
	var hadDeadline bool
	s.executeJob(context.Background(), Job{
		Name:    "j",
		Timeout: time.Second,
		Func: func(ctx context.Context) error {
			ran = true
			_, hadDeadline = ctx.Deadline()
			return nil
		},
	})

	if !ran {
		t.Error("expected Func to run")
	}
	if !hadDeadline {
		t.Error("expected the per-job timeout to give Func a context deadline")
	}
}

func TestExecuteJobSkipsWhenContextAlreadyCancelled(t *testing.T) {
	t.Parallel()
	s := NewJobScheduler(quietLogger())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ran := false
	s.executeJob(ctx, Job{
		Name:    "j",
		Timeout: time.Second,
		Func:    func(context.Context) error { ran = true; return nil },
	})

	if ran {
		t.Error("expected Func to be skipped when the context is already cancelled")
	}
}

func TestExecuteJobHandlesFuncError(t *testing.T) {
	t.Parallel()
	s := NewJobScheduler(quietLogger())

	// A failing Func must be handled (logged) without panicking.
	s.executeJob(context.Background(), Job{
		Name:    "j",
		Timeout: time.Second,
		Func:    func(context.Context) error { return errors.New("boom") },
	})
}

func TestStartRunsJobImmediately(t *testing.T) {
	t.Parallel()
	s := NewJobScheduler(quietLogger())
	ctx, cancel := context.WithCancel(context.Background())

	ran := make(chan struct{}, 1)
	s.Add(Job{
		Name:     "immediate",
		Interval: time.Hour, // long enough that the ticker never fires during the test
		Timeout:  time.Second,
		Func: func(context.Context) error {
			select {
			case ran <- struct{}{}:
			default:
			}
			return nil
		},
	})

	s.Start(ctx)

	select {
	case <-ran:
	case <-time.After(2 * time.Second):
		t.Fatal("job did not run within the timeout")
	}

	cancel()
	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
}

func TestShutdownTimesOutWhenJobBlocks(t *testing.T) {
	t.Parallel()
	s := NewJobScheduler(quietLogger())
	ctx, cancel := context.WithCancel(context.Background())

	block := make(chan struct{})
	t.Cleanup(func() {
		close(block) // let the blocked job finish
		cancel()     // let runJob's ticker loop exit
	})

	started := make(chan struct{}, 1)
	s.Add(Job{
		Name:     "blocker",
		Interval: time.Hour,
		Timeout:  time.Hour,
		Func: func(context.Context) error {
			select {
			case started <- struct{}{}:
			default:
			}
			<-block
			return nil
		},
	})

	s.Start(ctx)
	<-started // ensure the goroutine is inside Func before shutting down

	shutdownCtx, scancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer scancel()

	err := s.Shutdown(shutdownCtx)
	assertAppErrorCode(t, err, apperror.CodeInternal)
}
