package main

// This file sets up the Ginkgo test suite for the web-server.
// It configures the test runner and imports necessary testing packages.

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestWebServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Web-Server Suite")
}
