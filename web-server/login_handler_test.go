package main

// Unit tests for LoginHandler - tests user login/authentication endpoint
// Covers happy path, validation failures, authentication failures, and edge cases

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/bcrypt"
)

var _ = Describe("LoginHandler", func() {
	var (
		server       *Server
		mockDB       *sql.DB
		mock         sqlmock.Sqlmock
		recorder     *httptest.ResponseRecorder
		natsConn     *nats.Conn
		testSecret   = "test-secret-key-at-least-32-characters-long-for-jwt"
		testEmail    = "test@example.com"
		testPassword = "password123"
		testHash     string
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

		// Generate bcrypt hash for test password
		hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
		Expect(err).ShouldNot(HaveOccurred())
		testHash = string(hash)
	})

	AfterEach(func() {
		mockDB.Close()
		os.Unsetenv("JWT_SECRET")
		jwtSecret = nil
	})

	When("testing happy path scenarios", func() {
		It("should successfully login with valid credentials", func() {
			reqBody := `{"email":"test@example.com","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(reqBody))

			mock.ExpectQuery(`SELECT id, email, password_hash, created_at FROM users WHERE email = \$1`).
				WithArgs(testEmail).
				WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "created_at"}).
					AddRow(1, testEmail, testHash, "2024-01-01"))

			server.LoginHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusOK))

			var response AuthResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &response)).Should(Succeed())
			Expect(response.Email).Should(Equal(testEmail))
			Expect(response.Token).ShouldNot(BeEmpty())

			cookies := recorder.Result().Cookies()
			Expect(cookies).Should(HaveLen(2))
		})

		It("should return 200 OK with token on successful authentication", func() {
			reqBody := `{"email":"user@test.com","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(reqBody))

			hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
			mock.ExpectQuery(`SELECT id, email, password_hash, created_at FROM users WHERE email = \$1`).
				WithArgs("user@test.com").
				WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "created_at"}).
					AddRow(42, "user@test.com", string(hash), "2024-01-01"))

			server.LoginHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusOK))
			Expect(recorder.Header().Get("Content-Type")).Should(Equal("application/json"))
		})
	})

	When("testing request validation failures", func() {
		It("should return 400 Bad Request for malformed JSON", func() {
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{invalid`))

			server.LoginHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusBadRequest))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("invalid request body"))
		})

		It("should return 400 Bad Request when email is empty", func() {
			reqBody := `{"email":"","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(reqBody))

			server.LoginHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusBadRequest))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("email and password are required"))
		})

		It("should return 400 Bad Request when password is empty", func() {
			reqBody := `{"email":"test@example.com","password":""}`
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(reqBody))

			server.LoginHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusBadRequest))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("email and password are required"))
		})
	})

	When("testing authentication failures", func() {
		It("should return 401 Unauthorized when user does not exist", func() {
			reqBody := `{"email":"nonexistent@example.com","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(reqBody))

			mock.ExpectQuery("SELECT id, email, password_hash, created_at FROM users WHERE email = \\$1").
				WithArgs("nonexistent@example.com").
				WillReturnError(sql.ErrNoRows)

			server.LoginHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusUnauthorized))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("invalid credentials"))
		})

		It("should return 401 Unauthorized when password is incorrect", func() {
			reqBody := `{"email":"test@example.com","password":"wrongpassword"}`
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(reqBody))

			mock.ExpectQuery("SELECT id, email, password_hash, created_at FROM users WHERE email = \\$1").
				WithArgs(testEmail).
				WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "created_at"}).
					AddRow(1, testEmail, testHash, "2024-01-01"))

			server.LoginHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusUnauthorized))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("invalid credentials"))
		})

		It("should return 401 when database query fails", func() {
			reqBody := `{"email":"test@example.com","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(reqBody))

			mock.ExpectQuery("SELECT id, email, password_hash, created_at FROM users WHERE email = \\$1").
				WillReturnError(errors.New("database error"))

			server.LoginHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusUnauthorized))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("invalid credentials"))
		})
	})

	When("testing edge cases", func() {
		It("should handle case-sensitive email comparison", func() {
			reqBody := `{"email":"Test@Example.com","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(reqBody))

			mock.ExpectQuery("SELECT id, email, password_hash, created_at FROM users WHERE email = \\$1").
				WithArgs("Test@Example.com").
				WillReturnError(sql.ErrNoRows)

			server.LoginHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusUnauthorized))
		})
	})
})
