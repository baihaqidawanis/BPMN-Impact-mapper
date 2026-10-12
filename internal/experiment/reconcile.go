package experiment

import (
	"fmt"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/review"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/reviewpack"
)

func RunReconcile(reviewerAPath, reviewerBPath, outputDirectory string) (review.Reconciliation, error) {
	return RunReconcileWithManifest(reviewerAPath, reviewerBPath, "", outputDirectory)
}

func RunReconcileWithManifest(reviewerAPath, reviewerBPath, manifestPath, outputDirectory string) (review.Reconciliation, error) {
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
	if manifestPath != "" {
		manifestBytes, err := readRegular(manifestPath, 8<<20, "review pack manifest")
		if err != nil {
			return review.Reconciliation{}, err
		}
		var manifest reviewpack.Manifest
		if err := decodeStrict(manifestBytes, &manifest); err != nil {
			return review.Reconciliation{}, fmt.Errorf("decode review pack manifest: %w", err)
		}
		if err := reviewpack.ValidateCompletedCases(manifest, a.Cases); err != nil {
			return review.Reconciliation{}, fmt.Errorf("reviewer A does not match review pack: %w", err)
		}
		if err := reviewpack.ValidateCompletedCases(manifest, b.Cases); err != nil {
			return review.Reconciliation{}, fmt.Errorf("reviewer B does not match review pack: %w", err)
		}
		result.ReviewPack = &review.PackProvenance{PackID: manifest.PackID, InputFingerprint: manifest.InputFingerprint, CaseDigest: manifest.CaseDigest}
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
