package main

// This file implements the math problem solver.
// It uses the govaluate library to safely evaluate mathematical expressions.

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Knetic/govaluate"
)

// Security limits for expression evaluation to prevent resource exhaustion attacks
const (
	maxProblemLength  = 100             // Maximum characters in expression
	evaluationTimeout = 1 * time.Second // Maximum evaluation time
	maxNestingDepth   = 10              // Maximum parenthesis nesting depth
	maxOperatorCount  = 50              // Maximum number of operators
	maxNumberLength   = 15              // Maximum digits in a single number
)

// validateComplexity checks if an expression is too complex and could cause resource exhaustion.
// It validates operator count, nesting depth, and maximum number length to prevent DoS attacks
// via deeply nested expressions or large exponentiations.
//
// Returns an error if any complexity limit is exceeded, nil otherwise.
func validateComplexity(problem string) error {
	// Count operators to limit computational complexity
	operatorCount := strings.Count(problem, "+") +
		strings.Count(problem, "-") +
		strings.Count(problem, "*") +
		strings.Count(problem, "/") +
		strings.Count(problem, "^")

	if operatorCount > maxOperatorCount {
		return fmt.Errorf("expression too complex: maximum %d operators allowed", maxOperatorCount)
	}

	// Check parenthesis nesting depth to prevent stack exhaustion
	depth := 0
	maxDepth := 0
	for _, ch := range problem {
		if ch == '(' {
			depth++
			if depth > maxDepth {
				maxDepth = depth
			}
		} else if ch == ')' {
			depth--
		}
	}

	if maxDepth > maxNestingDepth {
		return fmt.Errorf("expression too complex: maximum nesting depth of %d exceeded", maxNestingDepth)
	}

	// Check for very long numbers that could cause overflow or slow evaluation
	currentNumberLength := 0
	for _, ch := range problem {
		if ch >= '0' && ch <= '9' {
			currentNumberLength++
			if currentNumberLength > maxNumberLength {
				return fmt.Errorf("number too large: maximum %d digits allowed", maxNumberLength)
			}
		} else {
			currentNumberLength = 0
		}
	}

	return nil
}

// SolveMath evaluates a mathematical expression and returns the computed result as an integer.
// It uses the govaluate library to parse and evaluate expressions containing basic arithmetic
// operations, with multiple security controls to prevent resource exhaustion attacks.
//
// Security Controls:
//   - Maximum input length: 100 characters (prevents memory exhaustion)
//   - Maximum operator count: 50 operators (prevents computational DoS)
//   - Maximum nesting depth: 10 levels (prevents stack exhaustion)
//   - Maximum number length: 15 digits (prevents overflow and slow evaluation)
//   - Evaluation timeout: 1 second (prevents hanging on complex expressions)
//
// Supported Operations:
//   - Addition: + (e.g., "5+3" returns 8)
//   - Subtraction: - (e.g., "10-4" returns 6)
//   - Multiplication: * (e.g., "6*7" returns 42)
//   - Division: / (e.g., "20/4" returns 5)
//   - Parentheses: ( ) for grouping (e.g., "(2+3)*4" returns 20)
//   - Negative numbers: (e.g., "-5+3" returns -2)
//
// Parameters:
//   - problem: A string containing the mathematical expression to evaluate.
//     Must not be empty and must not exceed security limits.
//     Examples: "25+75", "10*5-20", "(8+2)/2"
//
// Returns:
//   - int: The computed result, truncated to an integer if the result is a float.
//     For example, "7/2" evaluates to 3.5 internally but returns 3 (truncated, not rounded).
//   - error: Non-nil if the expression is empty, exceeds security limits, has invalid syntax,
//     cannot be parsed, evaluation fails, times out, or the result cannot be converted.
//
// Type Conversion Behavior:
//   - If govaluate returns float64: Truncated to int (7.5 → 7, not rounded to 8)
//   - If govaluate returns int: Returned directly
//   - If govaluate returns string: Attempts strconv.Atoi conversion (edge case, rarely occurs)
//   - Other types: Returns error with type information
//
// Error Cases:
//   - Empty string: "problem cannot be empty"
//   - Too long: "problem too long: maximum 100 characters allowed"
//   - Too many operators: "expression too complex: maximum 50 operators allowed"
//   - Too deeply nested: "expression too complex: maximum nesting depth of 10 exceeded"
//   - Number too large: "number too large: maximum 15 digits allowed"
//   - Timeout: "evaluation timeout: expression took too long to evaluate"
//   - Invalid syntax: "invalid expression: <details>"
//   - Evaluation failure: "evaluation failed: <details>"
//   - Unconvertible result: "result is not a number: <value>" or "unexpected result type: <type>"
//
// Example Usage:
//
//	// Basic arithmetic
//	result, err := SolveMath("25+75")
//	// result: 100, err: nil
//
//	// Division with truncation
//	result, err := SolveMath("7/2")
//	// result: 3 (truncated from 3.5), err: nil
//
//	// Complex expression
//	result, err := SolveMath("(10+5)*2-8")
//	// result: 22, err: nil
//
//	// Error case: empty input
//	result, err := SolveMath("")
//	// result: 0, err: "problem cannot be empty"
//
//	// Error case: too long
//	result, err := SolveMath(strings.Repeat("1+", 60) + "1")  // 121 characters
//	// result: 0, err: "problem too long: maximum 100 characters allowed"
//
//	// Error case: too complex
//	result, err := SolveMath("((((((((((1+1))))))))))")  // 11 levels deep
//	// result: 0, err: "expression too complex: maximum nesting depth of 10 exceeded"
func SolveMath(problem string) (int, error) {
	// Validation: Check for empty input
	if problem == "" {
		return 0, fmt.Errorf("problem cannot be empty")
	}

	// Validation: Check maximum length to prevent memory exhaustion
	if len(problem) > maxProblemLength {
		return 0, fmt.Errorf("problem too long: maximum %d characters allowed", maxProblemLength)
	}

	// Validation: Check complexity to prevent computational DoS
	if err := validateComplexity(problem); err != nil {
		return 0, err
	}

	// Parse expression
	expression, err := govaluate.NewEvaluableExpression(problem)
	if err != nil {
		return 0, fmt.Errorf("invalid expression: %w", err)
	}

	// Evaluate with timeout protection to prevent hanging on complex expressions
	ctx, cancel := context.WithTimeout(context.Background(), evaluationTimeout)
	defer cancel()

	// Use channels to receive result or error from goroutine
	resultChan := make(chan interface{}, 1)
	errChan := make(chan error, 1)

	// Run evaluation in goroutine so we can enforce timeout
	// Include panic recovery as defense-in-depth against vulnerabilities in govaluate
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Convert panic to error to prevent server crash
				errChan <- fmt.Errorf("evaluation panicked: %v", r)
			}
		}()

		result, err := expression.Evaluate(nil)
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- result
	}()

	// Wait for result, error, or timeout
	select {
	case <-ctx.Done():
		return 0, fmt.Errorf("evaluation timeout: expression took too long to evaluate")
	case err := <-errChan:
		return 0, fmt.Errorf("evaluation failed: %w", err)
	case result := <-resultChan:
		// Convert result to integer
		switch v := result.(type) {
		case float64:
			return int(v), nil
		case int:
			return v, nil
		case string:
			intVal, err := strconv.Atoi(v)
			if err != nil {
				return 0, fmt.Errorf("result is not a number: %v", v)
			}
			return intVal, nil
		default:
			return 0, fmt.Errorf("unexpected result type: %T", result)
		}
	}
}
