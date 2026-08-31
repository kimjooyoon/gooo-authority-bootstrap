package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

type corpusExpectation struct {
	name            string
	want            Status
	blockedBy       []string
	wantPartial     bool
	wantContinuable int
}

var exactCorpus = []corpusExpectation{
	{name: "01-valid-foundation", want: StatusClosed},
	{name: "02-missing-guardian-credentials", want: StatusUnknown, blockedBy: []string{"guardian_app_client_id", "guardian_app_private_key"}},
	{name: "03-null-required-status-checks", want: StatusUnknown, blockedBy: []string{"required_status_checks"}},
	{name: "04-stale-609-receipt", want: StatusRefuted},
	{name: "05-candidate-self-ownership", want: StatusRefuted},
	{name: "06-rest-truncated-path-list", want: StatusRefuted, wantPartial: true},
	{name: "07-independent-improvements", want: StatusUnknown, blockedBy: []string{"independent_observation"}, wantContinuable: 2},
	{name: "08-exact-replay-human-decision", want: StatusClosed},
}

func TestConformanceCorpus(t *testing.T) {
	if len(exactCorpus) != 8 {
		t.Fatalf("case corpus size = %d, want 8", len(exactCorpus))
	}
	counts := map[Status]int{}
	for _, expected := range exactCorpus {
		path := corpusPath(expected.name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		source, err := ParseSource(data)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ir, err := Lower(source)
		if err != nil {
			t.Fatalf("lower %s: %v", path, err)
		}
		metrics, err := BuildMetrics("", ir)
		if err != nil {
			t.Fatalf("metrics %s: %v", path, err)
		}
		result, err := Evaluate(ir, metrics)
		if err != nil {
			t.Fatalf("evaluate %s: %v", path, err)
		}
		counts[result.Proposal.Decision]++
		if result.Proposal.Decision != expected.want {
			t.Fatalf("%s decision = %s, want %s", expected.name, result.Proposal.Decision, expected.want)
		}
		if !reflect.DeepEqual(result.Proposal.BlockedBy, expected.blockedBy) {
			t.Fatalf("%s blocked_by = %#v, want %#v", expected.name, result.Proposal.BlockedBy, expected.blockedBy)
		}
		if (len(result.Proposal.PartialObservations) > 0) != expected.wantPartial {
			t.Fatalf("%s partial observation presence mismatch", expected.name)
		}
		if len(result.Proposal.ContinuableActivities) != expected.wantContinuable {
			t.Fatalf("%s continuable activities = %d, want %d", expected.name, len(result.Proposal.ContinuableActivities), expected.wantContinuable)
		}
		if result.Proposal.Metrics.CellDenominator != result.Proposal.Metrics.ActivityDenominator {
			t.Fatalf("%s cell/activity denominator mismatch", expected.name)
		}
		if result.Proposal.Metrics.OneToOneBindings != result.Proposal.Metrics.CellDenominator {
			t.Fatalf("%s one-to-one binding mismatch", expected.name)
		}
	}
	wantCounts := map[Status]int{StatusClosed: 2, StatusUnknown: 3, StatusRefuted: 3}
	if !reflect.DeepEqual(counts, wantCounts) {
		t.Fatalf("status counts = %#v, want %#v", counts, wantCounts)
	}
}

func TestDeterministicReplay(t *testing.T) {
	data, err := os.ReadFile(corpusPath("08-exact-replay-human-decision"))
	if err != nil {
		t.Fatal(err)
	}
	source, err := ParseSource(data)
	if err != nil {
		t.Fatal(err)
	}
	ir, err := Lower(source)
	if err != nil {
		t.Fatal(err)
	}
	metrics, err := BuildMetrics("", ir)
	if err != nil {
		t.Fatal(err)
	}
	first, err := Evaluate(ir, metrics)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Evaluate(ir, metrics)
	if err != nil {
		t.Fatal(err)
	}
	if first.Receipt.Digest != second.Receipt.Digest {
		t.Fatalf("receipt digest changed on replay: %s != %s", first.Receipt.Digest, second.Receipt.Digest)
	}
	firstProposal, _ := CanonicalJSON(first.Proposal)
	secondProposal, _ := CanonicalJSON(second.Proposal)
	if string(firstProposal) != string(secondProposal) {
		t.Fatal("proposal changed on replay")
	}
	firstGenerated, err := GenerateEvaluator(ir)
	if err != nil {
		t.Fatal(err)
	}
	secondGenerated, err := GenerateEvaluator(ir)
	if err != nil {
		t.Fatal(err)
	}
	if string(firstGenerated) != string(secondGenerated) {
		t.Fatal("generated evaluator changed on replay")
	}
}

func TestArtifactAudit(t *testing.T) {
	data, err := os.ReadFile(corpusPath("01-valid-foundation"))
	if err != nil {
		t.Fatal(err)
	}
	source, err := ParseSource(data)
	if err != nil {
		t.Fatal(err)
	}
	ir, err := Lower(source)
	if err != nil {
		t.Fatal(err)
	}
	metrics, err := BuildMetrics("", ir)
	if err != nil {
		t.Fatal(err)
	}
	evaluation, err := Evaluate(ir, metrics)
	if err != nil {
		t.Fatal(err)
	}
	generated, err := GenerateEvaluator(ir)
	if err != nil {
		t.Fatal(err)
	}
	output := t.TempDir()
	if err := WriteArtifacts(output, ir, evaluation, generated); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 6 {
		t.Fatalf("top-level artifact entries = %d, want 6", len(entries))
	}
	manifestData, err := os.ReadFile(filepath.Join(output, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest ArtifactManifest
	if err := jsonUnmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Files) != 5 {
		t.Fatalf("manifest files = %d, want 5", len(manifest.Files))
	}
	paths := make([]string, 0, len(manifest.Files))
	for _, file := range manifest.Files {
		paths = append(paths, file.Path)
		if file.Digest == "" || file.Bytes <= 0 || file.PhysicalLines <= 0 {
			t.Fatalf("incomplete manifest entry: %#v", file)
		}
	}
	sort.Strings(paths)
	wantPaths := []string{"bootstrap-proposal.json", "bootstrap-receipt.json", "generated/evaluator.go", "human-dossier.md", "semantic-ir.json"}
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("manifest paths = %#v, want %#v", paths, wantPaths)
	}
}

func TestStatusOrdering(t *testing.T) {
	if statusRank(StatusRefuted) <= statusRank(StatusUnknown) || statusRank(StatusUnknown) <= statusRank(StatusClosed) {
		t.Fatal("status ordering must be REFUTED > UNKNOWN > CLOSED")
	}
}

func jsonUnmarshal(data []byte, target any) error {
	return json.Unmarshal(data, target)
}

func corpusPath(name string) string {
	return filepath.Join("..", "..", "testdata", "corpus", name+".gooo")
}
