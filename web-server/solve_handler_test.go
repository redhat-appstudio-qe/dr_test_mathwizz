package main

// Unit tests for SolveHandler - tests mathematical expression solving endpoint
// Covers happy path, validation failures, authorization, and NATS resilience

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SolveHandler", func() {
	var (
		server     *Server
		mockDB     *sql.DB
		recorder   *httptest.ResponseRecorder
		testSecret = "test-secret-key-at-least-32-characters-long-for-jwt"
		testUserID = 42
		req        *http.Request
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

		// All solve requests need authenticated context
		req = httptest.NewRequest(http.MethodPost, "/solve", nil)
		ctx := context.WithValue(req.Context(), userIDKey, testUserID)
		req = req.WithContext(ctx)
	})

	AfterEach(func() {
		mockDB.Close()
		os.Unsetenv("JWT_SECRET")
		jwtSecret = nil
	})

	When("testing happy path scenarios", func() {
		It("should successfully solve simple addition problem", func() {
			reqBody := `{"problem":"25+75"}`
			req = httptest.NewRequest(http.MethodPost, "/solve", strings.NewReader(reqBody))
			ctx := context.WithValue(req.Context(), userIDKey, testUserID)
			req = req.WithContext(ctx)

			server.SolveHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusOK))

			var response SolveResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &response)).Should(Succeed())
			Expect(response.Problem).Should(Equal("25+75"))
			Expect(response.Answer).Should(Equal("100"))
		})

		It("should successfully solve multiplication problem", func() {
			reqBody := `{"problem":"10*5"}`
			req = httptest.NewRequest(http.MethodPost, "/solve", strings.NewReader(reqBody))
			ctx := context.WithValue(req.Context(), userIDKey, testUserID)
			req = req.WithContext(ctx)

			server.SolveHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusOK))

			var response SolveResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &response)).Should(Succeed())
			Expect(response.Problem).Should(Equal("10*5"))
			Expect(response.Answer).Should(Equal("50"))
		})

		It("should successfully solve complex expression with parentheses", func() {
			reqBody := `{"problem":"(10+5)*2"}`
			req = httptest.NewRequest(http.MethodPost, "/solve", strings.NewReader(reqBody))
			ctx := context.WithValue(req.Context(), userIDKey, testUserID)
			req = req.WithContext(ctx)

			server.SolveHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusOK))

			var response SolveResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &response)).Should(Succeed())
			Expect(response.Answer).Should(Equal("30"))
		})
	})

	When("testing request validation failures", func() {
		It("should return 400 Bad Request for malformed JSON", func() {
			req = httptest.NewRequest(http.MethodPost, "/solve", strings.NewReader(`{invalid`))
			ctx := context.WithValue(req.Context(), userIDKey, testUserID)
			req = req.WithContext(ctx)

			server.SolveHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusBadRequest))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("invalid request body"))
		})

		It("should return 400 Bad Request when problem is empty", func() {
			reqBody := `{"problem":""}`
			req = httptest.NewRequest(http.MethodPost, "/solve", strings.NewReader(reqBody))
			ctx := context.WithValue(req.Context(), userIDKey, testUserID)
			req = req.WithContext(ctx)

			server.SolveHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusBadRequest))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("problem is required"))
		})

		It("should return 400 Bad Request for invalid mathematical expression", func() {
			reqBody := `{"problem":"invalid expression"}`
			req = httptest.NewRequest(http.MethodPost, "/solve", strings.NewReader(reqBody))
			ctx := context.WithValue(req.Context(), userIDKey, testUserID)
			req = req.WithContext(ctx)

			server.SolveHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusBadRequest))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(ContainSubstring("failed to solve problem"))
		})
	})

	When("testing authorization failures", func() {
		It("should return 401 Unauthorized when user ID not in context", func() {
			reqBody := `{"problem":"2+2"}`
			req = httptest.NewRequest(http.MethodPost, "/solve", strings.NewReader(reqBody))

			server.SolveHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusUnauthorized))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("unauthorized"))
		})
	})

	When("testing NATS publishing resilience", func() {
		It("should return successful response even when NATS is nil", func() {
			reqBody := `{"problem":"5+5"}`
			req = httptest.NewRequest(http.MethodPost, "/solve", strings.NewReader(reqBody))
			ctx := context.WithValue(req.Context(), userIDKey, testUserID)
			req = req.WithContext(ctx)

			server.SolveHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusOK))

			var response SolveResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &response)).Should(Succeed())
			Expect(response.Answer).Should(Equal("10"))
		})
	})
})
