package main

// Unit tests for HistoryHandler - tests user history retrieval endpoint
// Covers happy path, pagination, authorization, and database errors

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HistoryHandler", func() {
	var (
		server     *Server
		mockDB     *sql.DB
		mock       sqlmock.Sqlmock
		recorder   *httptest.ResponseRecorder
		natsConn   *nats.Conn
		testSecret = "test-secret-key-at-least-32-characters-long-for-jwt"
		testUserID = 42
		req        *http.Request
	)

	BeforeEach(func() {
		os.Setenv("JWT_SECRET", testSecret)
		InitJWTSecret()

		var err error
		mockDB, mock, err = sqlmock.New()
		Expect(err).ShouldNot(HaveOccurred())

		server = &Server{
			DB:   mockDB,
			NATS: natsConn,
		}

		recorder = httptest.NewRecorder()

		// Create request with authenticated user context
		req = httptest.NewRequest(http.MethodGet, "/history", nil)
		ctx := context.WithValue(req.Context(), userIDKey, testUserID)
		req = req.WithContext(ctx)
	})

	AfterEach(func() {
		mockDB.Close()
		os.Unsetenv("JWT_SECRET")
		jwtSecret = nil
	})

	When("testing happy path scenarios", func() {
		It("should return empty array when user has no history", func() {
			mock.ExpectQuery(`SELECT id, user_id, problem_text, answer_text, created_at FROM history WHERE user_id = \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
				WithArgs(testUserID, 100, 0).
				WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "problem_text", "answer_text", "created_at"}))

			server.HistoryHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusOK))

			var history []HistoryItem
			Expect(json.Unmarshal(recorder.Body.Bytes(), &history)).Should(Succeed())
			Expect(history).Should(BeEmpty())
		})

		It("should return history items when user has solved problems", func() {
			mock.ExpectQuery(`SELECT id, user_id, problem_text, answer_text, created_at FROM history WHERE user_id = \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
				WithArgs(testUserID, 100, 0).
				WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "problem_text", "answer_text", "created_at"}).
					AddRow(1, testUserID, "2+2", "4", "2024-01-01").
					AddRow(2, testUserID, "10*5", "50", "2024-01-02"))

			server.HistoryHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusOK))

			var history []HistoryItem
			Expect(json.Unmarshal(recorder.Body.Bytes(), &history)).Should(Succeed())
			Expect(history).Should(HaveLen(2))
			Expect(history[0].ProblemText).Should(Equal("2+2"))
			Expect(history[1].ProblemText).Should(Equal("10*5"))
		})

		It("should use default limit of 100 when no limit parameter provided", func() {
			req = httptest.NewRequest(http.MethodGet, "/history", nil)
			ctx := context.WithValue(req.Context(), userIDKey, testUserID)
			req = req.WithContext(ctx)

			mock.ExpectQuery(`SELECT id, user_id, problem_text, answer_text, created_at FROM history WHERE user_id = \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
				WithArgs(testUserID, 100, 0).
				WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "problem_text", "answer_text", "created_at"}))

			server.HistoryHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusOK))
		})
	})

	When("testing pagination", func() {
		It("should respect custom limit parameter", func() {
			req = httptest.NewRequest(http.MethodGet, "/history?limit=50", nil)
			ctx := context.WithValue(req.Context(), userIDKey, testUserID)
			req = req.WithContext(ctx)

			mock.ExpectQuery(`SELECT id, user_id, problem_text, answer_text, created_at FROM history WHERE user_id = \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
				WithArgs(testUserID, 50, 0).
				WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "problem_text", "answer_text", "created_at"}))

			server.HistoryHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusOK))
		})

		It("should respect offset parameter for pagination", func() {
			req = httptest.NewRequest(http.MethodGet, "/history?limit=20&offset=40", nil)
			ctx := context.WithValue(req.Context(), userIDKey, testUserID)
			req = req.WithContext(ctx)

			mock.ExpectQuery(`SELECT id, user_id, problem_text, answer_text, created_at FROM history WHERE user_id = \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
				WithArgs(testUserID, 20, 40).
				WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "problem_text", "answer_text", "created_at"}))

			server.HistoryHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusOK))
		})

		It("should return 400 Bad Request for invalid limit parameter", func() {
			req = httptest.NewRequest(http.MethodGet, "/history?limit=notanumber", nil)
			ctx := context.WithValue(req.Context(), userIDKey, testUserID)
			req = req.WithContext(ctx)

			server.HistoryHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusBadRequest))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("invalid limit parameter"))
		})

		It("should return 400 Bad Request for invalid offset parameter", func() {
			req = httptest.NewRequest(http.MethodGet, "/history?offset=abc", nil)
			ctx := context.WithValue(req.Context(), userIDKey, testUserID)
			req = req.WithContext(ctx)

			server.HistoryHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusBadRequest))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("invalid offset parameter"))
		})
	})

	When("testing authorization failures", func() {
		It("should return 401 Unauthorized when user ID not in context", func() {
			req = httptest.NewRequest(http.MethodGet, "/history", nil)

			server.HistoryHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusUnauthorized))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("unauthorized"))
		})
	})

	When("testing database error scenarios", func() {
		It("should return 500 Internal Server Error when database query fails", func() {
			mock.ExpectQuery(`SELECT id, user_id, problem_text, answer_text, created_at FROM history WHERE user_id = \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
				WillReturnError(errors.New("database connection lost"))

			server.HistoryHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusInternalServerError))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("failed to retrieve history"))
		})
	})
})
