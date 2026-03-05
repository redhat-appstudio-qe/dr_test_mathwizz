package main

// This file implements the core worker logic for the history-worker service.
// It handles NATS event consumption, validation, and persistence to PostgreSQL.
//
// Architecture:
//   The history-worker is a consumer in the event-driven architecture that processes
//   ProblemSolvedEvent messages published by the web-server. It runs as a separate
//   service/pod and continuously listens for new events on the NATS message queue.
//
// Event Flow:
//   1. Web-server publishes ProblemSolvedEvent to NATS (fire-and-forget)
//   2. NATS routes event to all subscribers of "problem_solved" topic
//   3. StartWorker's subscription handler receives NATS message
//   4. ProcessEvent parses and validates event
//   5. CreateHistoryItem persists to PostgreSQL database
//   6. Success logged, event processing complete
//
// Error Handling:
//   - Parsing errors: Logged, event discarded (malformed JSON)
//   - Validation errors: Logged, event discarded (invalid data)
//   - Database errors: Logged, event discarded (no retry mechanism)
//   - Events are lost on failure (at-most-once delivery)

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
)

// ProblemSolvedSubject is the NATS topic name for problem-solving events.
// This constant defines the pub-sub topic that connects web-server (publisher)
// and history-worker (subscriber).
//
// Topic Name: "problem_solved"
//
// Publisher:
//   - web-server/queue.go:PublishProblemSolved() publishes to this topic
//   - Called from web-server/handlers.go:SolveHandler() in a goroutine
//   - Fire-and-forget semantics (no acknowledgment)
//
// Subscriber:
//   - history-worker/worker.go:StartWorker() subscribes to this topic
//   - Processes events asynchronously as they arrive
//   - Multiple history-worker instances can subscribe (load balancing via NATS queue groups if configured)
//
// Message Format:
//
//	JSON-encoded ProblemSolvedEvent struct:
//	{
//	    "user_id": 123,
//	    "problem": "25+75",
//	    "answer": "100"
//	}
//
// Delivery Guarantees:
//   - **At-most-once**: Events may be lost if worker is down or processing fails
//   - No persistence: NATS core doesn't persist messages (use JetStream for persistence)
//   - No retry: Failed processing results in lost event
//
// Production Considerations:
//   - Consider using NATS JetStream for persistent queue and at-least-once delivery
//   - Consider using queue groups for load balancing across multiple workers
//   - Consider adding dead letter queue for failed events
const (
	ProblemSolvedSubject = "problem_solved"
)

// ParseEvent deserializes a NATS message payload into a ProblemSolvedEvent struct.
// Performs JSON unmarshaling and validates the resulting event structure.
//
// Parameters:
//   - data: Raw NATS message payload (JSON-encoded byte array)
//
// Returns:
//   - *ProblemSolvedEvent: Pointer to parsed and validated event struct
//   - error: Non-nil if JSON is malformed or validation fails
//
// Processing Steps:
//  1. Unmarshal JSON byte array into ProblemSolvedEvent struct
//  2. Call validateEvent to check required fields
//  3. Return validated event or error
//
// JSON Unmarshaling:
//   - Uses encoding/json package (standard library)
//   - Matches JSON field names to struct tags (e.g., "user_id" → UserID)
//   - Handles type conversion (JSON number → Go int, JSON string → Go string)
//   - Returns error if JSON is malformed or fields have wrong types
//
// Validation:
//   - Delegates to validateEvent function for business logic validation
//   - Checks UserID > 0, ProblemText not empty, AnswerText not empty
//
// Error Cases:
//   - Malformed JSON: Returns "failed to unmarshal event: invalid character..."
//   - Missing required field: Unmarshal succeeds but validation fails
//   - Wrong field type: Returns "failed to unmarshal event: json: cannot unmarshal..."
//   - Invalid UserID: Returns "invalid user_id: 0"
//   - Empty ProblemText: Returns "problem_text cannot be empty"
//   - Empty AnswerText: Returns "answer_text cannot be empty"
//
// Example Usage:
//
//	// Valid event
//	data := []byte(`{"user_id":123,"problem":"25+75","answer":"100"}`)
//	event, err := ParseEvent(data)
//	// event.UserID = 123, event.ProblemText = "25+75", event.AnswerText = "100", err = nil
//
//	// Invalid JSON
//	data := []byte(`{invalid json}`)
//	event, err := ParseEvent(data)
//	// event = nil, err = "failed to unmarshal event: invalid character 'i'..."
//
//	// Invalid UserID
//	data := []byte(`{"user_id":0,"problem":"2+2","answer":"4"}`)
//	event, err := ParseEvent(data)
//	// event = nil, err = "invalid user_id: 0"
//
// Call Chain:
//
//	StartWorker → subscription handler → ProcessEvent → ParseEvent → validateEvent
//
// Testing:
//   - Test valid event parsing
//   - Test malformed JSON handling
//   - Test missing/invalid fields
//   - Test extremely long strings (should fail in production with length checks)
func ParseEvent(data []byte) (*ProblemSolvedEvent, error) {
	var event ProblemSolvedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	if err := validateEvent(&event); err != nil {
		return nil, err
	}

	return &event, nil
}

// validateEvent performs business logic validation on a ProblemSolvedEvent.
// Checks that all required fields are present and have valid values.
//
// Parameters:
//   - event: Pointer to ProblemSolvedEvent struct to validate
//
// Returns:
//   - error: Non-nil if any validation rule fails, nil if event is valid
//
// Validation Rules:
//  1. UserID must be greater than 0 (valid database primary key)
//  2. ProblemText must not be empty string
//  3. AnswerText must not be empty string
//
// Validation Logic:
//   - Uses switch statement for clear error messages per validation failure
//   - Checks in order: UserID, ProblemText, AnswerText
//   - Returns immediately on first failure (fail-fast)
//   - Returns nil if all checks pass
//
// Error Messages:
//   - "invalid user_id: %d" - UserID is 0 or negative
//   - "problem_text cannot be empty" - ProblemText is ""
//   - "answer_text cannot be empty" - AnswerText is ""
//
// Example Usage:
//
//	// Valid event
//	event := &ProblemSolvedEvent{UserID: 123, ProblemText: "2+2", AnswerText: "4"}
//	err := validateEvent(event)
//	// err = nil
//
//	// Invalid UserID
//	event := &ProblemSolvedEvent{UserID: 0, ProblemText: "2+2", AnswerText: "4"}
//	err := validateEvent(event)
//	// err = "invalid user_id: 0"
//
//	// Empty ProblemText
//	event := &ProblemSolvedEvent{UserID: 123, ProblemText: "", AnswerText: "4"}
//	err := validateEvent(event)
//	// err = "problem_text cannot be empty"
//
// Why This Matters:
//   - Fail fast: Reject invalid events before database interaction
//   - Clear errors: Help diagnose publisher issues (malformed events)
//   - Resource protection: Prevent database errors from oversized strings
//   - Data integrity: Ensure only valid events are persisted
//
// Testing:
//   - Test all validation rules independently
//   - Test boundary conditions (UserID = 0, 1, -1)
//   - Test empty strings vs whitespace
//   - Test extremely long strings (should add length validation)
func validateEvent(event *ProblemSolvedEvent) error {
	switch {
	case event.UserID <= 0:
		return fmt.Errorf("invalid user_id: %d", event.UserID)
	case event.ProblemText == "":
		return fmt.Errorf("problem_text cannot be empty")
	case event.AnswerText == "":
		return fmt.Errorf("answer_text cannot be empty")
	default:
		return nil
	}
}

// ProcessEvent orchestrates the complete event processing pipeline.
// Parses, validates, and persists a single ProblemSolvedEvent to the database.
//
// Parameters:
//   - db: Database connection (DBConnection interface for testability)
//   - data: Raw NATS message payload (JSON-encoded byte array)
//
// Returns:
//   - error: Non-nil if any step fails (parsing, validation, database insertion)
//
// Processing Pipeline:
//  1. Call ParseEvent to unmarshal JSON and validate event structure
//  2. Call CreateHistoryItem to insert event into database
//  3. Log successful processing with event details
//  4. Return nil on success, error on failure
//
// Error Handling:
//   - Parsing errors: Returned with "failed to parse event: <details>"
//   - Database errors: Returned with "failed to save history item: <details>"
//   - All errors are logged by caller (StartWorker's subscription handler)
//
// Logging:
//   - Success: Logs user_id, problem, and answer for audit trail
//   - Failure: Caller logs error (StartWorker subscription handler)
//   - Log format: "Processed event: user_id=%d, problem=%s, answer=%s"
//
// Example Success Flow:
//
//	data := []byte(`{"user_id":123,"problem":"25+75","answer":"100"}`)
//	err := ProcessEvent(db, data)
//	// Logs: "Processed event: user_id=123, problem=25+75, answer=100"
//	// Returns: nil
//
// Example Failure Scenarios:
//
//	// Malformed JSON
//	data := []byte(`{invalid}`)
//	err := ProcessEvent(db, data)
//	// Returns: "failed to parse event: failed to unmarshal event: ..."
//
//	// Invalid UserID
//	data := []byte(`{"user_id":0,"problem":"2+2","answer":"4"}`)
//	err := ProcessEvent(db, data)
//	// Returns: "failed to parse event: invalid user_id: 0"
//
//	// Database down
//	data := []byte(`{"user_id":123,"problem":"2+2","answer":"4"}`)
//	err := ProcessEvent(db, data)  // with database unavailable
//	// Returns: "failed to save history item: connection refused"
//
// Event Loss Scenarios:
//   - **Worker Down**: If history-worker is not running, events are lost (NATS doesn't persist)
//   - **Database Down**: Events fail to persist, logged but not retried
//   - **Validation Failure**: Invalid events discarded (logged)
//   - **Parse Failure**: Malformed JSON discarded (logged)
//
// Delivery Guarantees:
//   - **At-most-once delivery**: Each event processed maximum one time
//   - No acknowledgment to NATS (fire-and-forget)
//   - No retry on failure
//   - No dead letter queue for failed events
//
// Testing Considerations:
//   - Use DBConnection interface to mock database
//   - Test successful processing
//   - Test parse failures
//   - Test validation failures
//   - Test database failures (connection refused, constraint violations)
//   - Test concurrent event processing (multiple goroutines)
//
// Call Chain:
//
//	StartWorker → NATS subscription handler → ProcessEvent → ParseEvent / CreateHistoryItem
func ProcessEvent(db DBConnection, data []byte) error {
	event, err := ParseEvent(data)
	if err != nil {
		return fmt.Errorf("failed to parse event: %w", err)
	}

	if err := CreateHistoryItem(db, event.UserID, event.ProblemText, event.AnswerText); err != nil {
		return fmt.Errorf("failed to save history item: %w", err)
	}

	log.Printf("Processed event: user_id=%d, problem=%s, answer=%s",
		event.UserID, event.ProblemText, event.AnswerText)

	return nil
}

// NATSSubscriber is an interface for NATS subscription operations.
// This interface enables testing by allowing mock implementations of NATS connections.
//
// The interface wraps the Subscribe method from *nats.Conn, which is used by StartWorker
// to register message handlers for specific NATS subjects.
//
// Implementations:
//   - *nats.Conn: The production NATS client (automatically implements this interface)
//   - Mock implementations: For unit testing (see worker_test.go)
//
// Usage:
//
//	// Production code (main.go)
//	nc, _ := nats.Connect("nats://localhost:4222")
//	StartWorker(db, nc)  // *nats.Conn implements NATSSubscriber
//
//	// Test code (worker_test.go)
//	mockNATS := &MockNATSSubscriber{...}
//	StartWorker(db, mockNATS)  // Mock implements NATSSubscriber
//
// Methods:
//   - Subscribe: Registers a message handler for a NATS subject
//     Parameters:
//   - subject: NATS topic name (e.g., "problem_solved")
//   - handler: Function called when messages arrive
//     Returns:
//   - *nats.Subscription: Subscription handle (can be used to unsubscribe)
//   - error: Non-nil if subscription fails
//
// Testability:
//
//	This interface was introduced to enable unit testing of StartWorker.
//	Without the interface, StartWorker would require a real NATS server for testing.
//	With the interface, tests can use mocks to verify behavior without external dependencies.
type NATSSubscriber interface {
	Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error)
}

// StartWorker initializes the NATS subscription and begins processing events.
// This is the main entry point for the history-worker service.
//
// Parameters:
//   - db: PostgreSQL database connection (used by subscription handler)
//   - nc: NATS connection implementing NATSSubscriber interface
//
// Returns:
//   - error: Non-nil if subscription fails, nil if subscription succeeds
//
// Startup Sequence:
//  1. Log subscription attempt
//  2. Call nc.Subscribe to register message handler
//  3. Handler closure captures db for event processing
//  4. Handler calls ProcessEvent for each received message
//  5. Handler logs errors but continues processing (no crash on single event failure)
//  6. Log successful subscription
//  7. Return nil (worker is now running)
//
// Subscription Behavior:
//   - **Asynchronous**: Handler runs in separate goroutine for each message
//   - **Concurrent**: Multiple messages processed in parallel
//   - **Long-running**: Subscription remains active until process exits
//   - **Fire-and-forget**: No acknowledgment sent to NATS
//
// Handler Function:
//   - Inline anonymous function (closure)
//   - Captures db from outer scope
//   - Receives *nats.Msg with Data field (JSON payload)
//   - Calls ProcessEvent to handle message
//   - Logs errors but does NOT crash worker (resilient to individual failures)
//
// Error Handling:
//   - Subscription failure: Returns error, prevents worker startup
//   - Event processing errors: Logged, worker continues (resilient)
//   - Panics: Recovered and logged, worker continues (resilient)
//
// Logging:
//   - Startup: "Starting worker subscription to subject: problem_solved"
//   - Success: "Worker subscribed successfully"
//   - Per-event errors: "Error processing event: <details>"
//
// Example Startup:
//
//	db, _ := ConnectDB("localhost", "mathwizz", "password", "mathwizz", 5432, "disable")
//	nc, _ := nats.Connect("nats://localhost:4222")
//
//	err := StartWorker(db, nc)
//	if err != nil {
//	    log.Fatalf("Failed to start worker: %v", err)
//	}
//	// Worker now running, processing events as they arrive
//	// Blocks until SIGINT/SIGTERM
//
// Lifecycle Management:
//   - Worker runs indefinitely after successful subscription
//   - Graceful shutdown: main.go should handle SIGINT/SIGTERM and close connections
//   - Ungraceful shutdown: Process killed, in-flight events may be lost
//
// Concurrency:
//   - NATS client spawns goroutine for each message
//   - Multiple ProcessEvent calls can run concurrently
//   - Database connection pool handles concurrent queries safely
//   - No synchronization needed (events are independent)
//
// NATS Queue Groups (Load Balancing):
//
//	For horizontal scaling, use queue groups to distribute events across workers:
//	```go
//	nc.QueueSubscribe(ProblemSolvedSubject, "history-workers", handler)
//	```
//	- All workers in "history-workers" group share event load
//	- Each event delivered to only one worker in the group
//	- Automatic load balancing across worker instances
//
// Testing:
//   - Test successful subscription
//   - Test subscription failure (NATS down)
//   - Test event processing (publish test events)
//   - Test error handling (invalid events)
//   - Test panic recovery (force panic in ProcessEvent)
//   - Test concurrent event processing
//
// Related Functions:
//   - main.go: Calls StartWorker during application startup
//   - ProcessEvent: Called by subscription handler for each event
//   - ParseEvent: Used by ProcessEvent for validation
//   - CreateHistoryItem: Used by ProcessEvent for persistence
func StartWorker(db *sql.DB, nc NATSSubscriber) error {
	log.Printf("Starting worker subscription to subject: %s", ProblemSolvedSubject)

	_, err := nc.Subscribe(ProblemSolvedSubject, func(msg *nats.Msg) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in history-worker event handler: %v", r)
			}
		}()
		if err := ProcessEvent(db, msg.Data); err != nil {
			log.Printf("Error processing event: %v", err)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", ProblemSolvedSubject, err)
	}

	log.Println("Worker subscribed successfully")
	return nil
}
