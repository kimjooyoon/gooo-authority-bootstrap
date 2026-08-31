package bootstrap

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type ArtifactFile struct {
	Path         string `json:"path"`
	Bytes        int    `json:"bytes"`
	PhysicalLines int   `json:"physical_lines"`
	Digest       string `json:"digest"`
}

type ArtifactManifest struct {
	SchemaVersion string         `json:"schema_version"`
	Files         []ArtifactFile `json:"files"`
}

func BuildMetrics(inventoryRoot string, ir SemanticIR) (Metrics, error) {
	metrics := Metrics{
		BuildWall:                 "not-observed",
		TestWall:                  "not-observed",
		PeakRSS:                   "not-observed",
		NotObserved:               3,
		CellDenominator:           ir.Denominators.CellCount,
		ActivityDenominator:       ir.Denominators.ActivityCount,
		OneToOneBindings:          len(ir.Denominators.Bindings),
		ProductRepositoryWrites:   0,
		LocalTestExecutions:       0,
		CrossProjectRequiredGates: 0,
	}
	if inventoryRoot == "" {
		return metrics, nil
	}
	err := filepath.WalkDir(inventoryRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			relative, relativeErr := filepath.Rel(inventoryRoot, path)
			if relativeErr == nil && relative == ".git" {
				return fs.SkipDir
			}
			metrics.Directories++
			return nil
		}
		relative, relativeErr := filepath.Rel(inventoryRoot, path)
		if relativeErr == nil && relative == "README.md" {
			return nil
		}
		metrics.Files++
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines := physicalLines(data)
		if strings.HasSuffix(entry.Name(), ".go") {
			metrics.GoPhysicalLines += lines
		}
		if strings.HasSuffix(entry.Name(), ".gooo") {
			metrics.GoooPhysicalLines += lines
		}
		return nil
	})
	return metrics, err
}

func WriteArtifacts(outputDir string, ir SemanticIR, evaluation Evaluation, generated []byte) error {
	if outputDir == "" {
		return fmt.Errorf("output directory is required")
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "generated"), 0o755); err != nil {
		return err
	}
	proposal, err := CanonicalJSON(evaluation.Proposal)
	if err != nil {
		return err
	}
	receipt, err := CanonicalJSON(evaluation.Receipt)
	if err != nil {
		return err
	}
	semanticIR, err := CanonicalJSON(ir)
	if err != nil {
		return err
	}
	dossier := []byte(RenderDossier(ir, evaluation, evaluation.Proposal.Metrics))
	files := map[string][]byte{
		"bootstrap-proposal.json": proposal,
		"bootstrap-receipt.json": receipt,
		"semantic-ir.json":       semanticIR,
		"generated/evaluator.go": generated,
		"human-dossier.md":       dossier,
	}
	paths := []string{"bootstrap-proposal.json", "bootstrap-receipt.json", "semantic-ir.json", "generated/evaluator.go", "human-dossier.md"}
	manifest := ArtifactManifest{SchemaVersion: "gooo-artifact-manifest/v1", Files: make([]ArtifactFile, 0, len(paths))}
	for _, relative := range paths {
		data := files[relative]
		path := filepath.Join(outputDir, relative)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
		manifest.Files = append(manifest.Files, ArtifactFile{
			Path:          relative,
			Bytes:         len(data),
			PhysicalLines: physicalLines(data),
			Digest:        DigestBytes(data),
		})
	}
	manifestJSON, err := CanonicalJSON(manifest)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outputDir, "manifest.json"), manifestJSON, 0o644)
}

func physicalLines(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	lines := 0
	for _, byteValue := range data {
		if byteValue == '\n' {
			lines++
		}
	}
	if data[len(data)-1] != '\n' {
		lines++
	}
	return lines
}

func RenderDossier(ir SemanticIR, evaluation Evaluation, metrics Metrics) string {
	proposal := evaluation.Proposal
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Gooo bootstrap dossier\n\n")
	fmt.Fprintf(&builder, "Decision: `%s`\n\n", proposal.Decision)
	fmt.Fprintf(&builder, "Choice: `%s`\n\n", proposal.Choice)
	fmt.Fprintf(&builder, "Repository: `%s`\n\n", proposal.Repository)
	fmt.Fprintf(&builder, "Exact tuple: `%s` base `%s` head `%s` PR `%d`\n\n", proposal.Tuple.Repository, proposal.Tuple.Base, proposal.Tuple.Head, proposal.Tuple.PR)
	fmt.Fprintf(&builder, "The decision order is `REFUTED > UNKNOWN > CLOSED`. A green check or cache hit is not semantic closure. No score, percentage, or inferred priority is produced.\n\n")
	fmt.Fprintf(&builder, "## Authority choice\n\n")
	switch proposal.Choice {
	case ChoiceFoundation:
		builder.WriteString("FOUNDATION requires a pre-declared out-of-band or threshold authority bound before the candidate.\n\n")
	case ChoiceCoherence:
		builder.WriteString("COHERENCE checks consistency among independent authorities; it cannot create authority.\n\n")
	case ChoiceRegression:
		builder.WriteString("REGRESSION may preserve a previously valid exact baseline; it cannot authorize a new identity or head.\n\n")
	}
	fmt.Fprintf(&builder, "## Required external authority\n\n")
	if len(proposal.RequiredExternalAuthority) == 0 {
		builder.WriteString("None beyond the bound evidence.\n\n")
	}
	for _, requirement := range proposal.RequiredExternalAuthority {
		fmt.Fprintf(&builder, "- role `%s`: %s; binding `%s`\n", requirement.Role, requirement.Need, requirement.BoundBy)
	}
	if proposal.Decision == StatusUnknown {
		fmt.Fprintf(&builder, "\nMinimal blocked_by frontier: `%s`\n\n", strings.Join(proposal.BlockedBy, "`, `"))
	}
	if len(proposal.Refutations) > 0 {
		builder.WriteString("## Refutations\n\n")
		for _, refutation := range proposal.Refutations {
			fmt.Fprintf(&builder, "- `%s`: %s\n", refutation.Code, refutation.Evidence)
		}
		builder.WriteString("\n")
	}
	if len(proposal.PartialObservations) > 0 {
		builder.WriteString("## Partial observations\n\n")
		for _, observation := range proposal.PartialObservations {
			fmt.Fprintf(&builder, "- `%s`: observed %d of %d; valid as partial observation: `%t`; complete authority refuted: `%t`\n", observation.ArtifactID, observation.ObservedCount, observation.ExpectedCount, observation.ValidAsPartialObservation, observation.CompleteAuthorityRefuted)
		}
		builder.WriteString("\n")
	}
	if len(proposal.ContinuableActivities) > 0 {
		builder.WriteString("## Unrelated improvement work\n\n")
		builder.WriteString("These independently bound activities remain continuable while the core promotion route is blocked:\n\n")
		for _, activity := range proposal.ContinuableActivities {
			fmt.Fprintf(&builder, "- `%s`: %s\n", activity.ID, activity.Description)
		}
		builder.WriteString("\n")
	}
	fmt.Fprintf(&builder, "## Denominators and activity accounting\n\n")
	fmt.Fprintf(&builder, "- cells: `%d`\n- activities: `%d`\n- one-to-one bindings: `%d`\n- executed: `%d`\n- reused: `%d`\n- skipped: `%d`\n- not-observed: `%d`\n\n", metrics.CellDenominator, metrics.ActivityDenominator, metrics.OneToOneBindings, metrics.Executed, metrics.Reused, metrics.Skipped, metrics.NotObserved)
	fmt.Fprintf(&builder, "Inventory: `%d` files, `%d` directories, `%d` Go physical lines, `%d` Gooo physical lines.\n\n", metrics.Files, metrics.Directories, metrics.GoPhysicalLines, metrics.GoooPhysicalLines)
	fmt.Fprintf(&builder, "Build wall: `%s`; test wall: `%s`; peak RSS: `%s`.\n\n", metrics.BuildWall, metrics.TestWall, metrics.PeakRSS)
	fmt.Fprintf(&builder, "Product repository_writes: `%d`; local_test_executions: `%d`; cross_project_required_gates: `%d`.\n\n", metrics.ProductRepositoryWrites, metrics.LocalTestExecutions, metrics.CrossProjectRequiredGates)
	fmt.Fprintf(&builder, "Source digest: `%s`; semantic IR digest: `%s`; emitted receipt digest: `%s`.\n", ir.SourceDigest, ir.IRDigest, evaluation.Receipt.Digest)
	return builder.String()
}

func JSONIndent(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
