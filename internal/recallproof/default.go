package recallproof

import (
	"sync"

	"github.com/chuma-beep/mathua/internal/proof"
	"github.com/chuma-beep/mathua/internal/pytrack"
)

var (
	defaultOnce sync.Once
	defaultG    *Grader
	defaultErr  error
)

// Default is the process-wide written-answer grader on the process-wide
// proof grader (proof.Default), configured from MATHUA_LLM_RECALL*.
func Default() (*Grader, error) {
	defaultOnce.Do(func() {
		p, err := proof.Default()
		if err != nil {
			defaultErr = err
			return
		}
		defaultG = New(p, ConfigFromEnv(p.Cfg.APIKey != ""), nil)
	})
	return defaultG, defaultErr
}

// Install makes the process-wide grader pytrack's written-answer grader, so
// written-answer flashcards are served with the answer box and graded. Call
// it after the proof grader has its database store (proof.Grader.SetStore).
// With grading off it installs nothing and every card stays self-graded.
func Install() (*Grader, error) {
	g, err := Default()
	if err != nil {
		return nil, err
	}
	if g.On() {
		pytrack.SetWrittenGrader(g)
	}
	return g, nil
}
