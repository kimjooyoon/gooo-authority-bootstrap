package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-authority-bootstrap/internal/bootstrap"
)

func main() {
	sourcePath := flag.String("source", "", "caller-owned .gooo source path")
	outputDir := flag.String("output", "", "caller-owned output directory")
	inventoryRoot := flag.String("inventory-root", "", "optional read-only root for exact inventory metrics")
	flag.Parse()
	if *sourcePath == "" || *outputDir == "" {
		fail("-source and -output are required")
	}

	data, err := os.ReadFile(*sourcePath)
	if err != nil {
		fail("read source: %v", err)
	}
	source, err := bootstrap.ParseSource(data)
	if err != nil {
		fail("parse source: %v", err)
	}
	ir, err := bootstrap.Lower(source)
	if err != nil {
		fail("lower source: %v", err)
	}
	metrics, err := bootstrap.BuildMetrics(*inventoryRoot, ir)
	if err != nil {
		fail("inventory: %v", err)
	}
	evaluation, err := bootstrap.Evaluate(ir, metrics)
	if err != nil {
		fail("evaluate: %v", err)
	}
	generated, err := bootstrap.GenerateEvaluator(ir)
	if err != nil {
		fail("generate evaluator: %v", err)
	}
	if err := bootstrap.WriteArtifacts(filepath.Clean(*outputDir), ir, evaluation, generated); err != nil {
		fail("write caller-owned artifacts: %v", err)
	}
	fmt.Printf("decision=%s\nproposal=%s\nreceipt=%s\n", evaluation.Proposal.Decision, filepath.Join(*outputDir, "bootstrap-proposal.json"), filepath.Join(*outputDir, "bootstrap-receipt.json"))
}

func fail(format string, values ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", values...)
	os.Exit(1)
}
