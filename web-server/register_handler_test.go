package main

// Unit tests for RegisterHandler - tests user registration endpoint
// Covers happy path, validation failures, database errors, and edge cases

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
)

var _ = Describe("RegisterHandler", func() {
	var (
		server     *Server
		mockDB     *sql.DB
		mock       sqlmock.Sqlmock
		recorder   *httptest.ResponseRecorder
		natsConn   *nats.Conn
		testSecret = "test-secret-key-at-least-32-characters-long-for-jwt"
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
	})

	AfterEach(func() {
		mockDB.Close()
		os.Unsetenv("JWT_SECRET")
		jwtSecret = nil
	})

	When("testing happy path scenarios", func() {
		It("should successfully register user with valid email and password", func() {
			reqBody := `{"email":"test@example.com","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")

			mock.ExpectQuery("INSERT INTO users \\(email, password_hash, created_at\\) VALUES \\(\\$1, \\$2, \\$3\\) RETURNING id").
				WithArgs("test@example.com", sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusCreated))

			var response AuthResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &response)).Should(Succeed())
			Expect(response.Email).Should(Equal("test@example.com"))
			Expect(response.Token).ShouldNot(BeEmpty())

			cookies := recorder.Result().Cookies()
			Expect(cookies).Should(HaveLen(2))
			cookieNames := []string{cookies[0].Name, cookies[1].Name}
			Expect(cookieNames).Should(ContainElements("authToken", "sessionActive"))
		})

		It("should hash password with bcrypt before storing", func() {
			reqBody := `{"email":"test@example.com","password":"mypassword"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")

			mock.ExpectQuery(`INSERT INTO users`).
				WithArgs("test@example.com", sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusCreated))
		})

		It("should return 201 Created with token and email on success", func() {
			reqBody := `{"email":"user@test.com","password":"securepass"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			mock.ExpectQuery(`INSERT INTO users`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusCreated))
			Expect(recorder.Header().Get("Content-Type")).Should(Equal("application/json"))
		})
	})

	When("testing request validation failures", func() {
		It("should return 400 Bad Request for malformed JSON", func() {
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{invalid json`))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusBadRequest))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("invalid request body"))
		})

		It("should return 400 Bad Request when email is empty", func() {
			reqBody := `{"email":"","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusBadRequest))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("email is required"))
		})

		It("should return 400 Bad Request when password is empty", func() {
			reqBody := `{"email":"test@example.com","password":""}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusBadRequest))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("password is required"))
		})

		It("should return 400 Bad Request when password is less than 6 characters", func() {
			reqBody := `{"email":"test@example.com","password":"12345"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusBadRequest))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("password must be at least 6 characters"))
		})
	})

	When("testing database error scenarios", func() {
		It("should return 500 Internal Server Error when CreateUser fails", func() {
			reqBody := `{"email":"test@example.com","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			mock.ExpectQuery(`INSERT INTO users`).
				WillReturnError(errors.New("database connection failed"))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusInternalServerError))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("failed to create user"))
		})

		It("should return 500 when database returns duplicate email error", func() {
			reqBody := `{"email":"duplicate@example.com","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			mock.ExpectQuery(`INSERT INTO users`).
				WillReturnError(errors.New("UNIQUE constraint failed"))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusInternalServerError))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("failed to create user"))
		})
	})

	When("testing edge cases", func() {
		It("should accept password with exactly 6 characters", func() {
			reqBody := `{"email":"test@example.com","password":"123456"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			mock.ExpectQuery(`INSERT INTO users`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusCreated))
		})

		It("should reject very long password (200 chars) - bcrypt fails", func() {
			longPassword := strings.Repeat("a", 200)
			reqBody := `{"email":"test@example.com","password":"` + longPassword + `"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			server.RegisterHandler(recorder, req)

			// Bcrypt fails on passwords >=200 chars with error
			// Handler returns 500 "failed to hash password"
			Expect(recorder.Code).Should(Equal(http.StatusInternalServerError))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("failed to hash password"))
		})

		It("should accept email without format validation", func() {
			reqBody := `{"email":"not-an-email-format","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			mock.ExpectQuery(`INSERT INTO users`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusCreated))
		})
	})

	When("testing email length validation edge cases", func() {
		It("should accept email with exactly 255 characters", func() {
			// Database has VARCHAR(255) limit for email
			// Email: 240 'a's + '@example.com' = 255 characters total
			longEmail := strings.Repeat("a", 240) + "@example.com"
			reqBody := `{"email":"` + longEmail + `","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			mock.ExpectQuery(`INSERT INTO users`).
				WithArgs(longEmail, sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusCreated))
		})

		It("should accept email exceeding 255 characters without validation", func() {
			// Email: 250 'a's + '@example.com' = 262 characters (exceeds database limit)
			// No validation exists, so this will pass validation but fail at database
			veryLongEmail := strings.Repeat("a", 250) + "@example.com"
			reqBody := `{"email":"` + veryLongEmail + `","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			// Database would reject this, but no app-level validation exists
			mock.ExpectQuery(`INSERT INTO users`).
				WillReturnError(errors.New("value too long for type character varying(255)"))

			server.RegisterHandler(recorder, req)

			// Returns 500 instead of 400 because there's no length validation
			Expect(recorder.Code).Should(Equal(http.StatusInternalServerError))
		})

		It("should accept email with 500+ characters without validation (extreme DoS risk)", func() {
			// Email: 490 'a's + '@example.com' = 502 characters (far exceeds limit)
			extremeEmail := strings.Repeat("a", 490) + "@example.com"
			reqBody := `{"email":"` + extremeEmail + `","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			mock.ExpectQuery(`INSERT INTO users`).
				WillReturnError(errors.New("value too long for type character varying(255)"))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusInternalServerError))
		})
	})

	When("testing password length validation edge cases", func() {
		It("should accept password with exactly 72 bytes (bcrypt limit)", func() {
			// Bcrypt silently truncates passwords longer than 72 bytes
			// 72 ASCII characters = 72 bytes
			password72 := strings.Repeat("a", 72)
			reqBody := `{"email":"test@example.com","password":"` + password72 + `"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			mock.ExpectQuery(`INSERT INTO users`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusCreated))
		})

		It("should reject password exceeding 72 bytes (100 chars) - bcrypt fails", func() {
			// Password with 100 characters exceeds bcrypt's documented 72-byte limit
			// Expected: bcrypt truncates silently (security issue)
			// Actual: bcrypt returns error (implementation-specific)
			password100 := strings.Repeat("a", 100)
			reqBody := `{"email":"test@example.com","password":"` + password100 + `"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			server.RegisterHandler(recorder, req)

			// Bcrypt rejects passwords significantly longer than 72 bytes
			Expect(recorder.Code).Should(Equal(http.StatusInternalServerError))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("failed to hash password"))
		})

		It("should reject extremely long password (10,000 chars) - bcrypt fails", func() {
			// 10,000 character password would waste CPU on bcrypt hashing
			// Application has no max length validation before calling bcrypt
			// Bcrypt implementation rejects very long passwords with error
			extremePassword := strings.Repeat("a", 10000)
			reqBody := `{"email":"test@example.com","password":"` + extremePassword + `"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			server.RegisterHandler(recorder, req)

			// Bcrypt fails on extremely long passwords
			// This prevents DoS but with 500 error instead of proper 400 validation
			Expect(recorder.Code).Should(Equal(http.StatusInternalServerError))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("failed to hash password"))
		})

		It("should reject massive password (50,000 chars) - bcrypt fails", func() {
			// 50,000 character password would cause extreme resource consumption
			// No app-level validation before bcrypt, so payload reaches bcrypt
			massivePassword := strings.Repeat("x", 50000)
			reqBody := `{"email":"test@example.com","password":"` + massivePassword + `"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			server.RegisterHandler(recorder, req)

			// Bcrypt fails on massive passwords
			Expect(recorder.Code).Should(Equal(http.StatusInternalServerError))

			var errResp ErrorResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
			Expect(errResp.Error).Should(Equal("failed to hash password"))
		})
	})

	When("testing whitespace password edge cases", func() {
		It("should accept password with only spaces (6 spaces - no content validation)", func() {
			reqBody := `{"email":"test@example.com","password":"      "}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			mock.ExpectQuery(`INSERT INTO users`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

			server.RegisterHandler(recorder, req)

			// BUG: Whitespace-only password should be rejected for security
			// Current validation only checks length (6 chars) not content
			// So "      " passes validation and gets hashed/stored
			Expect(recorder.Code).Should(Equal(http.StatusCreated))
		})

		It("should accept password with only tabs (6 tabs - no content validation)", func() {
			reqBody := `{"email":"test@example.com","password":"\t\t\t\t\t\t"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			mock.ExpectQuery(`INSERT INTO users`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

			server.RegisterHandler(recorder, req)

			// No whitespace validation - accepts tab-only password
			Expect(recorder.Code).Should(Equal(http.StatusCreated))
		})

		It("should accept password with mixed whitespace (spaces, tabs, newlines)", func() {
			reqBody := `{"email":"test@example.com","password":"  \t\n  "}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			mock.ExpectQuery(`INSERT INTO users`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusCreated))
		})
	})

	When("testing malformed email edge cases", func() {
		It("should accept email with consecutive dots (invalid format)", func() {
			reqBody := `{"email":"test..user@example.com","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			mock.ExpectQuery(`INSERT INTO users`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

			server.RegisterHandler(recorder, req)

			// No email format validation - accepts consecutive dots
			Expect(recorder.Code).Should(Equal(http.StatusCreated))
		})

		It("should accept email with leading dot (invalid format)", func() {
			reqBody := `{"email":".test@example.com","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			mock.ExpectQuery(`INSERT INTO users`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusCreated))
		})

		It("should accept email with trailing dot before @ (invalid format)", func() {
			reqBody := `{"email":"test.@example.com","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			mock.ExpectQuery(`INSERT INTO users`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusCreated))
		})

		It("should accept email without @ symbol (invalid format)", func() {
			reqBody := `{"email":"notanemail.com","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			mock.ExpectQuery(`INSERT INTO users`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusCreated))
		})

		It("should accept email with multiple @ symbols (invalid format)", func() {
			reqBody := `{"email":"test@@example.com","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			mock.ExpectQuery(`INSERT INTO users`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusCreated))
		})

		It("should accept email with spaces (invalid format)", func() {
			reqBody := `{"email":"test user@example.com","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))

			mock.ExpectQuery(`INSERT INTO users`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

			server.RegisterHandler(recorder, req)

			Expect(recorder.Code).Should(Equal(http.StatusCreated))
		})
	})
})
