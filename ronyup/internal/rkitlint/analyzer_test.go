package rkitlint_test

import (
	"testing"

	"github.com/clubpay/ronykit/ronyup/internal/rkitlint"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), rkitlint.Analyzer,
		"must", "mapcollect", "unsafeconv", "helper", "ignore", "generated", rkitlint.RkitPath)
}
