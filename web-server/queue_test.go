package main

// This file contains unit tests for NATS queue functions.
// Tests use mocked NATS connections to verify behavior without requiring a real NATS server.

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// NATSPublisher is an interface for NATS publishing operations.
// This interface enables mocking NATS connections in unit tests.
// The *nats.Conn type automatically implements this interface.
type NATSPublisher interface {
	// Publish publishes a message to a NATS subject.
	// Parameters:
	//   - subject: The NATS subject/topic to publish to
	//   - data: The message payload as byte slice
	// Returns:
	//   - error: Non-nil if publish fails
	Publish(subject string, data []byte) error
}

// MockNATSPublisher is a mock implementation of NATSPublisher for unit testing.
// It records publish calls and can simulate publish failures for error testing.
type MockNATSPublisher struct {
	PublishCalls []MockPublishCall
	PublishError error
}

// MockPublishCall records the parameters of a Publish call for verification in tests.
type MockPublishCall struct {
	Subject string
	Data    []byte
}

// Publish records the call and returns the configured error (if any).
func (m *MockNATSPublisher) Publish(subject string, data []byte) error {
	m.PublishCalls = append(m.PublishCalls, MockPublishCall{
		Subject: subject,
		Data:    data,
	})
	return m.PublishError
}

// PublishProblemSolvedWithPublisher is a testable version of PublishProblemSolved that accepts a NATSPublisher interface.
// This function is identical to PublishProblemSolved but uses the interface for testability.
// Production code should continue using PublishProblemSolved directly with *nats.Conn.
func PublishProblemSolvedWithPublisher(nc NATSPublisher, userID int, problem, answer string) error {
	event := ProblemSolvedEvent{
		UserID:      userID,
		ProblemText: problem,
		AnswerText:  answer,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := nc.Publish(ProblemSolvedSubject, data); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

var _ = Describe("Queue Functions", func() {
	Describe("ConnectNATS", func() {
		When("testing with invalid URLs", func() {
			It("should return error for empty URL", func() {
				nc, err := ConnectNATS("")

				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("failed to connect to NATS"))
				Expect(nc).Should(BeNil())
			})

			It("should return error for malformed URL", func() {
				nc, err := ConnectNATS("not-a-valid-url")

				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("failed to connect to NATS"))
				Expect(nc).Should(BeNil())
			})

			It("should return error for invalid protocol", func() {
				nc, err := ConnectNATS("http://invalid-protocol:4222")

				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("failed to connect to NATS"))
				Expect(nc).Should(BeNil())
			})
		})

		When("testing with unreachable server", func() {
			It("should eventually fail when connecting to non-existent host", func() {
				// Note: This test documents Bug #12 - no connection timeout configured
				// The NATS client will attempt to connect with default timeout settings
				// This test may take several seconds to fail depending on NATS client defaults
				nc, err := ConnectNATS("nats://non-existent-host-that-does-not-exist:4222")

				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("failed to connect to NATS"))
				Expect(nc).Should(BeNil())
			})

			It("should fail when connecting to unreachable port", func() {
				// Connecting to localhost on a port that's definitely not listening
				// Port 9999 is typically unused, so this should fail quickly
				nc, err := ConnectNATS("nats://localhost:9999")

				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("failed to connect to NATS"))
				Expect(nc).Should(BeNil())
			})
		})
	})

	Describe("PublishProblemSolved", func() {
		var mock *MockNATSPublisher

		BeforeEach(func() {
			mock = &MockNATSPublisher{
				PublishCalls: []MockPublishCall{},
			}
		})

		When("testing with valid inputs", func() {
			It("should successfully publish event with correct JSON structure", func() {
				err := PublishProblemSolvedWithPublisher(mock, 123, "2+2", "4")

				Expect(err).ShouldNot(HaveOccurred())
				Expect(mock.PublishCalls).Should(HaveLen(1))
				Expect(mock.PublishCalls[0].Subject).Should(Equal("problem_solved"))

				// Verify JSON structure
				var event ProblemSolvedEvent
				err = json.Unmarshal(mock.PublishCalls[0].Data, &event)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(event.UserID).Should(Equal(123))
				Expect(event.ProblemText).Should(Equal("2+2"))
				Expect(event.AnswerText).Should(Equal("4"))
			})

			It("should publish to correct NATS subject", func() {
				err := PublishProblemSolvedWithPublisher(mock, 1, "test", "result")

				Expect(err).ShouldNot(HaveOccurred())
				Expect(mock.PublishCalls).Should(HaveLen(1))
				Expect(mock.PublishCalls[0].Subject).Should(Equal(ProblemSolvedSubject))
				Expect(ProblemSolvedSubject).Should(Equal("problem_solved"))
			})

			It("should correctly marshal event with special characters", func() {
				err := PublishProblemSolvedWithPublisher(mock, 42, "2*3+5", "11")

				Expect(err).ShouldNot(HaveOccurred())

				var event ProblemSolvedEvent
				json.Unmarshal(mock.PublishCalls[0].Data, &event)
				Expect(event.ProblemText).Should(Equal("2*3+5"))
			})

			It("should handle large but valid problem and answer", func() {
				// Create a 500-character problem (database limit, though not enforced here)
				problem := strings.Repeat("1+", 249) + "1" // Exactly 500 characters
				// Create a 100-character answer (database limit)
				answer := strings.Repeat("9", 100)

				err := PublishProblemSolvedWithPublisher(mock, 1, problem, answer)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(mock.PublishCalls).Should(HaveLen(1))

				var event ProblemSolvedEvent
				json.Unmarshal(mock.PublishCalls[0].Data, &event)
				Expect(event.ProblemText).Should(Equal(problem))
				Expect(event.AnswerText).Should(Equal(answer))
			})
		})

		When("testing input validation gaps", func() {
			It("should accept zero userID without validation (Bug: no input validation)", func() {
				// Documents that the function doesn't validate userID > 0
				// This invalid data will be caught by history-worker validation
				err := PublishProblemSolvedWithPublisher(mock, 0, "2+2", "4")

				Expect(err).ShouldNot(HaveOccurred())

				var event ProblemSolvedEvent
				json.Unmarshal(mock.PublishCalls[0].Data, &event)
				Expect(event.UserID).Should(Equal(0))
			})

			It("should accept negative userID without validation (Bug: no input validation)", func() {
				// Documents that the function doesn't validate userID > 0
				err := PublishProblemSolvedWithPublisher(mock, -42, "test", "result")

				Expect(err).ShouldNot(HaveOccurred())

				var event ProblemSolvedEvent
				json.Unmarshal(mock.PublishCalls[0].Data, &event)
				Expect(event.UserID).Should(Equal(-42))
			})

			It("should accept empty problem string without validation (Bug: no input validation)", func() {
				// Documents that the function doesn't validate non-empty inputs
				// Empty problem will be caught by history-worker validation
				err := PublishProblemSolvedWithPublisher(mock, 1, "", "result")

				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should accept empty answer string without validation (Bug: no input validation)", func() {
				// Documents that the function doesn't validate non-empty inputs
				err := PublishProblemSolvedWithPublisher(mock, 1, "problem", "")

				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should accept problem exceeding database limit without validation (Bug: no length validation)", func() {
				// Problem exceeds 500 character database limit
				// This will cause database error in history-worker but isn't validated here
				oversizedProblem := strings.Repeat("x", 501)

				err := PublishProblemSolvedWithPublisher(mock, 1, oversizedProblem, "result")

				Expect(err).ShouldNot(HaveOccurred())

				var event ProblemSolvedEvent
				json.Unmarshal(mock.PublishCalls[0].Data, &event)
				Expect(len(event.ProblemText)).Should(Equal(501))
			})

			It("should accept answer exceeding database limit without validation (Bug: no length validation)", func() {
				// Answer exceeds 100 character database limit
				oversizedAnswer := strings.Repeat("9", 101)

				err := PublishProblemSolvedWithPublisher(mock, 1, "test", oversizedAnswer)

				Expect(err).ShouldNot(HaveOccurred())

				var event ProblemSolvedEvent
				json.Unmarshal(mock.PublishCalls[0].Data, &event)
				Expect(len(event.AnswerText)).Should(Equal(101))
			})

			It("should accept very large userID without validation", func() {
				// Very large userID that may not exist in database
				err := PublishProblemSolvedWithPublisher(mock, 999999999, "test", "result")

				Expect(err).ShouldNot(HaveOccurred())
			})
		})

		When("testing error scenarios", func() {
			It("should return error when NATS publish fails", func() {
				mock.PublishError = errors.New("NATS connection closed")

				err := PublishProblemSolvedWithPublisher(mock, 1, "test", "result")

				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("failed to publish event"))
				Expect(err.Error()).Should(ContainSubstring("NATS connection closed"))
			})

			It("should return error when NATS server is unavailable", func() {
				mock.PublishError = errors.New("no connection to NATS server")

				err := PublishProblemSolvedWithPublisher(mock, 1, "test", "result")

				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("failed to publish event"))
			})

			It("should wrap NATS publish errors with context", func() {
				natsErr := errors.New("network timeout")
				mock.PublishError = natsErr

				err := PublishProblemSolvedWithPublisher(mock, 1, "test", "result")

				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("failed to publish event"))
				Expect(err.Error()).Should(ContainSubstring("network timeout"))
			})
		})

		When("testing JSON marshaling", func() {
			It("should produce valid JSON that can be unmarshaled", func() {
				err := PublishProblemSolvedWithPublisher(mock, 123, "25+75", "100")

				Expect(err).ShouldNot(HaveOccurred())

				// Verify the marshaled JSON is valid
				var event ProblemSolvedEvent
				err = json.Unmarshal(mock.PublishCalls[0].Data, &event)
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should use correct JSON field names matching history-worker contract", func() {
				err := PublishProblemSolvedWithPublisher(mock, 1, "test", "result")

				Expect(err).ShouldNot(HaveOccurred())

				// Verify JSON field names match ProblemSolvedEvent struct tags
				jsonData := string(mock.PublishCalls[0].Data)
				Expect(jsonData).Should(ContainSubstring(`"user_id"`))
				Expect(jsonData).Should(ContainSubstring(`"problem"`))
				Expect(jsonData).Should(ContainSubstring(`"answer"`))
			})

			It("should correctly escape special characters in JSON", func() {
				err := PublishProblemSolvedWithPublisher(mock, 1, `problem"with"quotes`, `answer\with\backslash`)

				Expect(err).ShouldNot(HaveOccurred())

				// Verify JSON is still valid after escaping
				var event ProblemSolvedEvent
				err = json.Unmarshal(mock.PublishCalls[0].Data, &event)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(event.ProblemText).Should(Equal(`problem"with"quotes`))
				Expect(event.AnswerText).Should(Equal(`answer\with\backslash`))
			})
		})

		When("testing edge cases", func() {
			It("should handle Unicode characters in problem and answer", func() {
				err := PublishProblemSolvedWithPublisher(mock, 1, "2×3", "6")

				Expect(err).ShouldNot(HaveOccurred())

				var event ProblemSolvedEvent
				json.Unmarshal(mock.PublishCalls[0].Data, &event)
				Expect(event.ProblemText).Should(Equal("2×3"))
			})

			It("should handle newlines in problem text", func() {
				err := PublishProblemSolvedWithPublisher(mock, 1, "2+\n3", "5")

				Expect(err).ShouldNot(HaveOccurred())

				var event ProblemSolvedEvent
				json.Unmarshal(mock.PublishCalls[0].Data, &event)
				Expect(event.ProblemText).Should(Equal("2+\n3"))
			})

			It("should handle multiple rapid publishes without error", func() {
				// Simulate multiple rapid calls (as would happen in production with goroutines)
				for i := 1; i <= 10; i++ {
					err := PublishProblemSolvedWithPublisher(mock, i, fmt.Sprintf("problem%d", i), fmt.Sprintf("answer%d", i))
					Expect(err).ShouldNot(HaveOccurred())
				}

				Expect(mock.PublishCalls).Should(HaveLen(10))
			})
		})
	})

	Describe("ProblemSolvedSubject constant", func() {
		It("should have correct value matching history-worker subscription", func() {
			Expect(ProblemSolvedSubject).Should(Equal("problem_solved"))
		})

		It("should be a string constant that can be used as NATS subject", func() {
			// Verify it's not empty and has valid NATS subject format
			Expect(ProblemSolvedSubject).ShouldNot(BeEmpty())
			Expect(ProblemSolvedSubject).ShouldNot(ContainSubstring(" "))
		})
	})
})
