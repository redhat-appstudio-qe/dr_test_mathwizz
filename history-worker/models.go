package main

// This file defines the data models used by the history-worker service.
// It contains structs for NATS events and database records related to problem-solving history.
//
// The history-worker is a consumer in the event-driven architecture:
//   1. Web-server publishes ProblemSolvedEvent to NATS (asynchronous, fire-and-forget)
//   2. History-worker subscribes to problem_solved topic
//   3. History-worker receives ProblemSolvedEvent via NATS
//   4. History-worker validates and persists event as HistoryItem in database
//
// This design achieves eventual consistency: the problem is solved immediately (synchronous response),
// but history recording happens asynchronously. If the worker is down, events may be lost unless
// NATS JetStream persistence is enabled.

import "time"

// ProblemSolvedEvent represents an event published to NATS when a user solves a math problem.
// This event is published by the web-server's SolveHandler after successfully evaluating
// a mathematical expression, and consumed by the history-worker to persist the result.
//
// Fields:
//   - UserID: The authenticated user's database ID who solved the problem (primary key from users table)
//   - ProblemText: The mathematical expression that was solved (e.g., "25+75", "10*5")
//   - AnswerText: The computed answer as a string (e.g., "100", "50")
//
// JSON Structure:
//
//	{
//	    "user_id": 123,
//	    "problem": "25+75",
//	    "answer": "100"
//	}
//
// NATS Topic: "problem_solved" (defined in worker.go:ProblemSolvedSubject)
//
// Event Flow:
//  1. User sends POST /solve with math problem
//  2. Web-server calls SolveMath() to evaluate expression
//  3. Web-server responds immediately with answer (synchronous)
//  4. Web-server publishes ProblemSolvedEvent to NATS in goroutine (asynchronous, fire-and-forget)
//  5. History-worker receives event via NATS subscription
//  6. History-worker validates event (validateEvent function)
//  7. History-worker persists event as HistoryItem in database
//
// Validation Rules (enforced by validateEvent in worker.go):
//   - UserID must be greater than 0
//   - ProblemText must not be empty
//   - AnswerText must not be empty
//
// Delivery Guarantees:
//   - **At-most-once delivery**: If worker is down or processing fails, event is lost
//   - No retry mechanism in web-server publisher
//   - No persistent queue (unless NATS JetStream is configured)
//   - Database failures result in lost events (logged but not recovered)
//
// Example Usage:
//
//	// Publishing (in web-server/handlers.go)
//	event := ProblemSolvedEvent{
//	    UserID:      userID,
//	    ProblemText: "25+75",
//	    AnswerText:  "100",
//	}
//	data, _ := json.Marshal(event)
//	nc.Publish("problem_solved", data)
//
//	// Consuming (in history-worker/worker.go)
//	nc.Subscribe("problem_solved", func(msg *nats.Msg) {
//	    var event ProblemSolvedEvent
//	    json.Unmarshal(msg.Data, &event)
//	    // Validate and persist
//	})
//
// Production Improvements Needed:
//   - Add maximum length validation (ProblemText: 1000 chars, AnswerText: 100 chars)
//   - Add timestamp field for when problem was solved
//   - Consider using NATS JetStream for persistent queue and at-least-once delivery
//   - Add event versioning/schema versioning for backward compatibility
//   - Add correlation ID for tracing events from request to persistence
type ProblemSolvedEvent struct {
	// UserID is the authenticated user's database ID who solved the problem.
	// Must be > 0 (validated by validateEvent).
	// References users.id in PostgreSQL.
	UserID int `json:"user_id"`

	// ProblemText is the mathematical expression that was solved.
	// Examples: "25+75", "10*5", "100/4"
	// Must not be empty (validated by validateEvent).
	ProblemText string `json:"problem"`

	// AnswerText is the computed answer as a string.
	// Examples: "100", "50", "25"
	// Must not be empty (validated by validateEvent).
	AnswerText string `json:"answer"`
}

// HistoryItem represents a math problem solving record stored in the PostgreSQL database.
// This struct is used when persisting ProblemSolvedEvent data to the history table.
//
// Database Table: history
// Table Schema:
//
//	CREATE TABLE history (
//	    id SERIAL PRIMARY KEY,
//	    user_id INTEGER NOT NULL REFERENCES users(id),
//	    problem_text VARCHAR(255) NOT NULL,
//	    answer_text VARCHAR(50) NOT NULL,
//	    created_at TIMESTAMP NOT NULL DEFAULT NOW()
//	);
//
// Fields:
//   - ID: Auto-incremented primary key (SERIAL in PostgreSQL)
//   - UserID: Foreign key referencing users.id
//   - ProblemText: The mathematical expression (VARCHAR(255) in database)
//   - AnswerText: The computed answer (VARCHAR(50) in database)
//   - CreatedAt: Timestamp when the record was inserted (set by CreateHistoryItem)
//
// Relationship to ProblemSolvedEvent:
//
//	ProblemSolvedEvent (NATS) → validated → persisted as → HistoryItem (PostgreSQL)
//
// Usage Context:
//   - Created by CreateHistoryItem() in database.go when processing events
//   - Retrieved by GetHistoryForUser() in web-server/database.go for GET /history endpoint
//   - Returned to frontend as JSON array
//
// JSON Representation (when returned by GET /history):
//
//	{
//	    "id": 42,
//	    "user_id": 123,
//	    "problem": "25+75",
//	    "answer": "100",
//	    "created_at": "2024-01-15T10:30:00Z"
//	}
//
// Database Constraints:
//   - user_id: NOT NULL, must reference existing user
//   - problem_text: VARCHAR(255), NOT NULL (255 char limit in schema)
//   - answer_text: VARCHAR(50), NOT NULL (50 char limit in schema)
//   - created_at: NOT NULL, defaults to NOW()
//
// Example Usage:
//
//	// Creating history item from event
//	err := CreateHistoryItem(db, event.UserID, event.ProblemText, event.AnswerText)
//
//	// Retrieving history (in web-server)
//	history, err := GetHistoryForUser(db, userID, 100, 0)
//	// Returns: []HistoryItem (up to 100 items, starting from offset 0)
//
// Production Improvements:
//   - Add validation before insert (length checks)
//   - Add index on user_id for query performance
//   - Implement pagination for history retrieval
//   - Consider partitioning by user_id or date for large datasets
type HistoryItem struct {
	// ID is the auto-incremented primary key from the history table.
	// Generated by PostgreSQL SERIAL type.
	ID int `json:"id"`

	// UserID is the foreign key referencing the user who solved the problem.
	// References users.id, NOT NULL constraint in database.
	UserID int `json:"user_id"`

	// ProblemText is the mathematical expression that was solved.
	// Stored as VARCHAR(255) in database.
	// Examples: "25+75", "10*5"
	ProblemText string `json:"problem"`

	// AnswerText is the computed answer as a string.
	// Stored as VARCHAR(50) in database.
	// Examples: "100", "50"
	AnswerText string `json:"answer"`

	// CreatedAt is the timestamp when the record was inserted.
	// Set by CreateHistoryItem using time.Now().
	// Stored as TIMESTAMP in database.
	CreatedAt time.Time `json:"created_at"`
}
