package job

import (
	"context"
	"log/slog"
	applogger "rss_reader/internal/applog"
	"sync"
	"time"
)

// Job defines a periodic unit of work.
type Job struct {
	Name     string
	Interval time.Duration
	Timeout  time.Duration
	Fnc      func(ctx context.Context) error
}

// Scheduler executes periodic background jobs.
type Scheduler struct {
	jobs   []Job
	wg     sync.WaitGroup
	logger *slog.Logger
}

// NewJobScheduler creates a scheduler with the provided logger.
func NewJobScheduler(logger *slog.Logger) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Scheduler{logger: logger}
}

// Add appends a job to the scheduler.
func (s *Scheduler) Add(j Job) {
	s.jobs = append(s.jobs, j)
}

// Start launches all registered jobs.
func (s *Scheduler) Start(ctx context.Context) {
	for _, j := range s.jobs {
		s.wg.Go(func() {
			s.runJob(ctx, j)
		})
	}
}

func (s *Scheduler) runJob(ctx context.Context, j Job) {
	s.executeJob(ctx, j)

	ticker := time.NewTicker(j.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			applogger.WithContext(ctx, s.logger).InfoContext(ctx, "job stopping", "job", j.Name)
			return
		case <-ticker.C:
			s.executeJob(ctx, j)
		}
	}
}

func (s *Scheduler) executeJob(ctx context.Context, j Job) {
	startedAt := time.Now()
	jobLogger := applogger.WithContext(ctx, s.logger).With("job", j.Name)
	jobLogger.InfoContext(ctx, "job started")

	if ctx.Err() != nil {
		jobLogger.WarnContext(ctx, "job canceled before execution", "error", ctx.Err())
		return
	}

	jobCtx, cancel := context.WithTimeout(ctx, j.Timeout)
	defer cancel()

	if err := j.Fnc(jobCtx); err != nil {
		jobLogger.ErrorContext(ctx,
			"job failed",
			"error", err,
			"duration_ms", time.Since(startedAt).Milliseconds(),
		)
		return
	}

	jobLogger.InfoContext(ctx,
		"job completed",
		"duration_ms", time.Since(startedAt).Milliseconds(),
	)
}

// Shutdown waits for all running job goroutines to stop.
func (s *Scheduler) Shutdown() {
	s.wg.Wait()
}
