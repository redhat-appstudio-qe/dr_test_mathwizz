package main

// Integration tests for HealthHandler - verifies health check endpoint with real dependencies
// These tests reveal Bug #22: health endpoint doesn't actually check database/NATS connectivity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var _ = Describe("Health Integration Tests", func() {
	var (
		ctx    context.Context
		dbCont testcontainers.Container
		server *Server
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Initialize JWT secret for tests (required by server setup)
		os.Setenv("JWT_SECRET", "test-integration-secret-key-at-least-32-characters-long")
		Expect(InitJWTSecret()).Should(Succeed())

		// Start PostgreSQL testcontainer
		req := testcontainers.ContainerRequest{
			Image:        "postgres:16-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_DB":       "mathwizz_test",
				"POSTGRES_USER":     "test",
				"POSTGRES_PASSWORD": "test",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60 * time.Second),
		}

		var err error
		dbCont, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		Expect(err).ShouldNot(HaveOccurred())

		host, err := dbCont.Host(ctx)
		Expect(err).ShouldNot(HaveOccurred())

		port, err := dbCont.MappedPort(ctx, "5432")
		Expect(err).ShouldNot(HaveOccurred())

		db, err := ConnectDB(host, "test", "test", "mathwizz_test", port.Int(), "disable")
		Expect(err).ShouldNot(HaveOccurred())

		// Create tables (health check should verify DB is reachable)
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE NOT NULL,
				password_hash VARCHAR(255) NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		Expect(err).ShouldNot(HaveOccurred())

		server = &Server{DB: db, NATS: nil}
	})

	AfterEach(func() {
		if server.DB != nil {
			server.DB.Close()
		}
		if dbCont != nil {
			Expect(dbCont.Terminate(ctx)).Should(Succeed())
		}
		os.Unsetenv("JWT_SECRET")
		jwtSecret = nil
	})

	When("all dependencies are healthy", func() {
		It("should return 200 OK when database is connected", func() {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			server.HealthHandler(w, req)

			Expect(w.Code).Should(Equal(http.StatusOK))

			var response map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(response["status"]).Should(Equal("healthy"))
		})
	})

	When("testing dependency health checks (revealing Bug #22)", func() {
		It("should return 503 Service Unavailable when database connection is closed", func() {
			// Close the database connection to simulate unhealthy state
			Expect(server.DB.Close()).Should(Succeed())

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			server.HealthHandler(w, req)

			// EXPECTED BEHAVIOR (when Bug #22 is fixed):
			// Health endpoint should verify database connectivity by running a ping query
			// When database is closed, it should return 503 Service Unavailable
			//
			// ACTUAL BEHAVIOR (Bug #22):
			// Health endpoint always returns 200 OK without checking dependencies
			// This test WILL FAIL, documenting that Bug #22 exists

			GinkgoWriter.Printf("\n=== BUG #22 EXPOSED ===\n")
			GinkgoWriter.Printf("Health endpoint returns 200 OK even when database is closed.\n")
			GinkgoWriter.Printf("EXPECTED: HTTP 503 Service Unavailable when DB connection closed\n")
			GinkgoWriter.Printf("ACTUAL: HTTP %d (health check doesn't verify DB connectivity)\n", w.Code)
			GinkgoWriter.Printf("FIX: HealthHandler should run db.Ping() and return 503 if it fails\n")
			GinkgoWriter.Printf("=========================\n")

			// This assertion documents the EXPECTED behavior (will fail with current implementation)
			Expect(w.Code).Should(Equal(http.StatusServiceUnavailable),
				"Health endpoint should return 503 when database is unavailable (Bug #22)")

			var response map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(response["status"]).Should(Equal("unhealthy"),
				"Health endpoint should return 'unhealthy' status when database is down")
		})

		It("should verify database connectivity with ping query", func() {
			// This test documents the EXPECTED implementation (will fail until Bug #22 is fixed)
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			server.HealthHandler(w, req)

			// When dependencies are healthy, should return 200
			Expect(w.Code).Should(Equal(http.StatusOK))

			// Now close DB and verify health check detects it
			server.DB.Close()

			req2 := httptest.NewRequest(http.MethodGet, "/health", nil)
			w2 := httptest.NewRecorder()

			server.HealthHandler(w2, req2)

			GinkgoWriter.Printf("\n=== BUG #22 EXPOSED ===\n")
			GinkgoWriter.Printf("Health endpoint should ping database to verify connectivity.\n")
			GinkgoWriter.Printf("EXPECTED: db.Ping() called, returns 503 when ping fails\n")
			GinkgoWriter.Printf("ACTUAL: No database check performed, always returns 200\n")
			GinkgoWriter.Printf("=========================\n")

			// Expected: 503 when database is closed
			Expect(w2.Code).Should(Equal(http.StatusServiceUnavailable),
				"Health endpoint should detect closed database connection via ping (Bug #22)")
		})

		It("should handle database ping errors gracefully without crashing server", func() {
			// This test verifies health check is isolated and doesn't crash the server
			// Even if DB ping fails, the handler should return 503 gracefully

			// Close database connection
			server.DB.Close()

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			// This should NOT panic or crash - should handle error gracefully
			server.HealthHandler(w, req)

			GinkgoWriter.Printf("\n=== BUG #22 EXPOSED ===\n")
			GinkgoWriter.Printf("Health check should handle database errors gracefully.\n")
			GinkgoWriter.Printf("EXPECTED: Returns 503 without panic when db.Ping() fails\n")
			GinkgoWriter.Printf("ACTUAL: Always returns 200, no error handling needed (no DB check)\n")
			GinkgoWriter.Printf("=========================\n")

			// Expected: graceful 503 response, not panic
			Expect(w.Code).Should(Equal(http.StatusServiceUnavailable),
				"Health endpoint should gracefully return 503 when database ping fails (Bug #22)")

			var response map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	When("testing NATS dependency health checks (revealing Bug #22)", func() {
		It("should return 503 when NATS connection is nil (not implemented)", func() {
			// Server is initialized with NATS: nil in BeforeEach
			// Health check should detect this and return unhealthy status

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			server.HealthHandler(w, req)

			GinkgoWriter.Printf("\n=== BUG #22 EXPOSED ===\n")
			GinkgoWriter.Printf("Health endpoint doesn't check NATS connectivity.\n")
			GinkgoWriter.Printf("EXPECTED: HTTP 503 when NATS is nil (service unhealthy)\n")
			GinkgoWriter.Printf("ACTUAL: HTTP %d (health check ignores NATS status)\n", w.Code)
			GinkgoWriter.Printf("NOTE: NATS is optional for some endpoints (login, register)\n")
			GinkgoWriter.Printf("FIX: Either require NATS and check it, or document that health only checks DB\n")
			GinkgoWriter.Printf("=========================\n")

			// This is a design decision: should health check require NATS?
			// For now, we document that it SHOULD check NATS if it's expected to be present
			// This test will fail with current implementation
			Expect(w.Code).Should(Equal(http.StatusServiceUnavailable),
				"Health endpoint should return 503 when NATS is nil (Bug #22) - OR document that NATS check is not required")

			var response map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(response["status"]).Should(Equal("unhealthy"),
				"Health endpoint should indicate unhealthy status when dependencies unavailable")
		})
	})

	When("testing health endpoint isolation", func() {
		It("should not affect server stability when health checks fail", func() {
			// Health checks failing should not bring down the server
			// This verifies the health endpoint is properly isolated

			// Close database
			server.DB.Close()

			// Multiple health check requests with failed DB should all return gracefully
			for i := 0; i < 5; i++ {
				req := httptest.NewRequest(http.MethodGet, "/health", nil)
				w := httptest.NewRecorder()

				// Should not panic
				server.HealthHandler(w, req)

				// Should return a response (even if it's wrong status code due to Bug #22)
				Expect(w.Body.Bytes()).ShouldNot(BeEmpty())

				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				Expect(err).ShouldNot(HaveOccurred(),
					"Health endpoint should return valid JSON even when dependencies fail")
			}

			GinkgoWriter.Printf("\n=== Test Passed ===\n")
			GinkgoWriter.Printf("Health endpoint doesn't crash server when dependencies fail.\n")
			GinkgoWriter.Printf("This test passes because handler is simple and doesn't check dependencies.\n")
			GinkgoWriter.Printf("When Bug #22 is fixed, verify error handling doesn't cause panics.\n")
			GinkgoWriter.Printf("==================\n")
		})
	})
})
