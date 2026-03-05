package main

// Unit tests for helper functions - tests respondJSON, respondError, and setAuthCookies
// Covers JSON encoding, error formatting, and cookie creation with security settings

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

var _ = Describe("Helper Functions", func() {
	var (
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

		recorder = httptest.NewRecorder()
	})

	AfterEach(func() {
		mockDB.Close()
		os.Unsetenv("JWT_SECRET")
		jwtSecret = nil
	})

	Describe("respondJSON", func() {
		When("testing normal JSON encoding", func() {
			It("should encode struct as JSON with correct status code", func() {
				data := map[string]string{"key": "value"}

				respondJSON(recorder, data, http.StatusOK)

				Expect(recorder.Code).Should(Equal(http.StatusOK))
				Expect(recorder.Header().Get("Content-Type")).Should(Equal("application/json"))

				var response map[string]string
				Expect(json.Unmarshal(recorder.Body.Bytes(), &response)).Should(Succeed())
				Expect(response["key"]).Should(Equal("value"))
			})

			It("should set Content-Type header to application/json", func() {
				respondJSON(recorder, map[string]int{"count": 42}, http.StatusCreated)

				Expect(recorder.Header().Get("Content-Type")).Should(Equal("application/json"))
			})

			It("should encode array as JSON", func() {
				data := []string{"item1", "item2"}

				respondJSON(recorder, data, http.StatusOK)

				Expect(recorder.Code).Should(Equal(http.StatusOK))

				var response []string
				Expect(json.Unmarshal(recorder.Body.Bytes(), &response)).Should(Succeed())
				Expect(response).Should(Equal(data))
			})
		})

		When("testing various HTTP status codes", func() {
			It("should return 201 Created when specified", func() {
				respondJSON(recorder, map[string]string{"result": "created"}, http.StatusCreated)

				Expect(recorder.Code).Should(Equal(http.StatusCreated))
			})

			It("should return 500 Internal Server Error when specified", func() {
				respondJSON(recorder, map[string]string{"error": "something went wrong"}, http.StatusInternalServerError)

				Expect(recorder.Code).Should(Equal(http.StatusInternalServerError))
			})
		})
	})

	Describe("respondError", func() {
		When("testing error responses", func() {
			It("should return ErrorResponse JSON with message", func() {
				respondError(recorder, "test error message", http.StatusBadRequest)

				Expect(recorder.Code).Should(Equal(http.StatusBadRequest))

				var errResp ErrorResponse
				Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
				Expect(errResp.Error).Should(Equal("test error message"))
			})

			It("should set Content-Type to application/json", func() {
				respondError(recorder, "error", http.StatusInternalServerError)

				Expect(recorder.Header().Get("Content-Type")).Should(Equal("application/json"))
			})

			It("should return 401 Unauthorized when specified", func() {
				respondError(recorder, "unauthorized access", http.StatusUnauthorized)

				Expect(recorder.Code).Should(Equal(http.StatusUnauthorized))

				var errResp ErrorResponse
				Expect(json.Unmarshal(recorder.Body.Bytes(), &errResp)).Should(Succeed())
				Expect(errResp.Error).Should(Equal("unauthorized access"))
			})
		})
	})

	Describe("setAuthCookies", func() {
		When("testing cookie creation", func() {
			It("should set both authToken and sessionActive cookies", func() {
				testToken := "test.jwt.token"

				setAuthCookies(recorder, testToken)

				cookies := recorder.Result().Cookies()
				Expect(cookies).Should(HaveLen(2))

				cookieMap := make(map[string]*http.Cookie)
				for _, cookie := range cookies {
					cookieMap[cookie.Name] = cookie
				}

				Expect(cookieMap).Should(HaveKey("authToken"))
				Expect(cookieMap).Should(HaveKey("sessionActive"))
			})

			It("should set authToken as httpOnly cookie with JWT value", func() {
				testToken := "test.jwt.token.value"

				setAuthCookies(recorder, testToken)

				cookies := recorder.Result().Cookies()
				var authTokenCookie *http.Cookie
				for _, cookie := range cookies {
					if cookie.Name == "authToken" {
						authTokenCookie = cookie
						break
					}
				}

				Expect(authTokenCookie).ShouldNot(BeNil())
				Expect(authTokenCookie.Value).Should(Equal(testToken))
				Expect(authTokenCookie.HttpOnly).Should(BeTrue())
				Expect(authTokenCookie.Path).Should(Equal("/"))
				Expect(authTokenCookie.MaxAge).Should(Equal(86400))
				Expect(authTokenCookie.SameSite).Should(Equal(http.SameSiteStrictMode))
			})

			It("should set sessionActive as non-httpOnly cookie with true value", func() {
				setAuthCookies(recorder, "token")

				cookies := recorder.Result().Cookies()
				var sessionCookie *http.Cookie
				for _, cookie := range cookies {
					if cookie.Name == "sessionActive" {
						sessionCookie = cookie
						break
					}
				}

				Expect(sessionCookie).ShouldNot(BeNil())
				Expect(sessionCookie.Value).Should(Equal("true"))
				Expect(sessionCookie.HttpOnly).Should(BeFalse())
				Expect(sessionCookie.Path).Should(Equal("/"))
				Expect(sessionCookie.MaxAge).Should(Equal(86400))
				Expect(sessionCookie.SameSite).Should(Equal(http.SameSiteStrictMode))
			})

			It("should set cookie MaxAge to 24 hours (86400 seconds)", func() {
				setAuthCookies(recorder, "token")

				cookies := recorder.Result().Cookies()
				for _, cookie := range cookies {
					Expect(cookie.MaxAge).Should(Equal(86400))
				}
			})

			It("should set SameSite to Strict mode for CSRF protection", func() {
				setAuthCookies(recorder, "token")

				cookies := recorder.Result().Cookies()
				for _, cookie := range cookies {
					Expect(cookie.SameSite).Should(Equal(http.SameSiteStrictMode))
				}
			})

			It("should set Secure flag to false for development", func() {
				setAuthCookies(recorder, "token")

				cookies := recorder.Result().Cookies()
				for _, cookie := range cookies {
					Expect(cookie.Secure).Should(BeFalse())
				}
			})
		})
	})
})
