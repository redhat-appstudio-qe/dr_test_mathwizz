package main

// Unit tests for data models (ProblemSolvedEvent and HistoryItem)
// Tests verify JSON serialization/deserialization and data contract between web-server and history-worker

import (
	"encoding/json"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Models", func() {
	Describe("ProblemSolvedEvent", func() {
		When("marshaling to JSON", func() {
			It("should produce correct JSON with all required fields", func() {
				event := ProblemSolvedEvent{
					UserID:      123,
					ProblemText: "25+75",
					AnswerText:  "100",
				}

				data, err := json.Marshal(event)

				Expect(err).ShouldNot(HaveOccurred())

				// Verify JSON is valid and contains all fields
				var result map[string]interface{}
				err = json.Unmarshal(data, &result)
				Expect(err).ShouldNot(HaveOccurred())

				Expect(result["user_id"]).Should(Equal(float64(123)))
				Expect(result["problem"]).Should(Equal("25+75"))
				Expect(result["answer"]).Should(Equal("100"))
			})

			It("should use correct JSON field names matching web-server contract", func() {
				event := ProblemSolvedEvent{
					UserID:      42,
					ProblemText: "10*5",
					AnswerText:  "50",
				}

				data, err := json.Marshal(event)
				Expect(err).ShouldNot(HaveOccurred())

				jsonString := string(data)

				// Verify field names match NATS contract (web-server publishes these names)
				Expect(jsonString).Should(ContainSubstring(`"user_id"`),
					"JSON must contain 'user_id' field (not 'UserID')")
				Expect(jsonString).Should(ContainSubstring(`"problem"`),
					"JSON must contain 'problem' field (not 'ProblemText')")
				Expect(jsonString).Should(ContainSubstring(`"answer"`),
					"JSON must contain 'answer' field (not 'AnswerText')")

				// Verify it does NOT contain struct field names
				Expect(jsonString).ShouldNot(ContainSubstring(`"UserID"`))
				Expect(jsonString).ShouldNot(ContainSubstring(`"ProblemText"`))
				Expect(jsonString).ShouldNot(ContainSubstring(`"AnswerText"`))
			})

			It("should handle special characters and escape them correctly", func() {
				event := ProblemSolvedEvent{
					UserID:      1,
					ProblemText: `"quoted" problem with \ backslash`,
					AnswerText:  "special\nchars\ttab",
				}

				data, err := json.Marshal(event)

				Expect(err).ShouldNot(HaveOccurred())

				// JSON should be valid despite special characters
				var result ProblemSolvedEvent
				err = json.Unmarshal(data, &result)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(result.ProblemText).Should(Equal(event.ProblemText))
				Expect(result.AnswerText).Should(Equal(event.AnswerText))
			})

			It("should handle Unicode characters in problem and answer", func() {
				event := ProblemSolvedEvent{
					UserID:      5,
					ProblemText: "2×3+5÷1",
					AnswerText:  "11✓",
				}

				data, err := json.Marshal(event)

				Expect(err).ShouldNot(HaveOccurred())

				var result ProblemSolvedEvent
				err = json.Unmarshal(data, &result)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(result.ProblemText).Should(Equal("2×3+5÷1"))
				Expect(result.AnswerText).Should(Equal("11✓"))
			})

			It("should handle empty strings (though validation should reject them)", func() {
				// This documents current marshaling behavior
				// Note: validateEvent in worker.go should reject empty strings before persistence
				event := ProblemSolvedEvent{
					UserID:      1,
					ProblemText: "",
					AnswerText:  "",
				}

				data, err := json.Marshal(event)

				Expect(err).ShouldNot(HaveOccurred(),
					"Marshaling empty strings should succeed (validation happens separately)")

				jsonString := string(data)
				Expect(jsonString).Should(ContainSubstring(`"problem":""`))
				Expect(jsonString).Should(ContainSubstring(`"answer":""`))
			})

			It("should handle zero UserID (though validation should reject it)", func() {
				// Documents current marshaling behavior
				// Note: validateEvent should reject UserID=0
				event := ProblemSolvedEvent{
					UserID:      0,
					ProblemText: "test",
					AnswerText:  "test",
				}

				data, err := json.Marshal(event)

				Expect(err).ShouldNot(HaveOccurred())

				var result map[string]interface{}
				json.Unmarshal(data, &result)
				Expect(result["user_id"]).Should(Equal(float64(0)))
			})

			It("should handle negative UserID (invalid but marshaling allows it)", func() {
				event := ProblemSolvedEvent{
					UserID:      -42,
					ProblemText: "test",
					AnswerText:  "test",
				}

				data, err := json.Marshal(event)

				Expect(err).ShouldNot(HaveOccurred())

				var result ProblemSolvedEvent
				json.Unmarshal(data, &result)
				Expect(result.UserID).Should(Equal(-42))
			})

			It("should handle very large UserID values", func() {
				event := ProblemSolvedEvent{
					UserID:      999999999,
					ProblemText: "test",
					AnswerText:  "test",
				}

				data, err := json.Marshal(event)

				Expect(err).ShouldNot(HaveOccurred())

				var result ProblemSolvedEvent
				json.Unmarshal(data, &result)
				Expect(result.UserID).Should(Equal(999999999))
			})

			It("should handle long problem text (500 chars at database limit)", func() {
				longProblem := strings.Repeat("1+", 250) // 500 chars total (2 chars × 250)
				event := ProblemSolvedEvent{
					UserID:      1,
					ProblemText: longProblem,
					AnswerText:  "250",
				}

				data, err := json.Marshal(event)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(len(event.ProblemText)).Should(Equal(500))

				var result ProblemSolvedEvent
				json.Unmarshal(data, &result)
				Expect(result.ProblemText).Should(Equal(longProblem))
			})

			It("should handle long answer text (100 chars at database limit)", func() {
				longAnswer := strings.Repeat("9", 100) // 100 chars
				event := ProblemSolvedEvent{
					UserID:      1,
					ProblemText: "test",
					AnswerText:  longAnswer,
				}

				data, err := json.Marshal(event)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(len(event.AnswerText)).Should(Equal(100))

				var result ProblemSolvedEvent
				json.Unmarshal(data, &result)
				Expect(result.AnswerText).Should(Equal(longAnswer))
			})
		})

		When("unmarshaling from JSON", func() {
			It("should successfully parse valid JSON from web-server", func() {
				// This JSON format matches what web-server publishes to NATS
				jsonData := []byte(`{
					"user_id": 456,
					"problem": "50*2",
					"answer": "100"
				}`)

				var event ProblemSolvedEvent
				err := json.Unmarshal(jsonData, &event)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(event.UserID).Should(Equal(456))
				Expect(event.ProblemText).Should(Equal("50*2"))
				Expect(event.AnswerText).Should(Equal("100"))
			})

			It("should handle JSON with extra whitespace", func() {
				jsonData := []byte(`{
					"user_id"  :  789  ,
					"problem"  :  "2+3"  ,
					"answer"   :  "5"
				}`)

				var event ProblemSolvedEvent
				err := json.Unmarshal(jsonData, &event)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(event.UserID).Should(Equal(789))
				Expect(event.ProblemText).Should(Equal("2+3"))
				Expect(event.AnswerText).Should(Equal("5"))
			})

			It("should handle JSON with fields in different order", func() {
				jsonData := []byte(`{
					"answer": "42",
					"user_id": 99,
					"problem": "6*7"
				}`)

				var event ProblemSolvedEvent
				err := json.Unmarshal(jsonData, &event)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(event.UserID).Should(Equal(99))
				Expect(event.ProblemText).Should(Equal("6*7"))
				Expect(event.AnswerText).Should(Equal("42"))
			})

			It("should return error for invalid JSON syntax", func() {
				jsonData := []byte(`{invalid json}`)

				var event ProblemSolvedEvent
				err := json.Unmarshal(jsonData, &event)

				Expect(err).Should(HaveOccurred())
			})

			It("should return error for JSON with wrong field types", func() {
				// user_id should be int, not string
				jsonData := []byte(`{
					"user_id": "not-a-number",
					"problem": "test",
					"answer": "test"
				}`)

				var event ProblemSolvedEvent
				err := json.Unmarshal(jsonData, &event)

				Expect(err).Should(HaveOccurred())
			})

			It("should handle missing optional fields by using zero values", func() {
				// All fields are technically optional in JSON unmarshaling
				// (Go will use zero values if missing)
				jsonData := []byte(`{}`)

				var event ProblemSolvedEvent
				err := json.Unmarshal(jsonData, &event)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(event.UserID).Should(Equal(0))
				Expect(event.ProblemText).Should(Equal(""))
				Expect(event.AnswerText).Should(Equal(""))
			})

			It("should handle JSON with unexpected extra fields gracefully", func() {
				jsonData := []byte(`{
					"user_id": 1,
					"problem": "test",
					"answer": "test",
					"extra_field": "ignored",
					"timestamp": "2024-01-01T00:00:00Z"
				}`)

				var event ProblemSolvedEvent
				err := json.Unmarshal(jsonData, &event)

				// Go's json.Unmarshal ignores unknown fields by default
				Expect(err).ShouldNot(HaveOccurred())
				Expect(event.UserID).Should(Equal(1))
				Expect(event.ProblemText).Should(Equal("test"))
				Expect(event.AnswerText).Should(Equal("test"))
			})
		})

		When("verifying data contract with web-server", func() {
			It("should match the exact JSON structure published by web-server", func() {
				// This test documents the contract between web-server (publisher)
				// and history-worker (consumer)

				// Web-server creates event and marshals it:
				webServerEvent := ProblemSolvedEvent{
					UserID:      123,
					ProblemText: "25+75",
					AnswerText:  "100",
				}
				publishedData, _ := json.Marshal(webServerEvent)

				// History-worker receives NATS message and unmarshals it:
				var workerEvent ProblemSolvedEvent
				err := json.Unmarshal(publishedData, &workerEvent)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(workerEvent).Should(Equal(webServerEvent),
					"Event should survive marshal→unmarshal round trip without data loss")
			})

			It("should maintain field name consistency across services", func() {
				// This verifies the JSON field names defined in struct tags match
				// what both services expect

				event := ProblemSolvedEvent{
					UserID:      1,
					ProblemText: "test",
					AnswerText:  "test",
				}

				data, _ := json.Marshal(event)
				jsonString := string(data)

				// These are the canonical field names in the NATS contract
				Expect(jsonString).Should(MatchRegexp(`"user_id"\s*:\s*1`))
				Expect(jsonString).Should(MatchRegexp(`"problem"\s*:\s*"test"`))
				Expect(jsonString).Should(MatchRegexp(`"answer"\s*:\s*"test"`))
			})
		})
	})

	Describe("HistoryItem", func() {
		When("marshaling to JSON", func() {
			It("should produce correct JSON with all fields", func() {
				timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
				item := HistoryItem{
					ID:          42,
					UserID:      123,
					ProblemText: "25+75",
					AnswerText:  "100",
					CreatedAt:   timestamp,
				}

				data, err := json.Marshal(item)

				Expect(err).ShouldNot(HaveOccurred())

				var result map[string]interface{}
				err = json.Unmarshal(data, &result)
				Expect(err).ShouldNot(HaveOccurred())

				Expect(result["id"]).Should(Equal(float64(42)))
				Expect(result["user_id"]).Should(Equal(float64(123)))
				Expect(result["problem"]).Should(Equal("25+75"))
				Expect(result["answer"]).Should(Equal("100"))
				Expect(result["created_at"]).ShouldNot(BeNil())
			})

			It("should use correct JSON field names for API responses", func() {
				item := HistoryItem{
					ID:          1,
					UserID:      2,
					ProblemText: "test",
					AnswerText:  "test",
					CreatedAt:   time.Now(),
				}

				data, err := json.Marshal(item)
				Expect(err).ShouldNot(HaveOccurred())

				jsonString := string(data)

				// Verify field names match API contract for GET /history endpoint
				Expect(jsonString).Should(ContainSubstring(`"id"`))
				Expect(jsonString).Should(ContainSubstring(`"user_id"`))
				Expect(jsonString).Should(ContainSubstring(`"problem"`))
				Expect(jsonString).Should(ContainSubstring(`"answer"`))
				Expect(jsonString).Should(ContainSubstring(`"created_at"`))
			})

			It("should format created_at timestamp as RFC3339", func() {
				timestamp := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
				item := HistoryItem{
					ID:          1,
					UserID:      1,
					ProblemText: "test",
					AnswerText:  "test",
					CreatedAt:   timestamp,
				}

				data, err := json.Marshal(item)

				Expect(err).ShouldNot(HaveOccurred())

				var result map[string]interface{}
				json.Unmarshal(data, &result)

				// Go's time.Time marshals to RFC3339 format by default
				createdAtStr, ok := result["created_at"].(string)
				Expect(ok).Should(BeTrue(), "created_at should be a string in JSON")
				Expect(createdAtStr).Should(Equal("2024-01-15T10:30:45Z"))
			})

			It("should handle zero time value", func() {
				item := HistoryItem{
					ID:          1,
					UserID:      1,
					ProblemText: "test",
					AnswerText:  "test",
					CreatedAt:   time.Time{}, // zero value
				}

				data, err := json.Marshal(item)

				Expect(err).ShouldNot(HaveOccurred())

				var result map[string]interface{}
				json.Unmarshal(data, &result)

				// Zero time marshals to "0001-01-01T00:00:00Z"
				Expect(result["created_at"]).Should(Equal("0001-01-01T00:00:00Z"))
			})
		})

		When("unmarshaling from JSON", func() {
			It("should successfully parse JSON from database query result", func() {
				jsonData := []byte(`{
					"id": 100,
					"user_id": 200,
					"problem": "10*10",
					"answer": "100",
					"created_at": "2024-01-15T14:30:00Z"
				}`)

				var item HistoryItem
				err := json.Unmarshal(jsonData, &item)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(item.ID).Should(Equal(100))
				Expect(item.UserID).Should(Equal(200))
				Expect(item.ProblemText).Should(Equal("10*10"))
				Expect(item.AnswerText).Should(Equal("100"))

				expectedTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
				Expect(item.CreatedAt).Should(Equal(expectedTime))
			})

			It("should handle different timestamp formats", func() {
				// Go's time.Time can unmarshal various RFC3339 formats
				jsonData := []byte(`{
					"id": 1,
					"user_id": 1,
					"problem": "test",
					"answer": "test",
					"created_at": "2024-01-15T10:30:45.123456Z"
				}`)

				var item HistoryItem
				err := json.Unmarshal(jsonData, &item)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(item.CreatedAt.Year()).Should(Equal(2024))
				Expect(item.CreatedAt.Month()).Should(Equal(time.January))
				Expect(item.CreatedAt.Day()).Should(Equal(15))
			})

			It("should return error for invalid timestamp format", func() {
				jsonData := []byte(`{
					"id": 1,
					"user_id": 1,
					"problem": "test",
					"answer": "test",
					"created_at": "not-a-timestamp"
				}`)

				var item HistoryItem
				err := json.Unmarshal(jsonData, &item)

				Expect(err).Should(HaveOccurred())
			})
		})

		When("verifying round-trip serialization", func() {
			It("should preserve all data through marshal and unmarshal cycle", func() {
				original := HistoryItem{
					ID:          999,
					UserID:      888,
					ProblemText: "complex + expression * 2",
					AnswerText:  "42",
					CreatedAt:   time.Date(2024, 12, 25, 12, 0, 0, 0, time.UTC),
				}

				// Marshal to JSON
				data, err := json.Marshal(original)
				Expect(err).ShouldNot(HaveOccurred())

				// Unmarshal back to struct
				var restored HistoryItem
				err = json.Unmarshal(data, &restored)
				Expect(err).ShouldNot(HaveOccurred())

				// Verify all fields match
				Expect(restored.ID).Should(Equal(original.ID))
				Expect(restored.UserID).Should(Equal(original.UserID))
				Expect(restored.ProblemText).Should(Equal(original.ProblemText))
				Expect(restored.AnswerText).Should(Equal(original.AnswerText))
				Expect(restored.CreatedAt.Equal(original.CreatedAt)).Should(BeTrue())
			})
		})
	})
})
