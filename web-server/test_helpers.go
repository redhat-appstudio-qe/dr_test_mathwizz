package main

// This file contains test helper utilities for NATS message queue testing.
// It provides functions for publishing test events and subscribing to topics during tests.

import (
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go"
)

// TestNATSSubscriber creates a subscriber for testing purposes.
// Returns the subscription and a channel to receive messages.
func TestNATSSubscriber(nc *nats.Conn, subject string) (*nats.Subscription, chan *nats.Msg, error) {
	msgChan := make(chan *nats.Msg, 10)

	sub, err := nc.ChanSubscribe(subject, msgChan)
	if err != nil {
		return nil, nil, err
	}

	return sub, msgChan, nil
}

// WaitForNATSMessage waits for a message on the channel with a timeout.
// Returns the received message or nil if timeout occurs.
func WaitForNATSMessage(msgChan chan *nats.Msg, timeout time.Duration) *nats.Msg {
	select {
	case msg := <-msgChan:
		return msg
	case <-time.After(timeout):
		return nil
	}
}

// ParseProblemSolvedEvent parses a NATS message into a ProblemSolvedEvent.
// Returns the parsed event or an error if parsing fails.
func ParseProblemSolvedEvent(msg *nats.Msg) (*ProblemSolvedEvent, error) {
	var event ProblemSolvedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// PublishTestEvent publishes a test event directly to NATS for testing.
// Useful for testing the history-worker subscriber.
func PublishTestEvent(nc *nats.Conn, userID int, problem, answer string) error {
	return PublishProblemSolved(nc, userID, problem, answer)
}
