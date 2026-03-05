package main

// Unit tests for HealthHandler - tests health check endpoint
// Covers basic functionality and verifies no authentication is required

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HealthHandler", func() {
	var (
		server     *Server
		mockDB     *sql.DB
		recorder   *httptest.ResponseRecorder
		testSecret = "test-secret-key-at-least-32-characters-long-for-jwt"
	)

	BeforeEach(func() {
		os.Setenv("JWT_SECRET", testSecret)
		InitJWTSecret()

		var err error
		mockDB, _, err = sqlmock.New()
		Expect(err).ShouldNot(HaveOccurred())

		server = &Server{
			DB:   mockDB,
			NATS: nil,
		}

		recorder = httptest.NewRecorder()
	})

	AfterEach(func() {
		mockDB.Close()
		os.Unsetenv("JWT_SECRET")
		jwtSecret = nil
	})

	When("testing health endpoint", func() {
		It("should return 200 OK with healthy status", func() {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)

			server.HealthHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusOK))

			var response map[string]string
			Expect(json.Unmarshal(recorder.Body.Bytes(), &response)).Should(Succeed())
			Expect(response["status"]).Should(Equal("healthy"))
		})

		It("should not require authentication", func() {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)

			server.HealthHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusOK))
		})

		It("should return application/json content type", func() {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)

			server.HealthHandler(recorder, req)

			Expect(recorder.Header().Get("Content-Type")).Should(Equal("application/json"))
		})
	})
})
