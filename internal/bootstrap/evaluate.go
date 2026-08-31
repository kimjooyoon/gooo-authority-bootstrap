package bootstrap

import (
	"fmt"
	"sort"
	"time"
)

var unknownFieldNames = []string{
	"credential_authority_identity",
	"guardian_app_client_id",
	"guardian_app_private_key",
	"required_status_checks",
	"independent_observation",
	"human_decision",
}

func Evaluate(ir SemanticIR, metrics Metrics) (Evaluation, error) {
	input := ir.Input
	refutations := make([]Refutation, 0)
	partial := make([]PartialObservation, 0)

	if input.ObservedArtifacts.Paths.ObservedCount < input.ObservedArtifacts.Paths.ExpectedCount {
		partial = append(partial, PartialObservation{
			ArtifactID:               input.ObservedArtifacts.Paths.ArtifactID,
			ExpectedCount:             input.ObservedArtifacts.Paths.ExpectedCount,
			ObservedCount:             input.ObservedArtifacts.Paths.ObservedCount,
			ValidAsPartialObservation: true,
			CompleteAuthorityRefuted:  input.ObservedArtifacts.Paths.Complete,
		})
		if input.ObservedArtifacts.Paths.Complete {
			refutations = append(refutations, Refutation{
				Code:     "truncated_complete_path_authority",
				Evidence: fmt.Sprintf("artifact %s claims complete authority with %d of %d paths", input.ObservedArtifacts.Paths.ArtifactID, input.ObservedArtifacts.Paths.ObservedCount, input.ObservedArtifacts.Paths.ExpectedCount),
			})
		}
	}

	if input.ObservedArtifacts.PriorReceipt != nil {
		prior := input.ObservedArtifacts.PriorReceipt
		if !prior.Tuple.Equal(input.Tuple) {
			refutations = append(refutations, Refutation{
				Code:     "receipt_tuple_mismatch",
				Evidence: "receipt base/head/PR/repository tuple differs; branch-name equality is not reusable authority",
			})
		}
		if prior.Branch != "" && prior.Branch != input.Candidate.Branch {
			refutations = append(refutations, Refutation{
				Code:     "receipt_branch_mismatch",
				Evidence: "receipt branch differs from candidate branch and is not a substitute for exact tuple equality",
			})
		}
		if prior.EvaluatorReleaseDigest != input.ImmutableEvaluatorReleaseDigest {
			refutations = append(refutations, Refutation{
				Code:     "evaluator_release_mismatch",
				Evidence: "observed receipt is not bound to the immutable evaluator release digest",
			})
		}
	}

	if candidateSelfAuthorizes(input) {
		refutations = append(refutations, Refutation{
			Code:     "candidate_self_authority",
			Evidence: "candidate identity, branch, or edit proposes a route, ownership, protection, or credential authority change",
		})
	}
	if authorityRoleCollision(input) {
		refutations = append(refutations, Refutation{
			Code:     "authority_role_collision",
			Evidence: "one actor or artifact is bound to more than one authority role",
		})
	}
	if input.Choice == ChoiceRegression && regressionAuthorizesNewIdentityOrHead(input) {
		refutations = append(refutations, Refutation{
			Code:     "regression_new_identity_or_head",
			Evidence: "REGRESSION may preserve an exact valid baseline but cannot authorize a new identity or head",
		})
	}

	blockedBy := minimalBlockedBy(ir)
	unknownFrontier := frontierFor(blockedBy)
	if len(refutations) == 0 && len(blockedBy) == 0 && !choiceAuthorityReady(input) {
		blockedBy = []string{"credential_authority_identity"}
		unknownFrontier = frontierFor(blockedBy)
	}
	if len(refutations) > 0 {
		blockedBy = nil
		unknownFrontier = nil
	}

	status := StatusClosed
	if len(refutations) > 0 {
		status = StatusRefuted
	} else if len(blockedBy) > 0 {
		status = StatusUnknown
	}

	requirements := requiredAuthority(input, status, blockedBy)
	proposal := Proposal{
		SchemaVersion:                 "gooo-proposal/v1",
		Decision:                      status,
		Choice:                        input.Choice,
		Repository:                    input.Repository,
		Tuple:                         input.Tuple,
		ProtectedPaths:                append([]string(nil), input.ProtectedPaths...),
		CurrentOwnershipPolicyDigest:  input.CurrentOwnershipPolicyDigest,
		EvaluatorReleaseDigest:        input.ImmutableEvaluatorReleaseDigest,
		ProposedAuthorityDelta:        input.ProposedAuthorityDelta,
		RequiredExternalAuthority:     requirements,
		BlockedBy:                     blockedBy,
		UnknownFrontier:               unknownFrontier,
		Refutations:                   refutations,
		PartialObservations:           partial,
		ContinuableActivities:         continuableActivities(input),
		Metrics:                       metrics,
	}

	receipt := Receipt{
		ReceiptID:              "bootstrap:" + ir.IRDigest,
		Decision:               status,
		Tuple:                  input.Tuple,
		Branch:                 input.Candidate.Branch,
		Choice:                 input.Choice,
		EvaluatorReleaseDigest: input.ImmutableEvaluatorReleaseDigest,
		EvaluatorIdentity:      input.ObservedArtifacts.EvaluatorIdentity,
		Immutable:              status == StatusClosed && input.ObservedArtifacts.PriorReceipt != nil && input.ObservedArtifacts.PriorReceipt.Immutable,
		ReplayDigest:           replayDigest(ir, status),
		AuthoritySet:           input.Authorities,
		IssuedAt:               input.Timestamps.IssuedAt,
	}
	if input.ObservedArtifacts.HumanDecision != nil {
		receipt.ApprovedBy = input.ObservedArtifacts.HumanDecision.Identity
	}
	receipt.Digest, _ = digestValue(struct {
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
	}{receipt.ReceiptID, receipt.Decision, receipt.Tuple, receipt.Branch, receipt.Choice, receipt.EvaluatorReleaseDigest, receipt.EvaluatorIdentity, receipt.Immutable, receipt.ReplayDigest, receipt.AuthoritySet, receipt.ApprovedBy, receipt.IssuedAt})
	return Evaluation{Proposal: proposal, Receipt: receipt}, nil
}

func minimalBlockedBy(ir SemanticIR) []string {
	input := ir.Input
	if !input.CredentialAuthority.Identity.Present() {
		return []string{"credential_authority_identity"}
	}
	missingCredentials := make([]string, 0, 2)
	if input.CredentialAuthority.ClientID == "" {
		missingCredentials = append(missingCredentials, "guardian_app_client_id")
	}
	if input.CredentialAuthority.PrivateKeyDigest == "" {
		missingCredentials = append(missingCredentials, "guardian_app_private_key")
	}
	if len(missingCredentials) > 0 {
		return missingCredentials
	}
	if input.BranchProtection.RequiredStatusChecks == nil {
		return []string{"required_status_checks"}
	}
	if !independentObservationReady(input) {
		return []string{"independent_observation"}
	}
	if !humanDecisionReady(input) {
		return []string{"human_decision"}
	}
	if input.ObservedArtifacts.PriorReceipt == nil || !input.ObservedArtifacts.PriorReceipt.Immutable {
		return []string{"independent_observation"}
	}
	if !replayReady(input) {
		return []string{"independent_observation"}
	}
	return nil
}

func frontierFor(blockedBy []string) *UnknownFrontier {
	if len(blockedBy) == 0 {
		return nil
	}
	frontier := &UnknownFrontier{}
	for _, field := range unknownFieldNames {
		value := "satisfied"
		for _, blocked := range blockedBy {
			if field == blocked {
				value = "required"
			}
		}
		switch field {
		case "credential_authority_identity":
			frontier.CredentialAuthorityIdentity = value
		case "guardian_app_client_id":
			frontier.GuardianAppClientID = value
		case "guardian_app_private_key":
			frontier.GuardianAppPrivateKey = value
		case "required_status_checks":
			frontier.RequiredStatusChecks = value
		case "independent_observation":
			frontier.IndependentObservation = value
		case "human_decision":
			frontier.HumanDecision = value
		}
	}
	return frontier
}

func choiceAuthorityReady(input CeremonyInput) bool {
	switch input.Choice {
	case ChoiceFoundation:
		foundation := input.PredeclaredAuthority
		if foundation == nil || !foundation.BoundBeforeCandidate || foundation.Identity == "" || foundation.BindingDigest == "" ||
			(foundation.Kind != "out-of-band" && foundation.Kind != "threshold") || foundation.Threshold <= 0 || len(foundation.Members) < foundation.Threshold {
			return false
		}
		boundAt, boundErr := time.Parse(time.RFC3339Nano, foundation.BoundAt)
		candidateAt, candidateErr := time.Parse(time.RFC3339Nano, input.Timestamps.CandidateStartedAt)
		return boundErr == nil && candidateErr == nil && boundAt.Before(candidateAt)
	case ChoiceCoherence:
		return input.Authorities.Policy.Present() && input.Authorities.Observation.Present() && input.Authorities.Execution.Present()
	case ChoiceRegression:
		return input.ObservedArtifacts.PriorReceipt != nil && input.ObservedArtifacts.PriorReceipt.Immutable
	default:
		return false
	}
}

func independentObservationReady(input CeremonyInput) bool {
	paths := input.ObservedArtifacts.Paths
	protection := input.ObservedArtifacts.Protection
	return paths.Complete && paths.Tuple.Equal(input.Tuple) && paths.ExpectedCount == paths.ObservedCount &&
		protection.Complete && protection.Digest == input.BranchProtection.Digest &&
		paths.Authority.Present() && protection.ObservedBy.Present() && paths.Authority.ID != input.Candidate.ID
}

func humanDecisionReady(input CeremonyInput) bool {
	decision := input.ObservedArtifacts.HumanDecision
	if decision == nil || !decision.Explicit || decision.Decision != "approve" || decision.Identity.ID == "" {
		return false
	}
	for _, approver := range input.ApproverIdentities {
		if approver.ID == decision.Identity.ID && approver.ArtifactID == decision.Identity.ArtifactID {
			return true
		}
	}
	return false
}

func replayReady(input CeremonyInput) bool {
	prior := input.ObservedArtifacts.PriorReceipt
	replay := input.ObservedArtifacts.Replay
	return prior != nil && prior.Immutable && prior.EvaluatorIdentity.Present() &&
		prior.EvaluatorReleaseDigest == input.ImmutableEvaluatorReleaseDigest &&
		replay.Equal && replay.Tuple.Equal(input.Tuple) && replay.ReceiptDigest == prior.Digest && replay.ArtifactID != ""
}

func candidateSelfAuthorizes(input CeremonyInput) bool {
	delta := input.ProposedAuthorityDelta
	self := delta.ProposerID == input.Candidate.ID || delta.ProposerBranch == input.Candidate.Branch || delta.ProposerEditDigest == input.Candidate.EditDigest
	changesAuthority := delta.ChangesRoute || delta.RegistersOwnership || delta.ChangesProtection || delta.ChangesCredential || delta.ProposedOwnershipMapDigest != "" || delta.ProposedProtectionDigest != "" || delta.ProposedCredentialIdentity != ""
	return self && changesAuthority
}

func authorityRoleCollision(input CeremonyInput) bool {
	type roleIdentity struct {
		role string
		ref  IdentityRef
	}
	roles := []roleIdentity{
		{"policy", input.Authorities.Policy},
		{"observation", input.Authorities.Observation},
		{"execution", input.Authorities.Execution},
		{"human_decision", input.Authorities.HumanDecision},
		{"credential_execution", input.CredentialAuthority.Identity},
		{"path_observation", input.ObservedArtifacts.Paths.Authority},
		{"protection_observation", input.ObservedArtifacts.Protection.ObservedBy},
		{"evaluator", input.ObservedArtifacts.EvaluatorIdentity},
	}
	for _, signer := range input.SignerIdentities {
		roles = append(roles, roleIdentity{"signer", signer})
	}
	for _, approver := range input.ApproverIdentities {
		roles = append(roles, roleIdentity{"approver", approver})
	}
	for i := range roles {
		if roles[i].ref.ID == "" && roles[i].ref.ArtifactID == "" {
			continue
		}
		for j := i + 1; j < len(roles); j++ {
			if canonicalRole(roles[i].role) == canonicalRole(roles[j].role) || (roles[j].ref.ID == "" && roles[j].ref.ArtifactID == "") {
				continue
			}
			if roles[i].ref.ID != "" && roles[i].ref.ID == roles[j].ref.ID {
				return true
			}
			if roles[i].ref.ArtifactID != "" && roles[i].ref.ArtifactID == roles[j].ref.ArtifactID {
				return true
			}
		}
	}
	return false
}

func canonicalRole(role string) string {
	switch role {
	case "credential_execution":
		return "execution"
	case "path_observation", "protection_observation":
		return "observation"
	case "approver":
		return "human_decision"
	default:
		return role
	}
}

func regressionAuthorizesNewIdentityOrHead(input CeremonyInput) bool {
	prior := input.ObservedArtifacts.PriorReceipt
	if prior == nil {
		return false
	}
	return (input.ProposedAuthorityDelta.NewIdentity != "" && input.ProposedAuthorityDelta.NewIdentity != prior.AuthoritySet.Policy.ID) ||
		(input.ProposedAuthorityDelta.NewHead != "" && input.ProposedAuthorityDelta.NewHead != prior.Tuple.Head)
}

func requiredAuthority(input CeremonyInput, status Status, blockedBy []string) []AuthorityRequirement {
	requirements := make([]AuthorityRequirement, 0)
	if input.Choice == ChoiceFoundation && status != StatusClosed {
		requirements = append(requirements, AuthorityRequirement{
			Role:    "foundation",
			Need:    "pre-declared out-of-band or threshold authority bound before candidate",
			BoundBy: "predeclared_authority.binding_digest",
		})
	}
	if status == StatusUnknown {
		for _, field := range blockedBy {
			requirements = append(requirements, AuthorityRequirement{
				Role:    field,
				Need:    "independent evidence for the named frontier field",
				BoundBy: "bootstrap proposal input",
			})
		}
	}
	if status == StatusClosed {
		requirements = append(requirements, AuthorityRequirement{
			Role:    "human_decision",
			Need:    "explicit approve decision bound to the exact tuple",
			BoundBy: "human_decision.digest",
		})
	}
	return requirements
}

func continuableActivities(input CeremonyInput) []ImprovementActivity {
	activities := append([]ImprovementActivity(nil), input.ObservedArtifacts.IndependentImprovements...)
	sort.Slice(activities, func(i, j int) bool { return activities[i].ID < activities[j].ID })
	result := activities[:0]
	for _, activity := range activities {
		if activity.Independent {
			result = append(result, activity)
		}
	}
	return result
}

func replayDigest(ir SemanticIR, status Status) string {
	value := struct {
		IRDigest string `json:"ir_digest"`
		Status   Status `json:"status"`
	}{ir.IRDigest, status}
	digest, _ := digestValue(value)
	return digest
}
