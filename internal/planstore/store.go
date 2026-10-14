// Package planstore defines persistence for immutable, evidence-grounded plans.
package planstore

import (
	"context"
	"errors"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

var ErrNotFound = errors.New("change plan not found")

type Store interface {
	Save(context.Context, planning.ChangePlan) error
	Get(context.Context, string) (planning.ChangePlan, error)
	ListByAnalysis(context.Context, string) ([]planning.PlanSummary, error)
}
