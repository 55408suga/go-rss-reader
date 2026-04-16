package postgres

import (
	"errors"
	"rss_reader/internal/apperror"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestClassifyDBErrorNotFound(t *testing.T) {
	err := classifyDBError(pgx.ErrNoRows, "op")
	if err == nil {
		t.Fatalf("expected classified error")
	}
	if err.Code != apperror.CodeNotFound {
		t.Fatalf("expected code %q, got %q", apperror.CodeNotFound, err.Code)
	}
}

func TestClassifyDBErrorConflict(t *testing.T) {
	err := classifyDBError(&pgconn.PgError{Code: "23505"}, "op")
	if err == nil {
		t.Fatalf("expected classified error")
	}
	if err.Code != apperror.CodeConflict {
		t.Fatalf("expected code %q, got %q", apperror.CodeConflict, err.Code)
	}
}

func TestClassifyDBErrorInternal(t *testing.T) {
	err := classifyDBError(errors.New("db down"), "op")
	if err == nil {
		t.Fatalf("expected classified error")
	}
	if err.Code != apperror.CodeInternal {
		t.Fatalf("expected code %q, got %q", apperror.CodeInternal, err.Code)
	}
}
