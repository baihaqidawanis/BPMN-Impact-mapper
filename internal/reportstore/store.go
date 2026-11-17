// Package reportstore defines the persistence port for completed analyses.
package reportstore

import (
	"context"
	"errors"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

var ErrNotFound = errors.New("analysis report not found")

type Store interface {
	Save(context.Context, domain.Report) error
	Get(context.Context, string) (domain.Report, error)
	List(context.Context) ([]domain.ReportSummary, error)
}

// BPMNStore persists immutable uploaded specifications separately from reports.
type BPMNStore interface {
	SaveBPMN(context.Context, domain.BPMNPair) error
	GetBPMN(context.Context, string) (domain.BPMNPair, error)
}
