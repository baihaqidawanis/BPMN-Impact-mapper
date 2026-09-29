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
}
