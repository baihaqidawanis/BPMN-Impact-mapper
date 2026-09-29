package experiment

import (
	"fmt"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/review"
)

func RunReconcile(reviewerAPath, reviewerBPath, outputDirectory string) (review.Reconciliation, error) {
	aBytes, err := readRegular(reviewerAPath, 8<<20, "reviewer A annotation")
	if err != nil {
		return review.Reconciliation{}, err
	}
	bBytes, err := readRegular(reviewerBPath, 8<<20, "reviewer B annotation")
	if err != nil {
		return review.Reconciliation{}, err
	}
	var a, b review.Annotation
	if err := decodeStrict(aBytes, &a); err != nil {
		return review.Reconciliation{}, fmt.Errorf("decode reviewer A: %w", err)
	}
	if err := decodeStrict(bBytes, &b); err != nil {
		return review.Reconciliation{}, fmt.Errorf("decode reviewer B: %w", err)
	}
	result, err := review.Reconcile(a, b)
	if err != nil {
		return review.Reconciliation{}, err
	}
	store, err := NewLocalFiles(outputDirectory)
	if err != nil {
		return review.Reconciliation{}, err
	}
	data, err := review.JSON(result)
	if err != nil {
		return review.Reconciliation{}, err
	}
	if err := store.Write("reconciliation.json", data); err != nil {
		return review.Reconciliation{}, err
	}
	if err := store.Write("reconciliation.md", review.Markdown(result)); err != nil {
		return review.Reconciliation{}, err
	}
	return result, nil
}
