package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type Status string

const (
	StatusClosed  Status = "CLOSED"
	StatusUnknown Status = "UNKNOWN"
	StatusRefuted Status = "REFUTED"
)

func statusRank(status Status) int {
	switch status {
	case StatusClosed:
		return 1
	case StatusUnknown:
		return 2
	case StatusRefuted:
		return 3
	default:
		return 0
	}
}

type Choice string

const (
	ChoiceFoundation Choice = "FOUNDATION"
	ChoiceCoherence  Choice = "COHERENCE"
	ChoiceRegression Choice = "REGRESSION"
)

// Tuple is the authority identity for a repository change. Branch names are
// deliberately not part of it: matching a branch cannot make a stale head
// reusable.
type Tuple struct {
	Repository string `json:"repository"`
	Base       string `json:"base"`
	Head       string `json:"head"`
	PR         int    `json:"pr"`
}

func (t Tuple) Equal(other Tuple) bool {
	return t.Repository == other.Repository &&
		t.Base == other.Base &&
		t.Head == other.Head &&
		t.PR == other.PR
}

type Candidate struct {
	ID         string `json:"id"`
	Branch     string `json:"branch"`
	EditDigest string `json:"edit_digest"`
}

type IdentityRef struct {
	ID         string `json:"id"`
	Role       string `json:"role"`
	ArtifactID string `json:"artifact_id"`
	Digest     string `json:"digest"`
}

func (i IdentityRef) Present() bool {
	return i.ID != "" || i.ArtifactID != ""
}

type AuthoritySet struct {
	Policy        IdentityRef `json:"policy"`
	Observation   IdentityRef `json:"observation"`
	Execution     IdentityRef `json:"execution"`
	HumanDecision IdentityRef `json:"human_decision"`
}

type CredentialAuthority struct {
	Identity         IdentityRef `json:"identity"`
	GuardianAppID    string      `json:"guardian_app_id"`
	ClientID         string      `json:"guardian_app_client_id"`
	PrivateKeyDigest string      `json:"guardian_app_private_key"`
}

type AuthorityDelta struct {
	ProposerID                 string   `json:"proposer_id"`
	ProposerBranch             string   `json:"proposer_branch"`
	ProposerEditDigest         string   `json:"proposer_edit_digest"`
	ChangesRoute               bool     `json:"changes_route"`
	RegistersOwnership         bool     `json:"registers_ownership"`
	ChangesProtection          bool     `json:"changes_protection"`
	ChangesCredential          bool     `json:"changes_credential"`
	NewIdentity                string   `json:"new_identity"`
	NewHead                    string   `json:"new_head"`
	Touches                    []string `json:"touches"`
	ProposedOwnershipMapDigest string   `json:"proposed_ownership_map_digest"`
	ProposedProtectionDigest   string   `json:"proposed_protection_digest"`
	ProposedCredentialIdentity string   `json:"proposed_credential_identity"`
}

type PredeclaredAuthority struct {
	Kind                 string   `json:"kind"`
	Identity             string   `json:"identity"`
	ArtifactID           string   `json:"artifact_id"`
	BindingDigest        string   `json:"binding_digest"`
	BoundBeforeCandidate bool     `json:"bound_before_candidate"`
	BoundAt              string   `json:"bound_at"`
	Threshold            int      `json:"threshold"`
	Members              []string `json:"members"`
}

type Timestamps struct {
	CandidateStartedAt string `json:"candidate_started_at"`
	ObservedAt         string `json:"observed_at"`
	SignedAt           string `json:"signed_at"`
	ApprovedAt         string `json:"approved_at"`
	IssuedAt           string `json:"issued_at"`
}

type RequiredStatusChecks struct {
	Contexts []string `json:"contexts"`
}

type BranchProtectionSnapshot struct {
	Source               string                `json:"source"`
	Digest               string                `json:"digest"`
	RequiredStatusChecks *RequiredStatusChecks `json:"required_status_checks"`
	Complete             bool                  `json:"complete"`
	ObservedBy           IdentityRef           `json:"observed_by"`
	ArtifactID           string                `json:"artifact_id"`
}

type PathObservation struct {
	Tuple         Tuple       `json:"tuple"`
	Authority     IdentityRef `json:"authority"`
	ArtifactID    string      `json:"artifact_id"`
	Complete      bool        `json:"complete"`
	ExpectedCount int         `json:"expected_count"`
	ObservedCount int         `json:"observed_count"`
	Paths         []string    `json:"paths"`
}

type HumanDecision struct {
	Identity   IdentityRef `json:"identity"`
	Decision   string      `json:"decision"`
	Explicit   bool        `json:"explicit"`
	At         string      `json:"at"`
	ArtifactID string      `json:"artifact_id"`
	Digest     string      `json:"digest"`
}

type ReplayEvidence struct {
	Equal         bool   `json:"equal"`
	Tuple         Tuple  `json:"tuple"`
	InputDigest   string `json:"input_digest"`
	ReceiptDigest string `json:"receipt_digest"`
	ArtifactID    string `json:"artifact_id"`
}

type ImprovementActivity struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Touches     []string `json:"touches"`
	Independent bool     `json:"independent"`
}

type Receipt struct {
	ReceiptID              string       `json:"receipt_id"`
	Decision               Status       `json:"decision"`
	Tuple                  Tuple        `json:"tuple"`
	Branch                 string       `json:"branch"`
	Choice                 Choice       `json:"choice"`
	EvaluatorReleaseDigest string       `json:"evaluator_release_digest"`
	EvaluatorIdentity      IdentityRef  `json:"evaluator_identity"`
	Immutable              bool         `json:"immutable"`
	ReplayDigest           string       `json:"replay_digest"`
	AuthoritySet           AuthoritySet `json:"authority_set"`
	ApprovedBy             IdentityRef  `json:"approved_by"`
	IssuedAt               string       `json:"issued_at"`
	Digest                 string       `json:"digest"`
}

type ObservedArtifacts struct {
	Protection              BranchProtectionSnapshot `json:"protection"`
	Paths                   PathObservation          `json:"paths"`
	PriorReceipt            *Receipt                 `json:"prior_receipt"`
	EvaluatorIdentity       IdentityRef              `json:"evaluator_identity"`
	HumanDecision           *HumanDecision           `json:"human_decision"`
	Replay                  ReplayEvidence           `json:"replay"`
	IndependentImprovements []ImprovementActivity    `json:"independent_improvements"`
	GreenCheckObserved      bool                     `json:"green_check_observed"`
}

type CeremonyInput struct {
	Choice                          Choice                   `json:"choice"`
	Repository                      string                   `json:"repository"`
	Tuple                           Tuple                    `json:"tuple"`
	Candidate                       Candidate                `json:"candidate"`
	ProtectedPaths                  []string                 `json:"protected_paths"`
	CurrentOwnershipPolicyDigest    string                   `json:"current_ownership_policy_digest"`
	ImmutableEvaluatorReleaseDigest string                   `json:"immutable_evaluator_release_digest"`
	BranchProtection                BranchProtectionSnapshot `json:"branch_protection_snapshot"`
	CredentialAuthority             CredentialAuthority      `json:"credential_authority"`
	ProposedAuthorityDelta          AuthorityDelta           `json:"proposed_authority_delta"`
	Authorities                     AuthoritySet             `json:"authorities"`
	SignerIdentities                []IdentityRef            `json:"signer_identities"`
	ApproverIdentities              []IdentityRef            `json:"approver_identities"`
	Timestamps                      Timestamps               `json:"timestamps"`
	ObservedArtifacts               ObservedArtifacts        `json:"observed_artifacts"`
	PredeclaredAuthority            *PredeclaredAuthority    `json:"predeclared_authority"`
}

type CellActivityBinding struct {
	Cell     string `json:"cell"`
	Activity string `json:"activity"`
	Source   string `json:"source"`
}

type Denominators struct {
	CellCount     int                   `json:"cell_count"`
	ActivityCount int                   `json:"activity_count"`
	OneToOne      bool                  `json:"one_to_one"`
	Bindings      []CellActivityBinding `json:"bindings"`
}

type SemanticIR struct {
	Version      string        `json:"version"`
	SourceDigest string        `json:"source_digest"`
	Input        CeremonyInput `json:"input"`
	Denominators Denominators  `json:"denominators"`
	IRDigest     string        `json:"ir_digest"`
}

type Refutation struct {
	Code     string `json:"code"`
	Evidence string `json:"evidence"`
}

type PartialObservation struct {
	ArtifactID                string `json:"artifact_id"`
	ExpectedCount             int    `json:"expected_count"`
	ObservedCount             int    `json:"observed_count"`
	ValidAsPartialObservation bool   `json:"valid_as_partial_observation"`
	CompleteAuthorityRefuted  bool   `json:"complete_authority_refuted"`
}

type AuthorityRequirement struct {
	Role    string `json:"role"`
	Need    string `json:"need"`
	BoundBy string `json:"bound_by"`
}

type UnknownFrontier struct {
	CredentialAuthorityIdentity string `json:"credential_authority_identity"`
	GuardianAppClientID         string `json:"guardian_app_client_id"`
	GuardianAppPrivateKey       string `json:"guardian_app_private_key"`
	RequiredStatusChecks        string `json:"required_status_checks"`
	IndependentObservation      string `json:"independent_observation"`
	HumanDecision               string `json:"human_decision"`
}

type Metrics struct {
	Files                     int    `json:"files"`
	Directories               int    `json:"directories"`
	GoPhysicalLines           int    `json:"go_physical_lines"`
	GoooPhysicalLines         int    `json:"gooo_physical_lines"`
	BuildWall                 string `json:"build_wall"`
	TestWall                  string `json:"test_wall"`
	PeakRSS                   string `json:"peak_rss"`
	Executed                  int    `json:"executed"`
	Reused                    int    `json:"reused"`
	Skipped                   int    `json:"skipped"`
	NotObserved               int    `json:"not_observed"`
	CellDenominator           int    `json:"cell_denominator"`
	ActivityDenominator       int    `json:"activity_denominator"`
	OneToOneBindings          int    `json:"one_to_one_bindings"`
	ProductRepositoryWrites   int    `json:"repository_writes"`
	LocalTestExecutions       int    `json:"local_test_executions"`
	CrossProjectRequiredGates int    `json:"cross_project_required_gates"`
}

type Proposal struct {
	SchemaVersion                string                 `json:"schema_version"`
	Decision                     Status                 `json:"decision"`
	Choice                       Choice                 `json:"choice"`
	Repository                   string                 `json:"repository"`
	Tuple                        Tuple                  `json:"tuple"`
	ProtectedPaths               []string               `json:"protected_paths"`
	CurrentOwnershipPolicyDigest string                 `json:"current_ownership_policy_digest"`
	EvaluatorReleaseDigest       string                 `json:"evaluator_release_digest"`
	ProposedAuthorityDelta       AuthorityDelta         `json:"proposed_authority_delta"`
	RequiredExternalAuthority    []AuthorityRequirement `json:"required_external_authority"`
	BlockedBy                    []string               `json:"blocked_by"`
	UnknownFrontier              *UnknownFrontier       `json:"unknown_frontier"`
	Refutations                  []Refutation           `json:"refutations"`
	PartialObservations          []PartialObservation   `json:"partial_observations"`
	ContinuableActivities        []ImprovementActivity  `json:"continuable_activities"`
	Metrics                      Metrics                `json:"metrics"`
}

type Evaluation struct {
	Proposal Proposal `json:"proposal"`
	Receipt  Receipt  `json:"receipt"`
}

func DigestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func CanonicalJSON(value any) ([]byte, error) {
	return json.Marshal(value)
}

func SortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func digestValue(value any) (string, error) {
	data, err := CanonicalJSON(value)
	if err != nil {
		return "", err
	}
	return DigestBytes(data), nil
}

func validateDigest(name, value string) error {
	if value == "" || !strings.HasPrefix(value, "sha256:") {
		return fmt.Errorf("%s must be a non-empty sha256 digest", name)
	}
	return nil
}
