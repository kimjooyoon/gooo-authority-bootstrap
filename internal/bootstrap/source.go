package bootstrap

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
)

type Source struct {
	Input        CeremonyInput `json:"input"`
	SourceDigest string        `json:"source_digest"`
}

var sourceKeys = []string{
	"choice",
	"repository",
	"tuple",
	"candidate",
	"protected_paths",
	"current_ownership_policy_digest",
	"immutable_evaluator_release_digest",
	"branch_protection_snapshot",
	"credential_authority",
	"proposed_authority_delta",
	"authorities",
	"signer_identities",
	"approver_identities",
	"timestamps",
	"observed_artifacts",
	"predeclared_authority",
}

func ParseSource(data []byte) (Source, error) {
	var source Source
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	lineNumber := 0
	first := true
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if first {
			if line != "gooo 1" {
				return source, fmt.Errorf("line %d: expected gooo 1", lineNumber)
			}
			first = false
			continue
		}
		key, raw, ok := strings.Cut(line, " ")
		if !ok || raw == "" {
			return source, fmt.Errorf("line %d: expected key and JSON value", lineNumber)
		}
		if !containsString(sourceKeys, key) {
			return source, fmt.Errorf("line %d: unknown key %q", lineNumber, key)
		}
		if seen[key] {
			return source, fmt.Errorf("line %d: duplicate key %q", lineNumber, key)
		}
		seen[key] = true
		if err := decodeField(key, raw, &source.Input); err != nil {
			return source, fmt.Errorf("line %d: %w", lineNumber, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return source, err
	}
	if first {
		return source, fmt.Errorf("missing gooo header")
	}
	for _, key := range sourceKeys {
		if key == "predeclared_authority" {
			continue
		}
		if !seen[key] {
			return source, fmt.Errorf("missing required key %q", key)
		}
	}
	source.SourceDigest = DigestBytes(data)
	return source, nil
}

func decodeField(key, raw string, input *CeremonyInput) error {
	decode := func(destination any) error {
		if err := json.Unmarshal([]byte(raw), destination); err != nil {
			return fmt.Errorf("field %q: invalid JSON value: %w", key, err)
		}
		return nil
	}
	switch key {
	case "choice":
		return decode(&input.Choice)
	case "repository":
		return decode(&input.Repository)
	case "tuple":
		return decode(&input.Tuple)
	case "candidate":
		return decode(&input.Candidate)
	case "protected_paths":
		return decode(&input.ProtectedPaths)
	case "current_ownership_policy_digest":
		return decode(&input.CurrentOwnershipPolicyDigest)
	case "immutable_evaluator_release_digest":
		return decode(&input.ImmutableEvaluatorReleaseDigest)
	case "branch_protection_snapshot":
		return decode(&input.BranchProtection)
	case "credential_authority":
		return decode(&input.CredentialAuthority)
	case "proposed_authority_delta":
		return decode(&input.ProposedAuthorityDelta)
	case "authorities":
		return decode(&input.Authorities)
	case "signer_identities":
		return decode(&input.SignerIdentities)
	case "approver_identities":
		return decode(&input.ApproverIdentities)
	case "timestamps":
		return decode(&input.Timestamps)
	case "observed_artifacts":
		return decode(&input.ObservedArtifacts)
	case "predeclared_authority":
		if stringTrimmed := strings.TrimSpace(raw); stringTrimmed == "null" {
			input.PredeclaredAuthority = nil
			return nil
		}
		var authority PredeclaredAuthority
		if err := decode(&authority); err != nil {
			return err
		}
		input.PredeclaredAuthority = &authority
		return nil
	default:
		return fmt.Errorf("unsupported field %q", key)
	}
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
