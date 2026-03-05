package main

// This file sets up the Ginkgo test suite for the history-worker.
// It configures the test runner and imports necessary testing packages.

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHistoryWorker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "History-Worker Suite")
}
