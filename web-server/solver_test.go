package main

// This file contains unit tests for the SolveMath function.
// Tests pure logic without database or HTTP dependencies.

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SolveMath", func() {
	When("solving valid mathematical expressions", func() {
		DescribeTable("returns correct results",
			func(problem string, expected int) {
				Expect(SolveMath(problem)).Should(Equal(expected))
			},
			Entry("addition", "2+2", 4),
			Entry("subtraction", "10-3", 7),
			Entry("multiplication", "5*10", 50),
			Entry("division", "20/4", 5),
			Entry("complex expression", "25+75", 100),
			Entry("expression with parentheses", "(10+5)*2", 30),
			Entry("multiple operations", "100-50+25", 75),
		)
	})

	When("handling division with fractional results", func() {
		// These tests verify that SolveMath truncates (not rounds) float results to integers.
		// Go's int(float64) conversion truncates toward zero: 2.9 → 2, -2.9 → -2.
		// This is important documented behavior that users and developers should understand.

		It("should truncate 5/2 to 2 (not round to 3)", func() {
			// 5/2 = 2.5 internally, truncates to 2 (NOT rounded to 3)
			Expect(SolveMath("5/2")).Should(Equal(2))
		})

		It("should truncate 7/2 to 3 (from 3.5)", func() {
			// 7/2 = 3.5 internally, truncates to 3
			Expect(SolveMath("7/2")).Should(Equal(3))
		})

		It("should truncate 1/2 to 0 (not round to 1)", func() {
			// 1/2 = 0.5 internally, truncates to 0 (NOT rounded to 1)
			Expect(SolveMath("1/2")).Should(Equal(0))
		})

		It("should truncate 9/4 to 2 (from 2.25)", func() {
			// 9/4 = 2.25 internally, truncates to 2
			Expect(SolveMath("9/4")).Should(Equal(2))
		})

		It("should truncate 10/3 to 3 (from 3.333...)", func() {
			// 10/3 = 3.333... internally, truncates to 3
			Expect(SolveMath("10/3")).Should(Equal(3))
		})

		It("should truncate negative division results toward zero", func() {
			// -7/2 = -3.5 internally, truncates to -3 (toward zero, not -4)
			Expect(SolveMath("-7/2")).Should(Equal(-3))
		})

		It("should truncate complex expressions with fractional intermediate results", func() {
			// (5/2) + (7/2) = 2.5 + 3.5 = 6.0 internally
			// But with int truncation at each step: (5/2) → 2, (7/2) → 3
			// Actually govaluate evaluates the full expression as float, then we truncate
			// So: (5/2) + (7/2) = 6.0 → truncates to 6
			Expect(SolveMath("(5/2)+(7/2)")).Should(Equal(6))
		})

		It("should handle multiplication with fractional division results", func() {
			// (10/3) * 2 = 3.333... * 2 = 6.666... → truncates to 6
			Expect(SolveMath("(10/3)*2")).Should(Equal(6))
		})
	})

	When("handling invalid or edge case inputs", func() {
		DescribeTable("returns appropriate errors",
			func(problem string, errorSubstring string) {
				_, err := SolveMath(problem)
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring(errorSubstring))
			},
			Entry("empty string", "", "cannot be empty"),
			Entry("invalid characters", "abc", "invalid expression"),
			Entry("incomplete expression", "5+", "invalid expression"),
			Entry("division by zero", "10/0", "evaluation failed"),
		)

		It("should handle negative results", func() {
			Expect(SolveMath("5-10")).Should(Equal(-5))
		})

		It("should handle zero result", func() {
			Expect(SolveMath("5-5")).Should(Equal(0))
		})
	})

	When("testing integer overflow scenarios and platform limits", func() {
		// These tests verify behavior when calculations approach or exceed integer type limits.
		// Go's int type is platform-dependent:
		// - On 32-bit systems: int is int32, range: -2,147,483,648 to 2,147,483,647
		// - On 64-bit systems: int is int64, range: -9,223,372,036,854,775,808 to 9,223,372,036,854,775,807
		//
		// The solver has maxNumberLength=15, so the largest input number is 999,999,999,999,999 (15 digits).
		// int64 max is 9,223,372,036,854,775,807 (19 digits), so operations can exceed this range.
		//
		// IMPORTANT: The current implementation does NOT detect or handle integer overflow.
		// These tests DOCUMENT this behavior and may reveal bugs where overflow causes incorrect results.

		When("testing numbers within valid int32 range", func() {
			// These tests verify correct handling on both 32-bit and 64-bit platforms

			It("should correctly handle int32 maximum value (2147483647)", func() {
				// 2,147,483,647 is int32 max (10 digits, well within 15-digit limit)
				// This should work correctly on all platforms
				Expect(SolveMath("2147483647")).Should(Equal(2147483647))
			})

			It("should correctly handle addition up to int32 max", func() {
				// 2147483646 + 1 = 2147483647 (int32 max)
				// Tests boundary arithmetic
				Expect(SolveMath("2147483646+1")).Should(Equal(2147483647))
			})

			It("should correctly handle int32 minimum value (-2147483648)", func() {
				// -2,147,483,648 is int32 min
				// Tests negative boundary
				Expect(SolveMath("-2147483648")).Should(Equal(-2147483648))
			})
		})

		When("testing numbers that exceed int32 but fit in int64", func() {
			// These tests may behave differently on 32-bit vs 64-bit platforms

			It("should handle multiplication exceeding int32 range", func() {
				// 100000 * 100000 = 10,000,000,000 (exceeds int32 max ~2.1 billion)
				// On 64-bit: should work correctly
				// On 32-bit: will overflow (platform-dependent behavior)
				result, err := SolveMath("100000*100000")
				Expect(err).ShouldNot(HaveOccurred())

				// On most modern systems (64-bit), this should be correct
				// On 32-bit systems, overflow behavior is platform-dependent
				expectedResult := 10000000000
				if result != expectedResult {
					GinkgoWriter.Printf("Platform-dependent behavior detected:\n")
					GinkgoWriter.Printf("  Expression: 100000*100000\n")
					GinkgoWriter.Printf("  Expected: %d\n", expectedResult)
					GinkgoWriter.Printf("  Actual: %d\n", result)
					GinkgoWriter.Printf("  Note: Running on 32-bit system or int overflow occurred\n")
				}
				// On 64-bit systems, expect correct result
				Expect(result).Should(Equal(expectedResult))
			})

			It("should handle large numbers approaching int64 limits", func() {
				// 500,000,000,000,000 * 10 = 5,000,000,000,000,000 (5 quadrillion)
				// Well within int64 range (max ~9.2 quintillion)
				Expect(SolveMath("500000000000000*10")).Should(Equal(5000000000000000))
			})
		})

		When("testing numbers at the 15-digit input validation limit", func() {
			// The maxNumberLength=15 constraint limits individual input numbers to 15 digits

			It("should accept 15-digit numbers at the validation boundary", func() {
				// 999,999,999,999,999 is the largest 15-digit number
				// Still well within int64 range
				Expect(SolveMath("999999999999999")).Should(Equal(999999999999999))
			})

			It("should accept addition of large 15-digit numbers within int64 range", func() {
				// 500000000000000 + 400000000000000 = 900,000,000,000,000
				// Both inputs are 15 digits, result fits in int64
				Expect(SolveMath("500000000000000+400000000000000")).Should(Equal(900000000000000))
			})

			It("should reject numbers exceeding 15 digits due to maxNumberLength validation", func() {
				// 9999999999999999 is 16 digits, should be rejected
				_, err := SolveMath("9999999999999999")
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("number too large"))
				Expect(err.Error()).Should(ContainSubstring("15 digits"))
			})

			It("should reject expressions with any number exceeding 15 digits", func() {
				// Even in valid expression, 16-digit numbers should be rejected
				_, err := SolveMath("1111111111111111+1")
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("number too large"))
			})
		})

		When("testing multiplication that causes int64 overflow", func() {
			// These tests reveal the bug: overflow is NOT detected or prevented
			// The function returns incorrect results without error

			It("should detect overflow when multiplying maximum 15-digit numbers", func() {
				// 999999999999999 * 999999999999999 = 999,999,999,999,998,000,000,000,000,001
				// This is approximately 10^30, which vastly exceeds int64 max (9.22 * 10^18)
				//
				// EXPECTED BEHAVIOR: Should return error indicating overflow
				// ACTUAL BEHAVIOR: Returns incorrect result due to float64->int conversion overflow
				//
				// This test SHOULD FAIL, revealing the overflow detection bug
				result, err := SolveMath("999999999999999*999999999999999")

				GinkgoWriter.Printf("\n=== INTEGER OVERFLOW BUG DETECTED ===\n")
				GinkgoWriter.Printf("Expression: 999999999999999*999999999999999\n")
				GinkgoWriter.Printf("Mathematical result: 999,999,999,999,998,000,000,000,000,001 (~10^30)\n")
				GinkgoWriter.Printf("int64 max value: 9,223,372,036,854,775,807 (~9.22 * 10^18)\n")
				GinkgoWriter.Printf("Actual result from function: %d\n", result)
				GinkgoWriter.Printf("Error returned: %v\n", err)
				GinkgoWriter.Printf("\nBUG: Overflow is not detected. Function should return error for results exceeding int64 range.\n")
				GinkgoWriter.Printf("Current behavior: govaluate computes as float64, int(float64) conversion causes overflow.\n")
				GinkgoWriter.Printf("=====================================\n\n")

				// The function should return an error for overflow, but it doesn't
				// This test FAILS to reveal the bug
				Expect(err).Should(HaveOccurred(), "overflow should be detected and return error")
				Expect(err.Error()).Should(ContainSubstring("overflow"), "error message should mention overflow")
			})

			It("should detect overflow for multiplication approaching int64 limits", func() {
				// 100000000000000 * 100000000000000 = 10,000,000,000,000,000,000,000,000,000 (10^28)
				// Exceeds int64 max by orders of magnitude
				//
				// EXPECTED: Error indicating overflow
				// ACTUAL: Incorrect result, no error
				result, err := SolveMath("100000000000000*100000000000000")

				GinkgoWriter.Printf("\n=== INTEGER OVERFLOW BUG (second scenario) ===\n")
				GinkgoWriter.Printf("Expression: 100000000000000*100000000000000\n")
				GinkgoWriter.Printf("Mathematical result: 10,000,000,000,000,000,000,000,000,000 (10^28)\n")
				GinkgoWriter.Printf("Actual result: %d\n", result)
				GinkgoWriter.Printf("Error: %v\n", err)
				GinkgoWriter.Printf("BUG: No overflow detection for results exceeding int64 range\n")
				GinkgoWriter.Printf("==============================================\n\n")

				// Should error, but doesn't - test reveals the bug
				Expect(err).Should(HaveOccurred(), "overflow should be detected")
			})

			It("should document float64 precision loss for large multiplications", func() {
				// 3000000000 * 3000000000 = 9,000,000,000,000,000,000 (9 * 10^18)
				// This is just under int64 max (9.22 * 10^18) and should fit
				// However, float64 has ~15-17 decimal digits of precision, and this is 19 digits
				//
				// EXPECTED: Exact result 9000000000000000000 (if within int64)
				// ACTUAL: May have precision loss due to float64 intermediate representation
				result, err := SolveMath("3000000000*3000000000")
				Expect(err).ShouldNot(HaveOccurred())

				expectedResult := int64(9000000000000000000)
				GinkgoWriter.Printf("\n=== FLOAT64 PRECISION TEST ===\n")
				GinkgoWriter.Printf("Expression: 3000000000*3000000000\n")
				GinkgoWriter.Printf("Expected: %d (9*10^18, within int64 range)\n", expectedResult)
				GinkgoWriter.Printf("Actual: %d\n", result)

				if result != 9000000000000000000 {
					GinkgoWriter.Printf("PRECISION LOSS DETECTED: Result differs from expected value\n")
					GinkgoWriter.Printf("Cause: float64 precision (~15-17 digits) insufficient for 19-digit result\n")
				} else {
					GinkgoWriter.Printf("Result is correct (within float64 precision)\n")
				}
				GinkgoWriter.Printf("==============================\n\n")

				// Expect exact result - if this fails, float64 precision is insufficient
				Expect(result).Should(Equal(9000000000000000000), "result should be exact when within int64 range")
			})

			It("should not validate output magnitude (only input digits)", func() {
				// 10000000000000 * 10000 = 100,000,000,000,000,000 (10^17, within int64)
				// Both inputs have ≤15 digits, result is 17 digits but fits in int64
				//
				// This documents that maxNumberLength only validates INPUT numbers,
				// not intermediate or final results
				result, err := SolveMath("10000000000000*10000")
				Expect(err).ShouldNot(HaveOccurred())

				GinkgoWriter.Printf("Output magnitude validation test:\n")
				GinkgoWriter.Printf("  Expression: 10000000000000*10000\n")
				GinkgoWriter.Printf("  Result: %d (17 digits)\n", result)
				GinkgoWriter.Printf("  Note: Input validation (15 digits) doesn't limit output magnitude\n")

				// Result should be correct (within int64 range)
				Expect(result).Should(Equal(100000000000000000))
			})
		})
	})

	When("handling floating point input numbers", func() {
		// These tests verify that SolveMath accepts floating point numbers (decimal notation)
		// in input expressions and processes them correctly. The govaluate library supports
		// float inputs, and the solver truncates float results to integers using the same
		// truncation behavior as division results (int(float64) conversion).
		//
		// This documents that users can provide inputs like "1.5+2.5" and the solver will
		// evaluate them, truncating the result to an integer.

		When("processing simple float operations", func() {
			It("should accept and compute float addition (1.5+2.5 → 4)", func() {
				// 1.5 + 2.5 = 4.0 internally, truncates to 4
				Expect(SolveMath("1.5+2.5")).Should(Equal(4))
			})

			It("should accept and compute float multiplication (2.5*2.0 → 5)", func() {
				// 2.5 * 2.0 = 5.0 internally, result is exact integer 5
				Expect(SolveMath("2.5*2.0")).Should(Equal(5))
			})

			It("should accept mixed integer and float addition (5+2.5 → 7)", func() {
				// 5 + 2.5 = 7.5 internally, truncates to 7
				Expect(SolveMath("5+2.5")).Should(Equal(7))
			})

			It("should accept float subtraction (10.5-3.2 → 7)", func() {
				// 10.5 - 3.2 = 7.3 internally, truncates to 7
				Expect(SolveMath("10.5-3.2")).Should(Equal(7))
			})
		})

		When("processing float operations with truncation", func() {
			It("should truncate float division result (7.5/2 → 3)", func() {
				// 7.5 / 2 = 3.75 internally, truncates to 3
				Expect(SolveMath("7.5/2")).Should(Equal(3))
			})

			It("should handle negative float operations (-1.5+0.5 → -1)", func() {
				// -1.5 + 0.5 = -1.0 internally, exact result
				Expect(SolveMath("-1.5+0.5")).Should(Equal(-1))
			})

			It("should truncate sum of multiple floats (1.1+2.2+3.3 → 6)", func() {
				// 1.1 + 2.2 + 3.3 = 6.6 internally (with potential float precision issues)
				// Truncates to 6
				// Note: Float precision may cause slight variation (e.g., 6.600000000000001)
				// but truncation to int is consistent
				result, err := SolveMath("1.1+2.2+3.3")
				Expect(err).ShouldNot(HaveOccurred())
				// Result should be 6, accounting for float precision
				Expect(result).Should(BeNumerically(">=", 6))
				Expect(result).Should(BeNumerically("<=", 6))
			})

			It("should handle very small float sums that truncate to zero", func() {
				// 0.1 + 0.1 + 0.1 = 0.3 internally, truncates to 0
				Expect(SolveMath("0.1+0.1+0.1")).Should(Equal(0))
			})

			It("should truncate negative float toward zero (-2.8 → -2)", func() {
				// -5 + 2.2 = -2.8, truncates to -2 (toward zero, not -3)
				Expect(SolveMath("-5+2.2")).Should(Equal(-2))
			})
		})

		When("processing complex expressions with floats", func() {
			It("should handle floats in parenthetical expressions", func() {
				// (1.5 + 2.5) * 2 = 4.0 * 2 = 8.0, returns 8
				Expect(SolveMath("(1.5+2.5)*2")).Should(Equal(8))
			})

			It("should handle floats that result in whole numbers", func() {
				// 99.9 + 0.1 = 100.0 exactly, returns 100
				Expect(SolveMath("99.9+0.1")).Should(Equal(100))
			})

			It("should handle nested float operations with mixed truncation", func() {
				// (10.5 / 2) * 3 = (5.25) * 3 = 15.75, truncates to 15
				// Note: govaluate evaluates entire expression, so: 10.5 / 2 * 3 = 15.75 → 15
				Expect(SolveMath("(10.5/2)*3")).Should(Equal(15))
			})

			It("should handle multiple decimal points in expression", func() {
				// 1.5 + 2.3 + 3.7 = 7.5, truncates to 7
				Expect(SolveMath("1.5+2.3+3.7")).Should(Equal(7))
			})

			It("should handle float multiplication resulting in truncation", func() {
				// 3.3 * 3.3 = 10.89, truncates to 10
				Expect(SolveMath("3.3*3.3")).Should(Equal(10))
			})
		})

		When("testing edge cases with float inputs", func() {
			It("should handle float with many decimal places", func() {
				// 1.123456789 + 2.987654321 = 4.11111111, truncates to 4
				Expect(SolveMath("1.123456789+2.987654321")).Should(Equal(4))
			})

			It("should handle floats in division with zero fractional result", func() {
				// 10.0 / 2.0 = 5.0 exactly, returns 5
				Expect(SolveMath("10.0/2.0")).Should(Equal(5))
			})

			It("should handle very small floats near zero", func() {
				// 0.001 + 0.002 + 0.003 = 0.006, truncates to 0
				Expect(SolveMath("0.001+0.002+0.003")).Should(Equal(0))
			})

			It("should handle large floats within validation limits", func() {
				// 999999999999.5 + 0.5 = 1000000000000.0 (12 digits)
				// Both numbers have ≤15 total digits (including decimal point and fractional part)
				Expect(SolveMath("999999999999.5+0.5")).Should(Equal(1000000000000))
			})
		})
	})
})
