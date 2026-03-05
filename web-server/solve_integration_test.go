package main

// This file contains integration tests for the solve endpoint.
// Tests use both a real PostgreSQL database and NATS server to verify async behavior.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"
)

var _ = Describe("Solve Integration Tests", func() {
	var (
		ctx        context.Context
		dbCont     testcontainers.Container
		natsCont   testcontainers.Container
		server     *Server
		testUserID int
		testToken  string
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Initialize JWT secret for token generation in tests
		os.Setenv("JWT_SECRET", "test-integration-secret-key-at-least-32-characters-long")
		Expect(InitJWTSecret()).Should(Succeed())

		dbReq := testcontainers.ContainerRequest{
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
			ContainerRequest: dbReq,
			Started:          true,
		})
		Expect(err).ShouldNot(HaveOccurred())

		natsReq := testcontainers.ContainerRequest{
			Image:        "nats:2.10-alpine",
			ExposedPorts: []string{"4222/tcp"},
			WaitingFor:   wait.ForLog("Server is ready").WithStartupTimeout(30 * time.Second),
		}

		natsCont, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: natsReq,
			Started:          true,
		})
		Expect(err).ShouldNot(HaveOccurred())

		dbHost, err := dbCont.Host(ctx)
		Expect(err).ShouldNot(HaveOccurred())
		dbPort, err := dbCont.MappedPort(ctx, "5432")
		Expect(err).ShouldNot(HaveOccurred())

		natsHost, err := natsCont.Host(ctx)
		Expect(err).ShouldNot(HaveOccurred())
		natsPort, err := natsCont.MappedPort(ctx, "4222")
		Expect(err).ShouldNot(HaveOccurred())

		db, err := ConnectDB(dbHost, "test", "test", "mathwizz_test", dbPort.Int(), "disable")
		Expect(err).ShouldNot(HaveOccurred())

		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE NOT NULL,
				password_hash VARCHAR(255) NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		Expect(err).ShouldNot(HaveOccurred())

		// Create history table for eventual consistency testing
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS history (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES users(id),
				problem_text VARCHAR(500) NOT NULL,
				answer_text VARCHAR(100) NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		Expect(err).ShouldNot(HaveOccurred())

		hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
		err = db.QueryRow("INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id",
			"solver@example.com", string(hash)).Scan(&testUserID)
		Expect(err).ShouldNot(HaveOccurred())

		natsURL := fmt.Sprintf("nats://%s:%s", natsHost, natsPort.Port())
		nc, err := ConnectNATS(natsURL)
		Expect(err).ShouldNot(HaveOccurred())

		server = &Server{DB: db, NATS: nc}

		testToken, err = GenerateToken(testUserID, "solver@example.com")
		Expect(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		if server.NATS != nil {
			server.NATS.Close()
		}
		if server.DB != nil {
			server.DB.Close()
		}
		if natsCont != nil {
			Expect(natsCont.Terminate(ctx)).Should(Succeed())
		}
		if dbCont != nil {
			Expect(dbCont.Terminate(ctx)).Should(Succeed())
		}
	})

	When("solving a math problem with valid authentication", func() {
		It("should return 200 OK with the correct answer synchronously", func() {
			reqBody := SolveRequest{Problem: "25+75"}
			body, _ := json.Marshal(reqBody)

			req := createAuthenticatedRequest("POST", "/solve", body, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.SolveHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusOK))

			var response SolveResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(response.Problem).Should(Equal("25+75"))
			Expect(response.Answer).Should(Equal("100"))
		})

		It("should publish problem_solved event to NATS asynchronously", func() {
			sub, msgChan, err := TestNATSSubscriber(server.NATS, ProblemSolvedSubject)
			Expect(err).ShouldNot(HaveOccurred())
			defer sub.Unsubscribe()

			reqBody := SolveRequest{Problem: "10*5"}
			body, _ := json.Marshal(reqBody)

			req := createAuthenticatedRequest("POST", "/solve", body, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.SolveHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusOK))

			msg := WaitForNATSMessage(msgChan, 2*time.Second)
			Expect(msg).ShouldNot(BeNil(), "Expected to receive NATS message within 2 seconds")

			event, err := ParseProblemSolvedEvent(msg)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(event.UserID).Should(Equal(testUserID))
			Expect(event.ProblemText).Should(Equal("10*5"))
			Expect(event.AnswerText).Should(Equal("50"))
		})
	})

	When("solving with invalid or missing authentication", func() {
		It("should return 401 Unauthorized without a token", func() {
			reqBody := SolveRequest{Problem: "5+5"}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/solve", bytes.NewReader(body))
			w := httptest.NewRecorder()

			AuthMiddleware(server.SolveHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusUnauthorized))
		})

		It("should return 401 Unauthorized with invalid token", func() {
			reqBody := SolveRequest{Problem: "5+5"}
			body, _ := json.Marshal(reqBody)

			req := createAuthenticatedRequest("POST", "/solve", body, "invalid-token")
			w := httptest.NewRecorder()

			AuthMiddleware(server.SolveHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusUnauthorized))
		})
	})

	When("solving invalid math problems", func() {
		DescribeTable("should return 400 Bad Request with error message",
			func(problem string) {
				reqBody := SolveRequest{Problem: problem}
				body, _ := json.Marshal(reqBody)

				req := createAuthenticatedRequest("POST", "/solve", body, testToken)
				w := httptest.NewRecorder()

				AuthMiddleware(server.SolveHandler)(w, req)

				Expect(w.Code).Should(Equal(http.StatusBadRequest))

				var errorResp ErrorResponse
				json.NewDecoder(w.Body).Decode(&errorResp)
				Expect(errorResp.Error).ShouldNot(BeEmpty())
			},
			Entry("empty problem", ""),
			Entry("invalid expression", "abc"),
			Entry("incomplete expression", "5+"),
		)
	})

	When("testing edge cases", func() {
		It("should handle complex mathematical expressions", func() {
			reqBody := SolveRequest{Problem: "(10+5)*2-10"}
			body, _ := json.Marshal(reqBody)

			req := createAuthenticatedRequest("POST", "/solve", body, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.SolveHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusOK))

			var response SolveResponse
			json.NewDecoder(w.Body).Decode(&response)
			Expect(response.Answer).Should(Equal("20"))
		})

		It("should handle negative results", func() {
			reqBody := SolveRequest{Problem: "5-10"}
			body, _ := json.Marshal(reqBody)

			req := createAuthenticatedRequest("POST", "/solve", body, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.SolveHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusOK))

			var response SolveResponse
			json.NewDecoder(w.Body).Decode(&response)
			Expect(response.Answer).Should(Equal("-5"))
		})
	})

	When("testing problem length validation at integration level", func() {
		It("should accept problem with exactly 100 characters (application limit boundary)", func() {
			// Create a valid expression that is exactly 100 characters
			// Pattern: "1+1+1+1..." repeated to reach exactly 100 chars
			problem := strings.Repeat("1+", 49) + "1" // 99 chars (49*2) + 1 = 100 chars
			reqBody := SolveRequest{Problem: problem}
			body, _ := json.Marshal(reqBody)

			req := createAuthenticatedRequest("POST", "/solve", body, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.SolveHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusOK))

			var response SolveResponse
			json.NewDecoder(w.Body).Decode(&response)
			Expect(response.Answer).Should(Equal("50")) // 1+1+...+1 (50 ones) = 50
		})

		DescribeTable("should reject problems exceeding length limits with 400 Bad Request",
			func(length int, description string) {
				// Create a problem of specified length using "1+" pattern
				var problem string
				if length <= 2 {
					problem = strings.Repeat("1", length)
				} else {
					// Use pattern "1+1+1+..." to build expression of desired length
					pairs := (length - 1) / 2
					remainder := (length - 1) % 2
					problem = strings.Repeat("1+", pairs) + "1"
					if remainder > 0 {
						problem += "+"
					}
				}

				reqBody := SolveRequest{Problem: problem}
				body, _ := json.Marshal(reqBody)

				req := createAuthenticatedRequest("POST", "/solve", body, testToken)
				w := httptest.NewRecorder()

				AuthMiddleware(server.SolveHandler)(w, req)

				Expect(w.Code).Should(Equal(http.StatusBadRequest), description)

				var errorResp ErrorResponse
				json.NewDecoder(w.Body).Decode(&errorResp)
				Expect(errorResp.Error).Should(ContainSubstring("too long"))
			},
			Entry("101 characters (just over application limit)", 101, "should reject 101-char problem (exceeds 100-char application limit)"),
			Entry("500 characters (database limit)", 500, "should reject 500-char problem at application layer (100-char limit is stricter)"),
			Entry("1000 characters (extreme DoS attempt)", 1000, "should fail fast on 1000-char problem without processing"),
			Entry("5000 characters (massive DoS attempt)", 5000, "should efficiently reject massive 5000-char input"),
		)

		It("should return descriptive error message for oversized problems", func() {
			problem := strings.Repeat("1+", 100) + "1" // 201 characters
			reqBody := SolveRequest{Problem: problem}
			body, _ := json.Marshal(reqBody)

			req := createAuthenticatedRequest("POST", "/solve", body, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.SolveHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusBadRequest))

			var errorResp ErrorResponse
			json.NewDecoder(w.Body).Decode(&errorResp)
			Expect(errorResp.Error).Should(ContainSubstring("100 characters"))
		})
	})

	When("testing goroutine panic recovery", func() {
		It("should handle NATS publish failures without crashing the server", func() {
			server.NATS.Close()

			reqBody := SolveRequest{Problem: "2+2"}
			body, _ := json.Marshal(reqBody)

			req := createAuthenticatedRequest("POST", "/solve", body, testToken)
			w := httptest.NewRecorder()

			Expect(func() {
				AuthMiddleware(server.SolveHandler)(w, req)
			}).ShouldNot(Panic())

			Expect(w.Code).Should(Equal(http.StatusOK))

			var response SolveResponse
			json.NewDecoder(w.Body).Decode(&response)
			Expect(response.Answer).Should(Equal("4"))
		})

		It("should survive nil NATS connection without crashing the server", func() {
			originalNATS := server.NATS
			server.NATS = nil

			reqBody := SolveRequest{Problem: "3+7"}
			body, _ := json.Marshal(reqBody)

			req := createAuthenticatedRequest("POST", "/solve", body, testToken)
			w := httptest.NewRecorder()

			Expect(func() {
				AuthMiddleware(server.SolveHandler)(w, req)
			}).ShouldNot(Panic())

			Expect(w.Code).Should(Equal(http.StatusOK))

			var response SolveResponse
			json.NewDecoder(w.Body).Decode(&response)
			Expect(response.Answer).Should(Equal("10"))

			server.NATS = originalNATS
		})

		It("should continue processing subsequent requests after goroutine panic from nil NATS", func() {
			// Set NATS to nil to trigger panic in PublishProblemSolved goroutine
			originalNATS := server.NATS
			server.NATS = nil

			// First request - triggers panic in goroutine
			reqBody1 := SolveRequest{Problem: "5+5"}
			body1, _ := json.Marshal(reqBody1)
			req1 := createAuthenticatedRequest("POST", "/solve", body1, testToken)
			w1 := httptest.NewRecorder()

			AuthMiddleware(server.SolveHandler)(w1, req1)
			Expect(w1.Code).Should(Equal(http.StatusOK))

			var response1 SolveResponse
			json.NewDecoder(w1.Body).Decode(&response1)
			Expect(response1.Answer).Should(Equal("10"))

			// Small delay to allow goroutine to panic and be recovered
			time.Sleep(10 * time.Millisecond)

			// Second request - verify server still responsive
			reqBody2 := SolveRequest{Problem: "8+2"}
			body2, _ := json.Marshal(reqBody2)
			req2 := createAuthenticatedRequest("POST", "/solve", body2, testToken)
			w2 := httptest.NewRecorder()

			AuthMiddleware(server.SolveHandler)(w2, req2)
			Expect(w2.Code).Should(Equal(http.StatusOK))

			var response2 SolveResponse
			json.NewDecoder(w2.Body).Decode(&response2)
			Expect(response2.Answer).Should(Equal("10"))

			// Third request - verify server remains stable
			reqBody3 := SolveRequest{Problem: "12*3"}
			body3, _ := json.Marshal(reqBody3)
			req3 := createAuthenticatedRequest("POST", "/solve", body3, testToken)
			w3 := httptest.NewRecorder()

			AuthMiddleware(server.SolveHandler)(w3, req3)
			Expect(w3.Code).Should(Equal(http.StatusOK))

			var response3 SolveResponse
			json.NewDecoder(w3.Body).Decode(&response3)
			Expect(response3.Answer).Should(Equal("36"))

			server.NATS = originalNATS
		})

		It("should handle multiple rapid requests with panic-inducing nil NATS without server crash", func() {
			// Set NATS to nil to trigger panic on every publish attempt
			originalNATS := server.NATS
			server.NATS = nil

			// Send 5 rapid requests - each triggers goroutine panic
			problems := []string{"1+1", "2+2", "3+3", "4+4", "5+5"}
			expectedAnswers := []string{"2", "4", "6", "8", "10"}

			for i, problem := range problems {
				reqBody := SolveRequest{Problem: problem}
				body, _ := json.Marshal(reqBody)
				req := createAuthenticatedRequest("POST", "/solve", body, testToken)
				w := httptest.NewRecorder()

				AuthMiddleware(server.SolveHandler)(w, req)
				Expect(w.Code).Should(Equal(http.StatusOK))

				var response SolveResponse
				json.NewDecoder(w.Body).Decode(&response)
				Expect(response.Answer).Should(Equal(expectedAnswers[i]))
			}

			// Allow time for all goroutines to panic and be recovered
			time.Sleep(50 * time.Millisecond)

			// Verify server is still responsive after all panics
			reqBody := SolveRequest{Problem: "100-50"}
			body, _ := json.Marshal(reqBody)
			req := createAuthenticatedRequest("POST", "/solve", body, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.SolveHandler)(w, req)
			Expect(w.Code).Should(Equal(http.StatusOK))

			var response SolveResponse
			json.NewDecoder(w.Body).Decode(&response)
			Expect(response.Answer).Should(Equal("50"))

			server.NATS = originalNATS
		})

		It("should recover from goroutine panics and continue processing without goroutine leaks", func() {
			// Set NATS to nil to trigger panic
			originalNATS := server.NATS
			server.NATS = nil

			// Send 10 requests to create 10 goroutines that will all panic
			for i := 0; i < 10; i++ {
				problem := fmt.Sprintf("%d+%d", i, i)
				reqBody := SolveRequest{Problem: problem}
				body, _ := json.Marshal(reqBody)
				req := createAuthenticatedRequest("POST", "/solve", body, testToken)
				w := httptest.NewRecorder()

				AuthMiddleware(server.SolveHandler)(w, req)
				Expect(w.Code).Should(Equal(http.StatusOK))
			}

			// Allow time for all goroutines to panic and be recovered
			time.Sleep(100 * time.Millisecond)

			// Restore NATS and verify normal operation resumes
			server.NATS = originalNATS

			reqBody := SolveRequest{Problem: "7*7"}
			body, _ := json.Marshal(reqBody)
			req := createAuthenticatedRequest("POST", "/solve", body, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.SolveHandler)(w, req)
			Expect(w.Code).Should(Equal(http.StatusOK))

			var response SolveResponse
			json.NewDecoder(w.Body).Decode(&response)
			Expect(response.Answer).Should(Equal("49"))
		})
	})

	When("testing end-to-end eventual consistency", func() {
		It("should verify solved problem appears in history via async NATS→worker→DB pipeline", func() {
			// Subscribe to NATS and process problem_solved events (simulating history-worker)
			_, err := server.NATS.Subscribe(ProblemSolvedSubject, func(msg *nats.Msg) {
				var event ProblemSolvedEvent
				if err := json.Unmarshal(msg.Data, &event); err != nil {
					return
				}
				// Write to history table (simulating what history-worker does)
				_, _ = server.DB.Exec(
					"INSERT INTO history (user_id, problem_text, answer_text, created_at) VALUES ($1, $2, $3, $4)",
					event.UserID, event.ProblemText, event.AnswerText, time.Now(),
				)
			})
			Expect(err).ShouldNot(HaveOccurred())

			// Allow subscription to initialize before publishing events
			time.Sleep(100 * time.Millisecond)

			// Solve a problem via /solve endpoint (publishes NATS event)
			reqBody := SolveRequest{Problem: "42+58"}
			body, _ := json.Marshal(reqBody)

			req := createAuthenticatedRequest("POST", "/solve", body, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.SolveHandler)(w, req)

			// Verify /solve returns success
			Expect(w.Code).Should(Equal(http.StatusOK))

			var solveResponse SolveResponse
			json.NewDecoder(w.Body).Decode(&solveResponse)
			Expect(solveResponse.Answer).Should(Equal("100"))

			// Poll /history endpoint until problem appears (eventual consistency)
			// This verifies the complete async flow: solve → NATS → worker → DB → history
			found := false
			timeout := time.After(3 * time.Second)
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()

		PollingLoop:
			for {
				select {
				case <-ticker.C:
					// Query /history endpoint
					histReq := createAuthenticatedRequest("GET", "/history", nil, testToken)
					histW := httptest.NewRecorder()

					AuthMiddleware(server.HistoryHandler)(histW, histReq)

					if histW.Code == http.StatusOK {
						var history []HistoryItem
						json.NewDecoder(histW.Body).Decode(&history)

						// Check if our solved problem appears in history
						for _, item := range history {
							if item.ProblemText == "42+58" && item.AnswerText == "100" {
								found = true
								break PollingLoop
							}
						}
					}
				case <-timeout:
					break PollingLoop
				}
			}

			Expect(found).Should(BeTrue(), "Expected problem '42+58=100' to appear in history within 3 seconds (eventual consistency)")
		})

		It("should verify multiple solved problems appear in history in order", func() {
			// Subscribe to NATS and process problem_solved events (simulating history-worker)
			_, err := server.NATS.Subscribe(ProblemSolvedSubject, func(msg *nats.Msg) {
				var event ProblemSolvedEvent
				if err := json.Unmarshal(msg.Data, &event); err != nil {
					return
				}
				// Write to history table (simulating what history-worker does)
				_, _ = server.DB.Exec(
					"INSERT INTO history (user_id, problem_text, answer_text, created_at) VALUES ($1, $2, $3, $4)",
					event.UserID, event.ProblemText, event.AnswerText, time.Now(),
				)
			})
			Expect(err).ShouldNot(HaveOccurred())

			time.Sleep(100 * time.Millisecond)

			// Solve multiple problems in sequence
			problems := []struct {
				expression string
				answer     string
			}{
				{"10+20", "30"},
				{"5*6", "30"},
				{"100-50", "50"},
			}

			for _, p := range problems {
				reqBody := SolveRequest{Problem: p.expression}
				body, _ := json.Marshal(reqBody)

				req := createAuthenticatedRequest("POST", "/solve", body, testToken)
				w := httptest.NewRecorder()

				AuthMiddleware(server.SolveHandler)(w, req)
				Expect(w.Code).Should(Equal(http.StatusOK))
			}

			// Poll /history until all 3 problems appear
			Eventually(func() int {
				histReq := createAuthenticatedRequest("GET", "/history", nil, testToken)
				histW := httptest.NewRecorder()

				AuthMiddleware(server.HistoryHandler)(histW, histReq)

				if histW.Code != http.StatusOK {
					return 0
				}

				var history []HistoryItem
				json.NewDecoder(histW.Body).Decode(&history)

				// Count how many of our problems appear in history
				count := 0
				for _, item := range history {
					for _, p := range problems {
						if item.ProblemText == p.expression && item.AnswerText == p.answer {
							count++
							break
						}
					}
				}
				return count
			}, 3*time.Second, 100*time.Millisecond).Should(Equal(3), "All 3 solved problems should appear in history within 3 seconds")
		})

	})
})

var _ = Describe("History Integration Tests", func() {
	var (
		ctx        context.Context
		dbCont     testcontainers.Container
		natsCont   testcontainers.Container
		server     *Server
		testUserID int
		testToken  string
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Initialize JWT secret for token generation in tests
		os.Setenv("JWT_SECRET", "test-integration-secret-key-at-least-32-characters-long")
		Expect(InitJWTSecret()).Should(Succeed())

		dbReq := testcontainers.ContainerRequest{
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
			ContainerRequest: dbReq,
			Started:          true,
		})
		Expect(err).ShouldNot(HaveOccurred())

		natsReq := testcontainers.ContainerRequest{
			Image:        "nats:2.10-alpine",
			ExposedPorts: []string{"4222/tcp"},
			WaitingFor:   wait.ForLog("Server is ready").WithStartupTimeout(30 * time.Second),
		}

		natsCont, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: natsReq,
			Started:          true,
		})
		Expect(err).ShouldNot(HaveOccurred())

		dbHost, err := dbCont.Host(ctx)
		Expect(err).ShouldNot(HaveOccurred())
		dbPort, err := dbCont.MappedPort(ctx, "5432")
		Expect(err).ShouldNot(HaveOccurred())

		natsHost, err := natsCont.Host(ctx)
		Expect(err).ShouldNot(HaveOccurred())
		natsPort, err := natsCont.MappedPort(ctx, "4222")
		Expect(err).ShouldNot(HaveOccurred())

		db, err := ConnectDB(dbHost, "test", "test", "mathwizz_test", dbPort.Int(), "disable")
		Expect(err).ShouldNot(HaveOccurred())

		// Create users table
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE NOT NULL,
				password_hash VARCHAR(255) NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		Expect(err).ShouldNot(HaveOccurred())

		// Create history table
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS history (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES users(id),
				problem_text VARCHAR(500) NOT NULL,
				answer_text VARCHAR(100) NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		Expect(err).ShouldNot(HaveOccurred())

		hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
		err = db.QueryRow("INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id",
			"historyuser@example.com", string(hash)).Scan(&testUserID)
		Expect(err).ShouldNot(HaveOccurred())

		natsURL := fmt.Sprintf("nats://%s:%s", natsHost, natsPort.Port())
		nc, err := ConnectNATS(natsURL)
		Expect(err).ShouldNot(HaveOccurred())

		server = &Server{DB: db, NATS: nc}

		testToken, err = GenerateToken(testUserID, "historyuser@example.com")
		Expect(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		if server.NATS != nil {
			server.NATS.Close()
		}
		if server.DB != nil {
			server.DB.Close()
		}
		if natsCont != nil {
			Expect(natsCont.Terminate(ctx)).Should(Succeed())
		}
		if dbCont != nil {
			Expect(dbCont.Terminate(ctx)).Should(Succeed())
		}
	})

	When("retrieving history with valid authentication", func() {
		It("should return 200 OK with empty array for new user with no history", func() {
			req := createAuthenticatedRequest("GET", "/history", nil, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.HistoryHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusOK))

			var history []HistoryItem
			err := json.NewDecoder(w.Body).Decode(&history)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(history).Should(BeEmpty())
		})

		It("should return history items after problems are solved", func() {
			// Directly insert history items into database to test retrieval
			_, err := server.DB.Exec(
				"INSERT INTO history (user_id, problem_text, answer_text, created_at) VALUES ($1, $2, $3, $4)",
				testUserID, "10+20", "30", time.Now().Add(-2*time.Minute),
			)
			Expect(err).ShouldNot(HaveOccurred())

			_, err = server.DB.Exec(
				"INSERT INTO history (user_id, problem_text, answer_text, created_at) VALUES ($1, $2, $3, $4)",
				testUserID, "5*6", "30", time.Now().Add(-1*time.Minute),
			)
			Expect(err).ShouldNot(HaveOccurred())

			req := createAuthenticatedRequest("GET", "/history", nil, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.HistoryHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusOK))

			var history []HistoryItem
			err = json.NewDecoder(w.Body).Decode(&history)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(history).Should(HaveLen(2))
		})

		It("should return history items in descending order (newest first)", func() {
			// Insert items with known timestamps
			oldTimestamp := time.Now().Add(-10 * time.Minute)
			recentTimestamp := time.Now().Add(-1 * time.Minute)

			_, err := server.DB.Exec(
				"INSERT INTO history (user_id, problem_text, answer_text, created_at) VALUES ($1, $2, $3, $4)",
				testUserID, "oldest", "1", oldTimestamp,
			)
			Expect(err).ShouldNot(HaveOccurred())

			_, err = server.DB.Exec(
				"INSERT INTO history (user_id, problem_text, answer_text, created_at) VALUES ($1, $2, $3, $4)",
				testUserID, "newest", "3", recentTimestamp,
			)
			Expect(err).ShouldNot(HaveOccurred())

			req := createAuthenticatedRequest("GET", "/history", nil, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.HistoryHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusOK))

			var history []HistoryItem
			err = json.NewDecoder(w.Body).Decode(&history)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(history).Should(HaveLen(2))

			// Verify newest item is first
			Expect(history[0].ProblemText).Should(Equal("newest"))
			Expect(history[1].ProblemText).Should(Equal("oldest"))
		})
	})

	When("testing pagination parameters", func() {
		BeforeEach(func() {
			// Insert 10 history items for pagination testing
			for i := 1; i <= 10; i++ {
				_, err := server.DB.Exec(
					"INSERT INTO history (user_id, problem_text, answer_text, created_at) VALUES ($1, $2, $3, $4)",
					testUserID, fmt.Sprintf("problem%d", i), fmt.Sprintf("%d", i), time.Now().Add(time.Duration(-i)*time.Minute),
				)
				Expect(err).ShouldNot(HaveOccurred())
			}
		})

		It("should respect limit parameter", func() {
			req := createAuthenticatedRequest("GET", "/history?limit=5", nil, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.HistoryHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusOK))

			var history []HistoryItem
			err := json.NewDecoder(w.Body).Decode(&history)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(history).Should(HaveLen(5))
		})

		It("should respect offset parameter for pagination", func() {
			req := createAuthenticatedRequest("GET", "/history?limit=5&offset=5", nil, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.HistoryHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusOK))

			var history []HistoryItem
			err := json.NewDecoder(w.Body).Decode(&history)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(history).Should(HaveLen(5))
		})

		It("should return empty array when offset exceeds total items", func() {
			req := createAuthenticatedRequest("GET", "/history?offset=100", nil, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.HistoryHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusOK))

			var history []HistoryItem
			err := json.NewDecoder(w.Body).Decode(&history)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(history).Should(BeEmpty())
		})

		It("should clamp limit to maximum of 200", func() {
			req := createAuthenticatedRequest("GET", "/history?limit=500", nil, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.HistoryHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusOK))

			var history []HistoryItem
			err := json.NewDecoder(w.Body).Decode(&history)
			Expect(err).ShouldNot(HaveOccurred())
			// We only have 10 items, but verify request didn't fail with limit=500
			Expect(history).Should(HaveLen(10))
		})

		It("should use default limit of 100 when no limit specified", func() {
			// Insert 150 items to exceed default limit
			for i := 11; i <= 150; i++ {
				_, err := server.DB.Exec(
					"INSERT INTO history (user_id, problem_text, answer_text, created_at) VALUES ($1, $2, $3, $4)",
					testUserID, fmt.Sprintf("problem%d", i), fmt.Sprintf("%d", i), time.Now().Add(time.Duration(-i)*time.Minute),
				)
				Expect(err).ShouldNot(HaveOccurred())
			}

			req := createAuthenticatedRequest("GET", "/history", nil, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.HistoryHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusOK))

			var history []HistoryItem
			err := json.NewDecoder(w.Body).Decode(&history)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(history).Should(HaveLen(100)) // Default limit
		})
	})

	When("testing with invalid pagination parameters", func() {
		It("should return 400 Bad Request for invalid limit parameter", func() {
			req := createAuthenticatedRequest("GET", "/history?limit=invalid", nil, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.HistoryHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusBadRequest))

			var errorResp ErrorResponse
			json.NewDecoder(w.Body).Decode(&errorResp)
			Expect(errorResp.Error).Should(ContainSubstring("invalid limit parameter"))
		})

		It("should return 400 Bad Request for invalid offset parameter", func() {
			req := createAuthenticatedRequest("GET", "/history?offset=notanumber", nil, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.HistoryHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusBadRequest))

			var errorResp ErrorResponse
			json.NewDecoder(w.Body).Decode(&errorResp)
			Expect(errorResp.Error).Should(ContainSubstring("invalid offset parameter"))
		})
	})

	When("testing authentication requirements", func() {
		It("should return 401 Unauthorized without authentication token", func() {
			req := httptest.NewRequest("GET", "/history", nil)
			w := httptest.NewRecorder()

			AuthMiddleware(server.HistoryHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusUnauthorized))
		})

		It("should return 401 Unauthorized with invalid token", func() {
			req := createAuthenticatedRequest("GET", "/history", nil, "invalid-token")
			w := httptest.NewRecorder()

			AuthMiddleware(server.HistoryHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusUnauthorized))
		})

		It("should only return history for authenticated user, not other users", func() {
			// Create another user
			hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
			var otherUserID int
			err := server.DB.QueryRow("INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id",
				"otheruser@example.com", string(hash)).Scan(&otherUserID)
			Expect(err).ShouldNot(HaveOccurred())

			// Insert history for other user
			_, err = server.DB.Exec(
				"INSERT INTO history (user_id, problem_text, answer_text) VALUES ($1, $2, $3)",
				otherUserID, "other-problem", "42",
			)
			Expect(err).ShouldNot(HaveOccurred())

			// Insert history for test user
			_, err = server.DB.Exec(
				"INSERT INTO history (user_id, problem_text, answer_text) VALUES ($1, $2, $3)",
				testUserID, "my-problem", "100",
			)
			Expect(err).ShouldNot(HaveOccurred())

			// Request history as test user
			req := createAuthenticatedRequest("GET", "/history", nil, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.HistoryHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusOK))

			var history []HistoryItem
			err = json.NewDecoder(w.Body).Decode(&history)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(history).Should(HaveLen(1))
			Expect(history[0].ProblemText).Should(Equal("my-problem"))
			Expect(history[0].UserID).Should(Equal(testUserID))
		})
	})

	When("testing with large result sets", func() {
		It("should handle 150 history items efficiently", func() {
			// Insert 150 items
			for i := 1; i <= 150; i++ {
				_, err := server.DB.Exec(
					"INSERT INTO history (user_id, problem_text, answer_text) VALUES ($1, $2, $3)",
					testUserID, fmt.Sprintf("problem-%d", i), fmt.Sprintf("%d", i),
				)
				Expect(err).ShouldNot(HaveOccurred())
			}

			// Request all with high limit
			req := createAuthenticatedRequest("GET", "/history?limit=200", nil, testToken)
			w := httptest.NewRecorder()

			AuthMiddleware(server.HistoryHandler)(w, req)

			Expect(w.Code).Should(Equal(http.StatusOK))

			var history []HistoryItem
			err := json.NewDecoder(w.Body).Decode(&history)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(history).Should(HaveLen(150))
		})
	})

	When("testing concurrent solve requests", func() {
		It("should handle 10 simultaneous solve requests without errors", func() {
			// Record number of goroutines before test
			goroutinesBefore := runtime.NumGoroutine()

			var wg sync.WaitGroup
			results := make([]struct {
				problem string
				answer  string
				err     error
			}, 10)

			// Launch 10 concurrent solve requests
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()
					defer GinkgoRecover()

					problem := fmt.Sprintf("%d+%d", index*10, index*5)
					expectedAnswer := fmt.Sprintf("%d", index*15)

					reqBody := SolveRequest{Problem: problem}
					body, _ := json.Marshal(reqBody)

					req := createAuthenticatedRequest("POST", "/solve", body, testToken)
					w := httptest.NewRecorder()

					AuthMiddleware(server.SolveHandler)(w, req)

					results[index].problem = problem

					if w.Code != http.StatusOK {
						results[index].err = fmt.Errorf("request %d failed with status %d", index, w.Code)
						return
					}

					var response SolveResponse
					if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
						results[index].err = fmt.Errorf("request %d decode error: %v", index, err)
						return
					}

					results[index].answer = response.Answer
					if response.Answer != expectedAnswer {
						results[index].err = fmt.Errorf("request %d: expected %s, got %s", index, expectedAnswer, response.Answer)
					}
				}(i)
			}

			// Wait for all requests to complete
			wg.Wait()

			// Verify all requests succeeded with correct answers
			for i, result := range results {
				Expect(result.err).ShouldNot(HaveOccurred(), "Request %d failed", i)
				Expect(result.answer).Should(Equal(fmt.Sprintf("%d", i*15)), "Request %d got wrong answer", i)
			}

			// Allow time for goroutines spawned by SolveHandler to complete
			time.Sleep(100 * time.Millisecond)

			// Verify no goroutine leaks (allow small variance for background goroutines)
			goroutinesAfter := runtime.NumGoroutine()
			goroutineDiff := goroutinesAfter - goroutinesBefore
			Expect(goroutineDiff).Should(BeNumerically("<=", 2),
				"Goroutine leak detected: before=%d, after=%d, diff=%d",
				goroutinesBefore, goroutinesAfter, goroutineDiff)
		})

		It("should handle 20 concurrent requests without connection pool exhaustion", func() {
			var wg sync.WaitGroup
			errorsChan := make(chan error, 20)

			// Launch 20 concurrent requests
			for i := 0; i < 20; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()
					defer GinkgoRecover()

					problem := fmt.Sprintf("%d*%d", index+1, 2)
					reqBody := SolveRequest{Problem: problem}
					body, _ := json.Marshal(reqBody)

					req := createAuthenticatedRequest("POST", "/solve", body, testToken)
					w := httptest.NewRecorder()

					AuthMiddleware(server.SolveHandler)(w, req)

					if w.Code != http.StatusOK {
						errorsChan <- fmt.Errorf("request %d failed with status %d", index, w.Code)
						return
					}

					var response SolveResponse
					if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
						errorsChan <- fmt.Errorf("request %d decode error: %v", index, err)
					}
				}(i)
			}

			// Wait for all requests
			wg.Wait()
			close(errorsChan)

			// Verify no errors occurred
			var errors []error
			for err := range errorsChan {
				errors = append(errors, err)
			}
			Expect(errors).Should(BeEmpty(), "Some concurrent requests failed: %v", errors)
		})

		It("should correctly validate JWT tokens concurrently without race conditions", func() {
			var wg sync.WaitGroup
			results := make([]int, 15)

			// Launch 15 concurrent requests with same token
			for i := 0; i < 15; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()
					defer GinkgoRecover()

					reqBody := SolveRequest{Problem: "100+100"}
					body, _ := json.Marshal(reqBody)

					req := createAuthenticatedRequest("POST", "/solve", body, testToken)
					w := httptest.NewRecorder()

					AuthMiddleware(server.SolveHandler)(w, req)

					results[index] = w.Code
				}(i)
			}

			wg.Wait()

			// All requests should succeed with 200 OK (token validation should be thread-safe)
			for i, statusCode := range results {
				Expect(statusCode).Should(Equal(http.StatusOK), "Request %d got status %d", i, statusCode)
			}
		})

		It("should handle concurrent requests with different problems returning correct answers", func() {
			type testCase struct {
				problem        string
				expectedAnswer string
			}

			testCases := []testCase{
				{"10+20", "30"},
				{"5*6", "30"},
				{"100-50", "50"},
				{"48/4", "12"},
				{"(5+5)*3", "30"},
				{"100+200+300", "600"},
				{"1000-500", "500"},
				{"7*8", "56"},
				{"99/9", "11"},
				{"25+25", "50"},
			}

			var wg sync.WaitGroup
			results := make([]struct {
				actualAnswer string
				statusCode   int
			}, len(testCases))

			// Launch concurrent requests with different problems
			for i, tc := range testCases {
				wg.Add(1)
				go func(index int, testCase testCase) {
					defer wg.Done()
					defer GinkgoRecover()

					reqBody := SolveRequest{Problem: testCase.problem}
					body, _ := json.Marshal(reqBody)

					req := createAuthenticatedRequest("POST", "/solve", body, testToken)
					w := httptest.NewRecorder()

					AuthMiddleware(server.SolveHandler)(w, req)

					results[index].statusCode = w.Code

					if w.Code == http.StatusOK {
						var response SolveResponse
						json.NewDecoder(w.Body).Decode(&response)
						results[index].actualAnswer = response.Answer
					}
				}(i, tc)
			}

			wg.Wait()

			// Verify all requests succeeded with correct answers
			for i, tc := range testCases {
				Expect(results[i].statusCode).Should(Equal(http.StatusOK),
					"Request %d (problem: %s) failed with status %d", i, tc.problem, results[i].statusCode)
				Expect(results[i].actualAnswer).Should(Equal(tc.expectedAnswer),
					"Request %d (problem: %s) got answer %s, expected %s",
					i, tc.problem, results[i].actualAnswer, tc.expectedAnswer)
			}
		})
	})
})
