package bootstrap

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func Lower(source Source) (SemanticIR, error) {
	input := source.Input
	if err := validateInput(input); err != nil {
		return SemanticIR{}, err
	}
	input.ProtectedPaths = SortedStrings(input.ProtectedPaths)
	input.ProposedAuthorityDelta.Touches = SortedStrings(input.ProposedAuthorityDelta.Touches)
	input.SignerIdentities = sortedIdentities(input.SignerIdentities)
	input.ApproverIdentities = sortedIdentities(input.ApproverIdentities)
	input.ObservedArtifacts.Paths.Paths = SortedStrings(input.ObservedArtifacts.Paths.Paths)
	input.ObservedArtifacts.Protection.RequiredStatusChecks = normalizeChecks(input.ObservedArtifacts.Protection.RequiredStatusChecks)
	input.BranchProtection.RequiredStatusChecks = normalizeChecks(input.BranchProtection.RequiredStatusChecks)
	bindings := make([]CellActivityBinding, 0, len(input.ProtectedPaths))
	for _, path := range input.ProtectedPaths {
		bindings = append(bindings, CellActivityBinding{
			Cell:     "protected-path:" + path,
			Activity: "observe-protected-path:" + path,
			Source:   "protected_paths[" + path + "]",
		})
	}
	ir := SemanticIR{
		Version:      "gooo-ir/v1",
		SourceDigest: source.SourceDigest,
		Input:        input,
		Denominators: Denominators{
			CellCount:     len(bindings),
			ActivityCount: len(bindings),
			OneToOne:      true,
			Bindings:      bindings,
		},
	}
	digest, err := digestValue(struct {
		Version      string        `json:"version"`
		SourceDigest string        `json:"source_digest"`
		Input        CeremonyInput `json:"input"`
		Denominators Denominators  `json:"denominators"`
	}{ir.Version, ir.SourceDigest, ir.Input, ir.Denominators})
	if err != nil {
		return SemanticIR{}, err
	}
	ir.IRDigest = digest
	return ir, nil
}

func validateInput(input CeremonyInput) error {
	if input.Choice != ChoiceFoundation && input.Choice != ChoiceCoherence && input.Choice != ChoiceRegression {
		return fmt.Errorf("choice must be FOUNDATION, COHERENCE, or REGRESSION")
	}
	if input.Repository == "" || input.Tuple.Repository == "" || input.Repository != input.Tuple.Repository {
		return fmt.Errorf("repository must be bound identically in input and tuple")
	}
	if input.Tuple.Base == "" || input.Tuple.Head == "" || input.Tuple.PR <= 0 {
		return fmt.Errorf("base, head, and positive PR must be bound")
	}
	if input.Candidate.ID == "" || input.Candidate.Branch == "" || input.Candidate.EditDigest == "" {
		return fmt.Errorf("candidate id, branch, and edit digest must be bound")
	}
	if len(input.ProtectedPaths) == 0 {
		return fmt.Errorf("protected path set must be non-empty")
	}
	if hasDuplicate(input.ProtectedPaths) {
		return fmt.Errorf("protected path set contains duplicates")
	}
	if err := validateDigest("current ownership policy digest", input.CurrentOwnershipPolicyDigest); err != nil {
		return err
	}
	if err := validateDigest("immutable evaluator release digest", input.ImmutableEvaluatorReleaseDigest); err != nil {
		return err
	}
	if err := validateDigest("branch protection snapshot digest", input.BranchProtection.Digest); err != nil {
		return err
	}
	if input.BranchProtection.Source == "" || input.BranchProtection.ArtifactID == "" {
		return fmt.Errorf("branch protection snapshot source and artifact id must be bound")
	}
	if err := validateTimestampSet(input.Timestamps); err != nil {
		return err
	}
	if input.ObservedArtifacts.Paths.ExpectedCount < 0 || input.ObservedArtifacts.Paths.ObservedCount < 0 {
		return fmt.Errorf("path observation counts cannot be negative")
	}
	if len(input.ObservedArtifacts.Paths.Paths) > input.ObservedArtifacts.Paths.ObservedCount {
		return fmt.Errorf("path observation lists more paths than its observed count")
	}
	if input.ObservedArtifacts.Paths.Tuple.Repository == "" {
		return fmt.Errorf("path observation tuple must be bound")
	}
	return nil
}

func validateTimestampSet(timestamps Timestamps) error {
	values := []struct {
		name  string
		value string
	}{
		{"candidate_started_at", timestamps.CandidateStartedAt},
		{"observed_at", timestamps.ObservedAt},
		{"signed_at", timestamps.SignedAt},
		{"approved_at", timestamps.ApprovedAt},
		{"issued_at", timestamps.IssuedAt},
	}
	for _, value := range values {
		if value.value == "" {
			return fmt.Errorf("timestamp %s must be bound", value.name)
		}
		if _, err := time.Parse(time.RFC3339Nano, value.value); err != nil {
			return fmt.Errorf("timestamp %s must be RFC3339: %w", value.name, err)
		}
	}
	return nil
}

func normalizeChecks(checks *RequiredStatusChecks) *RequiredStatusChecks {
	if checks == nil {
		return nil
	}
	copy := &RequiredStatusChecks{Contexts: SortedStrings(checks.Contexts)}
	return copy
}

func sortedIdentities(values []IdentityRef) []IdentityRef {
	result := append([]IdentityRef(nil), values...)
	sort.Slice(result, func(i, j int) bool {
		if result[i].ID != result[j].ID {
			return result[i].ID < result[j].ID
		}
		return result[i].ArtifactID < result[j].ArtifactID
	})
	return result
}

func hasDuplicate(values []string) bool {
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" || seen[value] {
			return true
		}
		seen[value] = true
	}
	return false
}
