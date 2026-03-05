package main

// This file handles all NATS message queue operations.
// It provides functions for connecting to NATS and publishing events.

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
)

// ProblemSolvedSubject is the NATS topic/subject name used for publishing problem-solved events.
// This subject is subscribed to by the history-worker service to persist problem history.
//
// Topic Name: "problem_solved"
// Publisher: web-server (via PublishProblemSolved function)
// Subscriber: history-worker (ProcessEvent function)
// Message Format: JSON-serialized ProblemSolvedEvent struct
const (
	ProblemSolvedSubject = "problem_solved"
)

// ConnectNATS establishes a connection to the NATS message queue server.
// It creates a persistent connection that can be used to publish and subscribe to NATS subjects/topics.
//
// Parameters:
//   - url: The NATS server URL (e.g., "nats://localhost:4222", "nats://nats-service:4222")
//
// Returns:
//   - *nats.Conn: A connection handle for publishing messages and subscribing to subjects. Callers should defer nc.Close()
//   - error: Non-nil if connection fails, with wrapped error details
//
// Connection Behavior:
//   - Attempts to connect to NATS server immediately (synchronous)
//   - Connection is persistent and will auto-reconnect on network issues (NATS default behavior)
//   - Uses NATS client library default settings (reconnect attempts, timeouts, etc.)
//
// Security Considerations:
//   - **NO CONNECTION TIMEOUT**: May hang indefinitely if NATS server is unresponsive
//   - No authentication configured (NATS server is assumed to be in trusted network)
//   - No TLS/encryption (all messages sent in plaintext over network)
//   - URL parameter not validated (empty string, malformed URL accepted)
//
// Error Cases:
//   - NATS server unreachable: Returns "failed to connect to NATS: <details>"
//   - Invalid URL format: Returns "failed to connect to NATS: <details>"
//   - Network timeout: May hang indefinitely (no explicit timeout configured)
//   - Authentication failure: Returns error if NATS server requires auth but none provided
//
// Example Usage:
//
//	// Basic connection
//	nc, err := ConnectNATS("nats://localhost:4222")
//	if err != nil {
//	    log.Fatalf("NATS connection failed: %v", err)
//	}
//	defer nc.Close()
//
//	// Connection is ready for publishing
//	err = PublishProblemSolved(nc, 123, "2+2", "4")
//
// Production Recommendations:
//
//	```go
//	nc, err := nats.Connect(url,
//	    nats.Timeout(10*time.Second),        // Connection timeout
//	    nats.ReconnectWait(2*time.Second),   // Time between reconnect attempts
//	    nats.MaxReconnects(10),              // Maximum reconnect attempts
//	    nats.Secure(),                       // Enable TLS
//	    nats.UserInfo(user, password),       // Authentication
//	    nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
//	        log.Printf("NATS error: %v", err)
//	    }),
//	)
//	```
//
// NATS Features Not Used:
//   - JetStream (persistent message storage)
//   - Request-reply pattern (only pub-sub used)
//   - Queue groups (load balancing subscribers)
//   - Message acknowledgment (fire-and-forget only)
func ConnectNATS(url string) (*nats.Conn, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	return nc, nil
}

// PublishProblemSolved publishes a ProblemSolvedEvent to the NATS message queue for asynchronous processing.
// This event notifies the history-worker service to persist the solved problem to the database.
// This function is called in a goroutine from the solve handler, enabling fire-and-forget asynchronous publishing.
//
// Parameters:
//   - nc: NATS connection handle (must not be nil, not validated)
//   - userID: The ID of the user who solved the problem (not validated)
//   - problem: The mathematical expression that was solved (not validated for length)
//   - answer: The computed result as a string (not validated for length)
//
// Returns:
//   - error: Non-nil if JSON marshaling fails or NATS publish fails, nil if successful
//
// Behavior:
//  1. Creates ProblemSolvedEvent struct with provided parameters
//  2. Marshals event to JSON
//  3. Publishes JSON to "problem_solved" NATS subject
//  4. Returns immediately (fire-and-forget, no delivery guarantee)
//
// Message Flow:
//  1. web-server calls this function from solve handler (in goroutine)
//  2. Event published to NATS subject "problem_solved"
//  3. NATS server receives and queues the message
//  4. history-worker subscription receives the message
//  5. history-worker processes event and writes to database
//
// Delivery Semantics:
//   - **At-most-once delivery**: No acknowledgment, no retry
//   - If NATS server is down, message is lost silently
//   - If no subscribers, message is discarded
//   - If history-worker crashes during processing, event is lost
//   - No guarantee that history will be persisted (eventual consistency)
//
// Security Considerations:
//   - **NO INPUT VALIDATION**: Accepts any values for userID, problem, answer
//   - userID not validated (could be 0, negative, non-existent user)
//   - problem not validated for length (database limit: 500 chars, but not enforced here)
//   - answer not validated for length (database limit: 100 chars, but not enforced here)
//   - Empty strings accepted (will be caught by history-worker validation, but wastes resources)
//   - **NO NIL CHECK**: If nc is nil, causes nil pointer panic crashing the goroutine (and potentially server)
//
// Error Handling:
//   - JSON marshal failure: Returns "failed to marshal event: <details>" (rare, only if struct has unmarshalable types)
//   - NATS publish failure: Returns "failed to publish event: <details>"
//   - Nil connection: Panics with nil pointer dereference (NOT caught, crashes goroutine)
//   - NATS server down: Publish may fail or timeout depending on NATS client settings
//
// Caller Responsibility:
//   - This function is typically called from a goroutine in handlers.go:
//     ```go
//     go func() {
//     if err := PublishProblemSolved(nc, userID, problem, answer); err != nil {
//     log.Printf("Failed to publish event: %v", err)
//     }
//     }()
//     ```
//   - Goroutine should have panic recovery
//   - Errors are logged but don't affect the synchronous /solve response
//
// Example Usage:
//
//	// From solve handler
//	userID := 123
//	problem := "25+75"
//	answer := "100"
//
//	// Publish asynchronously (in separate goroutine)
//	go func() {
//	    err := PublishProblemSolved(nc, userID, problem, answer)
//	    if err != nil {
//	        log.Printf("Failed to publish problem solved event: %v", err)
//	        // Error logged but user already received answer
//	    }
//	}()
//
// Recommended Improvements:
//
//	```go
//	func PublishProblemSolved(nc *nats.Conn, userID int, problem, answer string) error {
//	    // Validate connection
//	    if nc == nil {
//	        return fmt.Errorf("NATS connection is nil")
//	    }
//
//	    // Validate inputs
//	    if userID <= 0 {
//	        return fmt.Errorf("invalid userID: %d", userID)
//	    }
//	    if len(problem) == 0 || len(problem) > 500 {
//	        return fmt.Errorf("invalid problem length: %d", len(problem))
//	    }
//	    if len(answer) == 0 || len(answer) > 100 {
//	        return fmt.Errorf("invalid answer length: %d", len(answer))
//	    }
//
//	    // Create and publish event
//	    event := ProblemSolvedEvent{
//	        UserID:      userID,
//	        ProblemText: problem,
//	        AnswerText:  answer,
//	    }
//	    // ... rest of function
//	}
//	```
//
// Alternative Patterns (for production):
//   - Use NATS JetStream for persistent messages and at-least-once delivery
//   - Implement retry logic for failed publishes
//   - Add message deduplication to prevent duplicate history entries
//   - Use request-reply pattern to get acknowledgment from history-worker
func PublishProblemSolved(nc *nats.Conn, userID int, problem, answer string) error {
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
