package bootstrap

import (
	"fmt"
	"strconv"
)

func GenerateEvaluator(ir SemanticIR) ([]byte, error) {
	data, err := CanonicalJSON(ir)
	if err != nil {
		return nil, err
	}
	semanticJSON := strconv.Quote(string(data))
	generated := fmt.Sprintf("package generated\n\nconst SourceDigest = %q\n\nconst SemanticIRDigest = %q\n\nconst GeneratedEvaluatorRelease = %q\n\nconst SemanticIRJSON = %s\n", ir.SourceDigest, ir.IRDigest, ir.Input.ImmutableEvaluatorReleaseDigest, semanticJSON)
	return []byte(generated), nil
}
