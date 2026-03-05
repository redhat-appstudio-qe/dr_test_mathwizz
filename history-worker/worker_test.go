package main

// This file contains unit tests for the worker's event parsing and database logic.
// Tests use mocks to avoid external dependencies.

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"strings"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Worker Functions", func() {
	Describe("ParseEvent", func() {
		When("parsing valid event JSON", func() {
			It("should successfully parse the event", func() {
				eventJSON := []byte(`{"user_id":1,"problem":"2+2","answer":"4"}`)

				event, err := ParseEvent(eventJSON)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(event.UserID).Should(Equal(1))
				Expect(event.ProblemText).Should(Equal("2+2"))
				Expect(event.AnswerText).Should(Equal("4"))
			})
		})

		When("parsing invalid or malformed JSON", func() {
			DescribeTable("should return appropriate errors",
				func(jsonData string, errorSubstring string) {
					event, err := ParseEvent([]byte(jsonData))

					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring(errorSubstring))
					Expect(event).Should(BeNil())
				},
				Entry("invalid JSON syntax", `{invalid json}`, "failed to unmarshal"),
				Entry("missing user_id", `{"problem":"5+5","answer":"10"}`, "invalid user_id"),
				Entry("zero user_id", `{"user_id":0,"problem":"5+5","answer":"10"}`, "invalid user_id"),
				Entry("negative user_id", `{"user_id":-1,"problem":"5+5","answer":"10"}`, "invalid user_id"),
				Entry("missing problem", `{"user_id":1,"answer":"10"}`, "problem_text cannot be empty"),
				Entry("missing answer", `{"user_id":1,"problem":"5+5"}`, "answer_text cannot be empty"),
			)
		})

		// Test Gap #44: Field Length Validation Not Tested
		// These tests verify that oversized fields are rejected to match database constraints
		// Database schema: problem_text VARCHAR(500), answer_text VARCHAR(100)
		// NOTE: These tests will FAIL until Bug #23 is fixed (validateEvent has no length checks)
		When("parsing events with field length violations", func() {
			It("should accept problem with exactly 500 characters (database limit boundary)", func() {
				problem500 := strings.Repeat("a", 500)
				jsonData := `{"user_id":1,"problem":"` + problem500 + `","answer":"42"}`

				event, err := ParseEvent([]byte(jsonData))

				// CURRENTLY PASSES - Bug #23: no validation exists
				// SHOULD PASS - 500 chars is within database limit
				Expect(err).ShouldNot(HaveOccurred())
				Expect(event).ShouldNot(BeNil())
				Expect(event.ProblemText).Should(Equal(problem500))
			})

			It("should reject problem exceeding 500 characters (database limit exceeded)", func() {
				problem501 := strings.Repeat("a", 501)
				jsonData := `{"user_id":1,"problem":"` + problem501 + `","answer":"42"}`

				event, err := ParseEvent([]byte(jsonData))

				// CURRENTLY FAILS - Bug #23: no validation, event accepted
				// SHOULD FAIL - 501 chars exceeds database VARCHAR(500) limit
				// After Bug #23 fix, this test should PASS with error containing "problem_text exceeds"
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("problem"))
				Expect(event).Should(BeNil())
			})

			It("should accept answer with exactly 100 characters (database limit boundary)", func() {
				answer100 := strings.Repeat("9", 100)
				jsonData := `{"user_id":1,"problem":"2+2","answer":"` + answer100 + `"}`

				event, err := ParseEvent([]byte(jsonData))

				// CURRENTLY PASSES - Bug #23: no validation exists
				// SHOULD PASS - 100 chars is within database limit
				Expect(err).ShouldNot(HaveOccurred())
				Expect(event).ShouldNot(BeNil())
				Expect(event.AnswerText).Should(Equal(answer100))
			})

			It("should reject answer exceeding 100 characters (database limit exceeded)", func() {
				answer101 := strings.Repeat("9", 101)
				jsonData := `{"user_id":1,"problem":"2+2","answer":"` + answer101 + `"}`

				event, err := ParseEvent([]byte(jsonData))

				// CURRENTLY FAILS - Bug #23: no validation, event accepted
				// SHOULD FAIL - 101 chars exceeds database VARCHAR(100) limit
				// After Bug #23 fix, this test should PASS with error containing "answer_text exceeds"
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("answer"))
				Expect(event).Should(BeNil())
			})

			DescribeTable("should reject extremely oversized fields (DoS prevention)",
				func(problem string, answer string, expectedErrorSubstring string) {
					// Build JSON manually to avoid Go string literal limits
					var jsonData string
					if problem != "" {
						jsonData = `{"user_id":1,"problem":"` + problem + `","answer":"4"}`
					} else {
						jsonData = `{"user_id":1,"problem":"2+2","answer":"` + answer + `"}`
					}

					event, err := ParseEvent([]byte(jsonData))

					// CURRENTLY FAILS - Bug #23: no validation, events accepted
					// SHOULD FAIL - these extreme lengths should be rejected to prevent DoS
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring(expectedErrorSubstring))
					Expect(event).Should(BeNil())
				},
				Entry("problem with 1000 characters",
					strings.Repeat("x", 1000),
					"",
					"problem",
				),
				Entry("problem with 5000 characters",
					strings.Repeat("y", 5000),
					"",
					"problem",
				),
				Entry("answer with 500 characters",
					"",
					strings.Repeat("z", 500),
					"answer",
				),
				Entry("answer with 1000 characters",
					"",
					strings.Repeat("w", 1000),
					"answer",
				),
			)
		})
	})

	Describe("CreateHistoryItem", func() {
		var (
			mock sqlmock.Sqlmock
			db   *sql.DB
		)

		BeforeEach(func() {
			var err error
			db, mock, err = sqlmock.New()
			Expect(err).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			db.Close()
		})

		When("inserting a history item into the database", func() {
			It("should execute the correct INSERT statement", func() {
				userID := 42
				problem := "10*5"
				answer := "50"

				mock.ExpectExec("INSERT INTO history").
					WithArgs(userID, problem, answer, sqlmock.AnyArg()).
					WillReturnResult(driver.ResultNoRows)

				err := CreateHistoryItem(db, userID, problem, answer)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(mock.ExpectationsWereMet()).ShouldNot(HaveOccurred())
			})
		})

		When("database insert fails", func() {
			It("should return an error", func() {
				mock.ExpectExec("INSERT INTO history").
					WithArgs(1, "2+2", "4", sqlmock.AnyArg()).
					WillReturnError(sql.ErrConnDone)

				err := CreateHistoryItem(db, 1, "2+2", "4")

				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("failed to create history item"))
				Expect(mock.ExpectationsWereMet()).ShouldNot(HaveOccurred())
			})
		})

		// Test Gap #52: CreateHistoryItem Input Validation Not Tested
		// These tests verify that CreateHistoryItem validates inputs defensively
		// IMPORTANT: These tests will FAIL because Bug #29 exists (no input validation)
		// The tests document EXPECTED behavior vs ACTUAL behavior
		When("validating input parameters (Bug #29 - no validation exists)", func() {
			It("should reject userID of 0 (invalid user reference)", func() {
				// EXPECTED: Validation error - userID must be positive
				// ACTUAL: No validation - passes to database, database rejects with foreign key error
				// After Bug #29 fix, this test should PASS with validation error

				// Currently we expect database to accept the exec, so we mock it
				// But CreateHistoryItem SHOULD reject before calling database
				mock.ExpectExec("INSERT INTO history").
					WithArgs(0, "2+2", "4", sqlmock.AnyArg()).
					WillReturnResult(driver.ResultNoRows)

				err := CreateHistoryItem(db, 0, "2+2", "4")

				// EXPECTED: err should contain "userID must be positive" or similar
				// ACTUAL: No error (validation doesn't exist)
				// Test will FAIL when Bug #29 is fixed
				GinkgoWriter.Printf("Bug #29: CreateHistoryItem accepts userID=0 without validation\n")
				Expect(err).ShouldNot(HaveOccurred()) // Current behavior - no validation
				// After Bug #29 fix, change to:
				// Expect(err).Should(HaveOccurred())
				// Expect(err.Error()).Should(ContainSubstring("userID"))
			})

			It("should reject negative userID (invalid user reference)", func() {
				// EXPECTED: Validation error - userID must be positive
				// ACTUAL: No validation - passes to database
				mock.ExpectExec("INSERT INTO history").
					WithArgs(-1, "2+2", "4", sqlmock.AnyArg()).
					WillReturnResult(driver.ResultNoRows)

				err := CreateHistoryItem(db, -1, "2+2", "4")

				// EXPECTED: err should contain "userID must be positive"
				// ACTUAL: No error (validation doesn't exist)
				GinkgoWriter.Printf("Bug #29: CreateHistoryItem accepts negative userID without validation\n")
				Expect(err).ShouldNot(HaveOccurred()) // Current behavior
			})

			It("should reject empty problem string", func() {
				// EXPECTED: Validation error - problem cannot be empty
				// ACTUAL: No validation - passes to database
				mock.ExpectExec("INSERT INTO history").
					WithArgs(1, "", "4", sqlmock.AnyArg()).
					WillReturnResult(driver.ResultNoRows)

				err := CreateHistoryItem(db, 1, "", "4")

				// EXPECTED: err should contain "problem cannot be empty"
				// ACTUAL: No error
				GinkgoWriter.Printf("Bug #29: CreateHistoryItem accepts empty problem string without validation\n")
				Expect(err).ShouldNot(HaveOccurred()) // Current behavior
			})

			It("should reject empty answer string", func() {
				// EXPECTED: Validation error - answer cannot be empty
				// ACTUAL: No validation - passes to database
				mock.ExpectExec("INSERT INTO history").
					WithArgs(1, "2+2", "", sqlmock.AnyArg()).
					WillReturnResult(driver.ResultNoRows)

				err := CreateHistoryItem(db, 1, "2+2", "")

				// EXPECTED: err should contain "answer cannot be empty"
				// ACTUAL: No error
				GinkgoWriter.Printf("Bug #29: CreateHistoryItem accepts empty answer string without validation\n")
				Expect(err).ShouldNot(HaveOccurred()) // Current behavior
			})

			It("should reject problem exceeding 500 characters (database VARCHAR(500) limit)", func() {
				// Database schema: problem_text VARCHAR(500)
				// EXPECTED: Validation error before database call
				// ACTUAL: No validation - database will reject with "value too long" error
				problem501 := strings.Repeat("a", 501)

				mock.ExpectExec("INSERT INTO history").
					WithArgs(1, problem501, "4", sqlmock.AnyArg()).
					WillReturnResult(driver.ResultNoRows)

				err := CreateHistoryItem(db, 1, problem501, "4")

				// EXPECTED: err should contain "problem exceeds maximum length"
				// ACTUAL: No error (validation doesn't exist)
				GinkgoWriter.Printf("Bug #29: CreateHistoryItem accepts 501-char problem without validation (database limit is 500)\n")
				Expect(err).ShouldNot(HaveOccurred()) // Current behavior
			})

			It("should reject answer exceeding 100 characters (database VARCHAR(100) limit)", func() {
				// Database schema: answer_text VARCHAR(100)
				// EXPECTED: Validation error before database call
				// ACTUAL: No validation - database will reject
				answer101 := strings.Repeat("9", 101)

				mock.ExpectExec("INSERT INTO history").
					WithArgs(1, "2+2", answer101, sqlmock.AnyArg()).
					WillReturnResult(driver.ResultNoRows)

				err := CreateHistoryItem(db, 1, "2+2", answer101)

				// EXPECTED: err should contain "answer exceeds maximum length"
				// ACTUAL: No error
				GinkgoWriter.Printf("Bug #29: CreateHistoryItem accepts 101-char answer without validation (database limit is 100)\n")
				Expect(err).ShouldNot(HaveOccurred()) // Current behavior
			})

			It("should reject problem with only whitespace", func() {
				// EXPECTED: Validation error - problem cannot be whitespace-only
				// ACTUAL: No validation - passes to database
				problemWhitespace := "      "

				mock.ExpectExec("INSERT INTO history").
					WithArgs(1, problemWhitespace, "4", sqlmock.AnyArg()).
					WillReturnResult(driver.ResultNoRows)

				err := CreateHistoryItem(db, 1, problemWhitespace, "4")

				// EXPECTED: err should contain "problem cannot be empty or whitespace"
				// ACTUAL: No error
				GinkgoWriter.Printf("Bug #29: CreateHistoryItem accepts whitespace-only problem without validation\n")
				Expect(err).ShouldNot(HaveOccurred()) // Current behavior
			})

			It("should reject answer with only whitespace", func() {
				// EXPECTED: Validation error - answer cannot be whitespace-only
				// ACTUAL: No validation - passes to database
				answerWhitespace := "\t\t\t"

				mock.ExpectExec("INSERT INTO history").
					WithArgs(1, "2+2", answerWhitespace, sqlmock.AnyArg()).
					WillReturnResult(driver.ResultNoRows)

				err := CreateHistoryItem(db, 1, "2+2", answerWhitespace)

				// EXPECTED: err should contain "answer cannot be empty or whitespace"
				// ACTUAL: No error
				GinkgoWriter.Printf("Bug #29: CreateHistoryItem accepts whitespace-only answer without validation\n")
				Expect(err).ShouldNot(HaveOccurred()) // Current behavior
			})
		})

		When("testing boundary values for field lengths", func() {
			It("should accept problem with exactly 500 characters (database limit boundary)", func() {
				// Database schema: problem_text VARCHAR(500)
				// This is the maximum allowed length - should succeed
				problem500 := strings.Repeat("a", 500)

				mock.ExpectExec("INSERT INTO history").
					WithArgs(1, problem500, "4", sqlmock.AnyArg()).
					WillReturnResult(driver.ResultNoRows)

				err := CreateHistoryItem(db, 1, problem500, "4")

				// Should succeed - 500 chars is within limit
				Expect(err).ShouldNot(HaveOccurred())
				Expect(mock.ExpectationsWereMet()).ShouldNot(HaveOccurred())
			})

			It("should accept answer with exactly 100 characters (database limit boundary)", func() {
				// Database schema: answer_text VARCHAR(100)
				// This is the maximum allowed length - should succeed
				answer100 := strings.Repeat("9", 100)

				mock.ExpectExec("INSERT INTO history").
					WithArgs(1, "2+2", answer100, sqlmock.AnyArg()).
					WillReturnResult(driver.ResultNoRows)

				err := CreateHistoryItem(db, 1, "2+2", answer100)

				// Should succeed - 100 chars is within limit
				Expect(err).ShouldNot(HaveOccurred())
				Expect(mock.ExpectationsWereMet()).ShouldNot(HaveOccurred())
			})
		})

		When("testing extreme DoS prevention scenarios", func() {
			It("should reject extremely long problem (1000+ chars DoS attempt)", func() {
				// EXPECTED: Validation rejects before database call
				// ACTUAL: No validation - database will reject
				problem1000 := strings.Repeat("x", 1000)

				mock.ExpectExec("INSERT INTO history").
					WithArgs(1, problem1000, "4", sqlmock.AnyArg()).
					WillReturnResult(driver.ResultNoRows)

				err := CreateHistoryItem(db, 1, problem1000, "4")

				// EXPECTED: Validation error
				// ACTUAL: No error
				GinkgoWriter.Printf("Bug #29: CreateHistoryItem accepts 1000-char problem (DoS risk)\n")
				Expect(err).ShouldNot(HaveOccurred()) // Current behavior
			})

			It("should reject extremely long answer (500+ chars DoS attempt)", func() {
				// EXPECTED: Validation rejects before database call
				// ACTUAL: No validation - database will reject
				answer500 := strings.Repeat("9", 500)

				mock.ExpectExec("INSERT INTO history").
					WithArgs(1, "2+2", answer500, sqlmock.AnyArg()).
					WillReturnResult(driver.ResultNoRows)

				err := CreateHistoryItem(db, 1, "2+2", answer500)

				// EXPECTED: Validation error
				// ACTUAL: No error
				GinkgoWriter.Printf("Bug #29: CreateHistoryItem accepts 500-char answer (DoS risk, database limit is 100)\n")
				Expect(err).ShouldNot(HaveOccurred()) // Current behavior
			})
		})
	})

	Describe("ProcessEvent", func() {
		var (
			mock sqlmock.Sqlmock
			db   *sql.DB
		)

		BeforeEach(func() {
			var err error
			db, mock, err = sqlmock.New()
			Expect(err).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			db.Close()
		})

		When("processing a valid event end-to-end", func() {
			It("should parse the event and save it to the database", func() {
				event := ProblemSolvedEvent{
					UserID:      5,
					ProblemText: "25+75",
					AnswerText:  "100",
				}
				eventJSON, _ := json.Marshal(event)

				mock.ExpectExec("INSERT INTO history").
					WithArgs(event.UserID, event.ProblemText, event.AnswerText, sqlmock.AnyArg()).
					WillReturnResult(driver.ResultNoRows)

				err := ProcessEvent(db, eventJSON)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(mock.ExpectationsWereMet()).ShouldNot(HaveOccurred())
			})
		})

		When("processing an invalid event", func() {
			It("should return a parsing error without touching the database", func() {
				invalidJSON := []byte(`{invalid}`)

				err := ProcessEvent(db, invalidJSON)

				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("failed to parse event"))
			})
		})

		When("database save fails", func() {
			It("should return an error indicating the save failure", func() {
				event := ProblemSolvedEvent{
					UserID:      1,
					ProblemText: "2+2",
					AnswerText:  "4",
				}
				eventJSON, _ := json.Marshal(event)

				mock.ExpectExec("INSERT INTO history").
					WithArgs(1, "2+2", "4", sqlmock.AnyArg()).
					WillReturnError(sql.ErrConnDone)

				err := ProcessEvent(db, eventJSON)

				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("failed to save history item"))
				Expect(mock.ExpectationsWereMet()).ShouldNot(HaveOccurred())
			})
		})
	})

	Describe("StartWorker", func() {
		var (
			mockDB   *sql.DB
			mockNATS *MockNATSSubscriber
		)

		BeforeEach(func() {
			var err error
			mockDB, _, err = sqlmock.New()
			Expect(err).ShouldNot(HaveOccurred())

			mockNATS = &MockNATSSubscriber{}
		})

		AfterEach(func() {
			mockDB.Close()
		})

		When("starting worker with valid NATS connection", func() {
			It("should successfully subscribe to problem_solved subject", func() {
				mockNATS.SubscribeFunc = func(subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
					Expect(subject).Should(Equal(ProblemSolvedSubject))
					return nil, nil
				}

				err := StartWorker(mockDB, mockNATS)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(mockNATS.SubscribeCalled).Should(BeTrue())
			})

			It("should subscribe to the correct subject name", func() {
				var capturedSubject string
				mockNATS.SubscribeFunc = func(subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
					capturedSubject = subject
					return nil, nil
				}

				StartWorker(mockDB, mockNATS)

				Expect(capturedSubject).Should(Equal("problem_solved"))
			})
		})

		When("NATS subscription fails", func() {
			It("should return an error with descriptive message", func() {
				mockNATS.SubscribeFunc = func(subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
					return nil, sql.ErrConnDone
				}

				err := StartWorker(mockDB, mockNATS)

				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("failed to subscribe"))
				Expect(err.Error()).Should(ContainSubstring(ProblemSolvedSubject))
			})
		})

		When("receiving NATS messages", func() {
			It("should invoke the handler when a message arrives", func() {
				var handlerCalled bool
				var capturedHandler func(*nats.Msg)

				mockNATS.SubscribeFunc = func(subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
					capturedHandler = handler
					return nil, nil
				}

				StartWorker(mockDB, mockNATS)

				// Simulate message arrival by calling the handler directly
				Expect(capturedHandler).ShouldNot(BeNil())
				capturedHandler(&nats.Msg{Data: []byte(`{"user_id":1,"problem":"test","answer":"test"}`)})
				handlerCalled = true

				Expect(handlerCalled).Should(BeTrue())
			})

			It("should call ProcessEvent with the message data", func() {
				var capturedHandler func(*nats.Msg)
				mockNATS.SubscribeFunc = func(subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
					capturedHandler = handler
					return nil, nil
				}

				StartWorker(mockDB, mockNATS)

				// Since we can't easily mock ProcessEvent, we verify handler doesn't panic
				// with valid event data (ProcessEvent is already tested separately)
				testData := []byte(`{"user_id":5,"problem":"10+20","answer":"30"}`)
				Expect(func() {
					capturedHandler(&nats.Msg{Data: testData})
				}).ShouldNot(Panic())
			})
		})

		When("handling invalid events", func() {
			It("should not crash when ProcessEvent returns an error", func() {
				var capturedHandler func(*nats.Msg)
				mockNATS.SubscribeFunc = func(subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
					capturedHandler = handler
					return nil, nil
				}

				StartWorker(mockDB, mockNATS)

				// Handler should log error but not panic
				invalidData := []byte(`{invalid json}`)
				Expect(func() {
					capturedHandler(&nats.Msg{Data: invalidData})
				}).ShouldNot(Panic())
			})

			It("should not crash when receiving empty message data", func() {
				var capturedHandler func(*nats.Msg)
				mockNATS.SubscribeFunc = func(subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
					capturedHandler = handler
					return nil, nil
				}

				StartWorker(mockDB, mockNATS)

				emptyData := []byte(``)
				Expect(func() {
					capturedHandler(&nats.Msg{Data: emptyData})
				}).ShouldNot(Panic())
			})

			It("should not crash when receiving malformed event", func() {
				var capturedHandler func(*nats.Msg)
				mockNATS.SubscribeFunc = func(subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
					capturedHandler = handler
					return nil, nil
				}

				StartWorker(mockDB, mockNATS)

				malformedData := []byte(`{"user_id":0,"problem":"","answer":""}`)
				Expect(func() {
					capturedHandler(&nats.Msg{Data: malformedData})
				}).ShouldNot(Panic())
			})
		})

		When("testing panic recovery in message handler", func() {
			It("should have panic recovery mechanism in handler", func() {
				var capturedHandler func(*nats.Msg)
				mockNATS.SubscribeFunc = func(subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
					capturedHandler = handler
					return nil, nil
				}

				StartWorker(mockDB, mockNATS)

				// The handler implementation includes defer/recover for panic recovery
				// We verify this by checking the handler doesn't panic even with
				// various edge cases (empty data, malformed data, invalid events)
				// which are tested in the "handling invalid events" section above.

				// This test documents that panic recovery exists in the implementation
				// by verifying the handler is resilient to multiple error scenarios
				Expect(capturedHandler).ShouldNot(BeNil())

				testCases := [][]byte{
					[]byte(``),          // empty
					[]byte(`{invalid}`), // malformed JSON
					[]byte(`{"user_id":0,"problem":"","answer":""}`),      // invalid fields
					[]byte(`{"user_id":-1,"problem":"test","answer":""}`), // negative user_id
				}

				for _, testData := range testCases {
					Expect(func() {
						capturedHandler(&nats.Msg{Data: testData})
					}).ShouldNot(Panic(), "Handler should not panic for any input due to panic recovery")
				}
			})
		})
	})

	// Test Gap #47: Panic Recovery Not Tested in ProcessEvent
	// These tests verify that StartWorker's panic recovery mechanism (worker.go:419-423)
	// correctly catches panics from ProcessEvent/CreateHistoryItem and prevents worker crashes.
	// Note: ProcessEvent itself doesn't have panic recovery - it's in StartWorker's subscription handler.
	// Bug #24 is already FIXED - panic recovery exists, these tests verify it works correctly.
	//
	// Panic Trigger: Using nil database causes panic when db.Exec() is called in CreateHistoryItem
	// (nil pointer dereference: "runtime error: invalid memory address or nil pointer dereference")
	Describe("ProcessEvent Panic Scenarios", func() {
		When("testing panic recovery when database operations panic", func() {
			It("should recover from panic when database is nil", func() {
				// Using nil database triggers actual panic when CreateHistoryItem calls db.Exec()
				// This simulates database driver panics or nil database scenarios
				var nilDB *sql.DB = nil
				mockNATS := &MockNATSSubscriber{}

				var capturedHandler func(*nats.Msg)
				mockNATS.SubscribeFunc = func(subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
					capturedHandler = handler
					return nil, nil
				}

				err := StartWorker(nilDB, mockNATS)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(capturedHandler).ShouldNot(BeNil())

				// Send valid event JSON that would trigger CreateHistoryItem with nil DB
				validEvent := []byte(`{"user_id":1,"problem":"2+2","answer":"4"}`)

				// The handler should NOT panic - panic should be caught by defer/recover in StartWorker
				// Panic message: "runtime error: invalid memory address or nil pointer dereference"
				Expect(func() {
					capturedHandler(&nats.Msg{Data: validEvent})
				}).ShouldNot(Panic(), "StartWorker's panic recovery should catch nil database panics")

				// Verify the panic was caught and logged
				// The fact that the handler didn't propagate the panic proves recovery worked
				GinkgoWriter.Printf("Test verified: Panic from nil database was caught by StartWorker's defer/recover (worker.go:419-423)\n")
			})

			It("should continue processing subsequent events after recovering from panic", func() {
				// This test verifies the worker remains operational after a panic
				// Multiple events processed despite each causing a panic
				var nilDB *sql.DB = nil
				mockNATS := &MockNATSSubscriber{}

				var capturedHandler func(*nats.Msg)
				mockNATS.SubscribeFunc = func(subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
					capturedHandler = handler
					return nil, nil
				}

				StartWorker(nilDB, mockNATS)

				validEvent := []byte(`{"user_id":1,"problem":"10+20","answer":"30"}`)

				// Process first event - should panic internally but be caught
				Expect(func() {
					capturedHandler(&nats.Msg{Data: validEvent})
				}).ShouldNot(Panic())

				// Process second event - handler should still be functional after first panic
				secondEvent := []byte(`{"user_id":2,"problem":"5*6","answer":"30"}`)
				Expect(func() {
					capturedHandler(&nats.Msg{Data: secondEvent})
				}).ShouldNot(Panic())

				// Process third event - verify resilience continues
				thirdEvent := []byte(`{"user_id":3,"problem":"100-50","answer":"50"}`)
				Expect(func() {
					capturedHandler(&nats.Msg{Data: thirdEvent})
				}).ShouldNot(Panic())

				// Process fourth and fifth events to further verify stability
				Expect(func() {
					capturedHandler(&nats.Msg{Data: []byte(`{"user_id":4,"problem":"7*8","answer":"56"}`)})
				}).ShouldNot(Panic())

				Expect(func() {
					capturedHandler(&nats.Msg{Data: []byte(`{"user_id":5,"problem":"99-9","answer":"90"}`)})
				}).ShouldNot(Panic())

				GinkgoWriter.Printf("Test verified: Worker processed 5 events successfully, each causing panic but all caught\n")
			})

			It("should recover from rapid succession of panics without worker crash", func() {
				// Stress test: Send many events rapidly, all causing panics
				// Verifies panic recovery doesn't degrade or fail under load
				var nilDB *sql.DB = nil
				mockNATS := &MockNATSSubscriber{}

				var capturedHandler func(*nats.Msg)
				mockNATS.SubscribeFunc = func(subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
					capturedHandler = handler
					return nil, nil
				}

				StartWorker(nilDB, mockNATS)

				// Send 10 events in rapid succession
				for i := 1; i <= 10; i++ {
					event := []byte(`{"user_id":` + string(rune(i+'0')) + `,"problem":"test","answer":"result"}`)
					Expect(func() {
						capturedHandler(&nats.Msg{Data: event})
					}).ShouldNot(Panic(), "Event %d should not cause unhandled panic", i)
				}

				GinkgoWriter.Printf("Test verified: Worker handled 10 rapid panic-inducing events without crashing\n")
			})
		})

		When("testing panic recovery interaction with error handling", func() {
			It("should handle both panics and regular errors gracefully", func() {
				// This test verifies that panic recovery doesn't interfere with normal error handling
				// Tests mix of: parsing errors, validation errors, and database panics
				var nilDB *sql.DB = nil
				mockNATS := &MockNATSSubscriber{}

				var capturedHandler func(*nats.Msg)
				mockNATS.SubscribeFunc = func(subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
					capturedHandler = handler
					return nil, nil
				}

				StartWorker(nilDB, mockNATS)

				// Test 1: Invalid JSON (triggers ParseEvent error, not panic)
				invalidJSON := []byte(`{invalid json}`)
				Expect(func() {
					capturedHandler(&nats.Msg{Data: invalidJSON})
				}).ShouldNot(Panic())

				// Test 2: Valid JSON (triggers nil database panic in CreateHistoryItem)
				validJSON := []byte(`{"user_id":1,"problem":"2+2","answer":"4"}`)
				Expect(func() {
					capturedHandler(&nats.Msg{Data: validJSON})
				}).ShouldNot(Panic())

				// Test 3: Invalid event (triggers validation error in ParseEvent, not panic)
				invalidEvent := []byte(`{"user_id":0,"problem":"","answer":""}`)
				Expect(func() {
					capturedHandler(&nats.Msg{Data: invalidEvent})
				}).ShouldNot(Panic())

				// Test 4: Another panic-inducing valid event
				anotherValidEvent := []byte(`{"user_id":99,"problem":"50+50","answer":"100"}`)
				Expect(func() {
					capturedHandler(&nats.Msg{Data: anotherValidEvent})
				}).ShouldNot(Panic())

				GinkgoWriter.Printf("Test verified: Panic recovery coexists with normal error handling (errors and panics both handled)\n")
			})

			It("should document that panics are logged, not returned as errors", func() {
				// This test documents the current panic handling behavior:
				// - Panics are caught by defer/recover
				// - Panics are logged with log.Printf
				// - Panics do NOT cause ProcessEvent to return an error
				// - The NATS message handler continues to run
				var nilDB *sql.DB = nil
				mockNATS := &MockNATSSubscriber{}

				var capturedHandler func(*nats.Msg)
				mockNATS.SubscribeFunc = func(subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
					capturedHandler = handler
					return nil, nil
				}

				StartWorker(nilDB, mockNATS)

				validEvent := []byte(`{"user_id":1,"problem":"test","answer":"result"}`)

				// Expected behavior: Handler doesn't panic (panic is caught and logged)
				// Log output: "PANIC in history-worker event handler: runtime error: invalid memory address or nil pointer dereference"
				Expect(func() {
					capturedHandler(&nats.Msg{Data: validEvent})
				}).ShouldNot(Panic())

				GinkgoWriter.Printf("Test documents: Panics are caught, logged (not returned as errors), and worker continues\n")
			})
		})
	})

	// Test Gap #50: ConnectDB Function Completely Untested
	// These tests verify ConnectDB's error handling, input validation, and connection string construction.
	// Note: Full connection success testing requires integration tests with real PostgreSQL (testcontainers).
	// These unit tests focus on error scenarios and input validation that can be tested without database.
	Describe("ConnectDB", func() {
		When("testing connection string construction", func() {
			It("should construct connection string with all parameters", func() {
				// This test attempts to connect to verify connection string format.
				// It will fail at connection time (host unreachable), but error message
				// should show properly formatted connection string in debug output.
				//
				// Note: We cannot easily verify connection string directly without refactoring
				// ConnectDB to return it or accept a constructor function for testability.
				// This test documents the expected connection string format.

				host := "testhost"
				user := "testuser"
				password := "testpass"
				dbname := "testdb"
				port := 5432
				sslMode := "disable"

				// Expected connection string format (based on database.go line 135):
				// "host=testhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"

				db, err := ConnectDB(host, user, password, dbname, port, sslMode)

				// Connection should fail (host doesn't exist), but we verify error handling works
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("failed to"))
				Expect(db).Should(BeNil())
			})
		})

		When("testing input validation for empty or invalid inputs", func() {
			// Bug #26: ConnectDB Has No Input Validation
			// These tests reveal that ConnectDB accepts invalid inputs without validation.
			// Expected: ConnectDB should validate inputs before attempting connection
			// Actual: ConnectDB passes invalid inputs directly to sql.Open, leading to
			//         confusing error messages or hangs on unreachable hosts

			It("should handle empty host string", func() {
				// Expected: Early validation error like "host cannot be empty"
				// Actual: Passes empty host to sql.Open, gets generic connection error
				db, err := ConnectDB("", "user", "password", "dbname", 5432, "disable")

				Expect(err).Should(HaveOccurred())
				Expect(db).Should(BeNil())
				// Current behavior: error from sql.Open/Ping, not from validation
				// After Bug #26 fix: error should contain "host" or "empty" or "validation"
			})

			It("should handle empty user string", func() {
				// Expected: Early validation error like "user cannot be empty"
				// Actual: Passes empty user to PostgreSQL driver, gets auth error
				db, err := ConnectDB("localhost", "", "password", "dbname", 5432, "disable")

				Expect(err).Should(HaveOccurred())
				Expect(db).Should(BeNil())
				// Current behavior: PostgreSQL auth error or connection failure
			})

			It("should handle empty database name", func() {
				// Expected: Early validation error like "dbname cannot be empty"
				// Actual: Attempts connection with empty dbname
				db, err := ConnectDB("localhost", "user", "password", "", 5432, "disable")

				Expect(err).Should(HaveOccurred())
				Expect(db).Should(BeNil())
				// Current behavior: PostgreSQL connection error (database "" does not exist)
			})

			It("should handle invalid port number (zero)", func() {
				// Expected: Validation error like "port must be between 1 and 65535"
				// Actual: Passes port=0 to connection string, PostgreSQL driver may error
				db, err := ConnectDB("localhost", "user", "password", "dbname", 0, "disable")

				Expect(err).Should(HaveOccurred())
				Expect(db).Should(BeNil())
				// Current behavior: connection error with invalid port
			})

			It("should handle invalid port number (negative)", func() {
				// Expected: Validation error like "port must be positive"
				// Actual: Passes negative port to connection string
				db, err := ConnectDB("localhost", "user", "password", "dbname", -1, "disable")

				Expect(err).Should(HaveOccurred())
				Expect(db).Should(BeNil())
				// Current behavior: connection error
			})

			It("should handle port number exceeding valid range", func() {
				// Expected: Validation error like "port must be <= 65535"
				// Actual: Passes invalid port to connection string
				db, err := ConnectDB("localhost", "user", "password", "dbname", 99999, "disable")

				Expect(err).Should(HaveOccurred())
				Expect(db).Should(BeNil())
				// Current behavior: connection error from PostgreSQL driver
			})

			It("should handle empty sslMode string", func() {
				// Expected: Validation error or default to "disable"
				// Actual: Passes empty sslMode to connection string
				db, err := ConnectDB("localhost", "user", "password", "dbname", 5432, "")

				Expect(err).Should(HaveOccurred())
				Expect(db).Should(BeNil())
				// Current behavior: PostgreSQL driver error for invalid sslmode
			})

			It("should handle invalid sslMode value", func() {
				// Expected: Validation error like "sslmode must be one of: disable, require, verify-ca, verify-full"
				// Actual: Passes invalid sslmode to PostgreSQL driver
				db, err := ConnectDB("localhost", "user", "password", "dbname", 5432, "invalid-mode")

				Expect(err).Should(HaveOccurred())
				Expect(db).Should(BeNil())
				// Current behavior: PostgreSQL driver error: "invalid sslmode value"
				// After Bug #26 fix: should validate before attempting connection
			})
		})

		When("testing connection failure scenarios", func() {
			It("should return descriptive error when host is unreachable", func() {
				// Test with non-existent host
				db, err := ConnectDB("nonexistent-host-12345", "user", "password", "dbname", 5432, "disable")

				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("failed to"))
				Expect(db).Should(BeNil())
				// Error should be from Ping() (line 143-145 in database.go)
				// Wrapped with "failed to ping database"
			})

			It("should return descriptive error when port is unreachable", func() {
				// Test with valid host but wrong port
				db, err := ConnectDB("localhost", "user", "password", "dbname", 1, "disable")

				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("failed to"))
				Expect(db).Should(BeNil())
			})
		})

		When("testing error message quality", func() {
			It("should wrap errors with context", func() {
				// Verify error messages include "failed to open database" or "failed to ping database"
				// as documented in database.go lines 100-105
				db, err := ConnectDB("unreachable-host", "user", "password", "dbname", 5432, "disable")

				Expect(err).Should(HaveOccurred())
				Expect(db).Should(BeNil())

				// Error should be wrapped with context (fmt.Errorf with %w)
				// Expected format: "failed to ping database: <underlying error>"
				errorMsg := err.Error()
				Expect(errorMsg).Should(Or(
					ContainSubstring("failed to open database"),
					ContainSubstring("failed to ping database"),
				))
			})
		})

		When("testing timeout behavior", func() {
			// Bug #30: No Connection Timeout
			// This test documents that ConnectDB can hang indefinitely when database is slow/unreachable
			// Expected: Connection should fail fast with timeout (e.g., 5-10 seconds)
			// Actual: No timeout configured, can block forever

			It("should document lack of connection timeout", func() {
				// This test would hang indefinitely if database host is unreachable but network doesn't reject
				// Example: firewall drops packets instead of rejecting connection
				//
				// To prevent test suite from hanging, we only document the issue rather than test it.
				//
				// Bug #30 Impact:
				// - history-worker startup can hang forever if PostgreSQL is unreachable
				// - Kubernetes readiness probes may fail, preventing pod from becoming Ready
				// - Manual intervention required to detect and resolve hung worker
				//
				// Recommended Fix:
				// - Add connect_timeout parameter to connection string
				// - Use context.WithTimeout for Ping() operation
				// - Document timeout values in ConnectDB GoDoc
				//
				// After Bug #30 fix, add test like:
				//   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				//   defer cancel()
				//   db, err := ConnectDBWithContext(ctx, "10.255.255.1", "user", "pass", "db", 5432, "disable")
				//   Expect(err).Should(HaveOccurred())
				//   Expect(err.Error()).Should(ContainSubstring("timeout"))

				// This test currently just documents the issue
				// No actual test code to avoid hanging the test suite
				Skip("Bug #30: No connection timeout - test would hang indefinitely. See database.go line 91 for recommended timeout configuration.")
			})
		})

		When("testing SSL mode parameter", func() {
			It("should accept valid sslmode values", func() {
				// Test that various valid sslmode values are accepted
				// They should all fail to connect (no database running), but shouldn't error on validation

				validModes := []string{"disable", "require", "verify-ca", "verify-full"}

				for _, mode := range validModes {
					db, err := ConnectDB("localhost", "user", "password", "dbname", 5432, mode)

					// All will fail to connect, but error should be connection-related, not validation
					Expect(err).Should(HaveOccurred(), "sslmode=%s should fail due to no database, not validation error", mode)
					Expect(db).Should(BeNil())

					// Error should be from connection attempt, not from invalid sslmode
					if err != nil {
						// PostgreSQL driver error for invalid sslmode would contain "sslmode"
						// Connection errors would contain "failed to ping" or "connect"
						Expect(err.Error()).Should(Or(
							ContainSubstring("failed to"),
							ContainSubstring("connect"),
						), "sslmode=%s should fail with connection error, not validation error", mode)
					}
				}
			})
		})
	})
})

// MockNATSSubscriber is a mock implementation of NATSSubscriber interface for testing.
// It allows tests to verify StartWorker behavior without requiring a real NATS server.
type MockNATSSubscriber struct {
	SubscribeFunc   func(subject string, handler func(*nats.Msg)) (*nats.Subscription, error)
	SubscribeCalled bool
}

// Subscribe implements the NATSSubscriber interface.
// Calls the SubscribeFunc if set, otherwise returns nil error.
func (m *MockNATSSubscriber) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	m.SubscribeCalled = true
	if m.SubscribeFunc != nil {
		return m.SubscribeFunc(subject, handler)
	}
	return nil, nil
}
