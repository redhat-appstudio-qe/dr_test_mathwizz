// Unit tests for utility functions
// Tests pure, non-UI helper functions in isolation

import {
  validateEmail,
  validatePassword,
  validateProblem,
  formatDate,
} from '../utils';

describe('Utils - validateEmail', () => {
  test('returns null for valid email', () => {
    expect(validateEmail('test@example.com')).toBeNull();
    expect(validateEmail('user.name@domain.co.uk')).toBeNull();
  });

  test('returns error for invalid email', () => {
    expect(validateEmail('invalid')).toBe('Invalid email format');
    expect(validateEmail('test@')).toBe('Invalid email format');
    expect(validateEmail('@example.com')).toBe('Invalid email format');
  });

  test('returns error for empty email', () => {
    expect(validateEmail('')).toBe('Email is required');
    expect(validateEmail(null)).toBe('Email is required');
    expect(validateEmail(undefined)).toBe('Email is required');
  });

  // Length validation tests - testing Bug #36 (no length validation exists)
  // Database limits email to VARCHAR(255), but frontend has no length check
  // These tests will FAIL (expose Bug #36) because validateEmail doesn't check length

  describe('length validation tests', () => {
    test('accepts email with exactly 255 characters (boundary - database limit)', () => {
      // Create a valid email with exactly 255 characters
      // Using 'a' repeated 243 times + '@example.com' (12 chars) = 255 total
      const exactlyTwoFiftyFive = 'a'.repeat(243) + '@example.com';

      // Should pass validation (null) - this is the boundary condition
      // Currently PASSES because no length validation exists (Bug #36)
      // After Bug #36 is fixed, should still PASS (255 is within limit)
      expect(exactlyTwoFiftyFive.length).toBe(255);
      expect(validateEmail(exactlyTwoFiftyFive)).toBeNull();
    });

    test('rejects email with 256 characters (exceeds database limit)', () => {
      // Create a valid email with exactly 256 characters
      // Using 'a' repeated 244 times + '@example.com' (12 chars) = 256 total
      const twoFiftySix = 'a'.repeat(244) + '@example.com';

      // EXPECTED: Should return error about length limit
      // ACTUAL: Returns null (Bug #36 - no validation)
      // Test will FAIL exposing Bug #36
      expect(twoFiftySix.length).toBe(256);
      expect(validateEmail(twoFiftySix)).toBe('Email must be 255 characters or less');
      // When this test FAILS, it proves Bug #36 exists (no length validation)
    });

    test('rejects email with 500 characters (extreme DoS prevention)', () => {
      // Create a valid email with 500 characters
      // Using 'a' repeated 488 times + '@example.com' (12 chars) = 500 total
      const fiveHundred = 'a'.repeat(488) + '@example.com';

      // EXPECTED: Should return error about length limit
      // ACTUAL: Returns null (Bug #36 - no validation)
      // This is a DoS prevention test - extremely long inputs should be rejected
      expect(fiveHundred.length).toBe(500);
      expect(validateEmail(fiveHundred)).toBe('Email must be 255 characters or less');
      // When this test FAILS, it documents a security gap (DoS risk)
    });

    test('rejects email with 1000 characters (massive DoS attempt)', () => {
      // Create a valid email with 1000 characters
      // Using 'a' repeated 988 times + '@example.com' (12 chars) = 1000 total
      const oneThousand = 'a'.repeat(988) + '@example.com';

      // EXPECTED: Should return error about length limit
      // ACTUAL: Returns null (Bug #36 - no validation)
      // This tests fail-fast rejection of massive inputs
      expect(oneThousand.length).toBe(1000);
      expect(validateEmail(oneThousand)).toBe('Email must be 255 characters or less');
      // Test failure demonstrates critical security gap (no DoS protection)
    });
  });
});

describe('Utils - validatePassword', () => {
  test('returns null for valid password', () => {
    expect(validatePassword('password123')).toBeNull();
    expect(validatePassword('123456')).toBeNull();
  });

  test('returns error for short password', () => {
    expect(validatePassword('12345')).toBe('Password must be at least 6 characters');
    expect(validatePassword('abc')).toBe('Password must be at least 6 characters');
  });

  test('returns error for empty password', () => {
    expect(validatePassword('')).toBe('Password is required');
    expect(validatePassword(null)).toBe('Password is required');
    expect(validatePassword(undefined)).toBe('Password is required');
  });

  // Length validation tests - testing Bug #37 (no maximum length validation exists)
  // bcrypt has a 72-byte limit - passwords > 72 bytes are silently truncated (Bug #14)
  // Frontend should validate maximum length to prevent:
  // 1. Security risk (user thinks they set long password, but only first 72 bytes are hashed)
  // 2. DoS with extremely long passwords causing bcrypt processing delays
  // These tests will FAIL (expose Bug #37) because validatePassword doesn't check maximum length

  describe('maximum length validation tests', () => {
    test('accepts password with exactly 72 characters (bcrypt limit boundary)', () => {
      // bcrypt's documented limit is 72 bytes (not characters, but for ASCII they're equal)
      // A password at this limit should be accepted but documented
      const exactlySeventyTwo = 'a'.repeat(72);

      // This is the boundary condition - should this be accepted or rejected?
      // Currently PASSES because no maximum length validation exists
      // After Bug #37 is fixed, this test documents whether 72 chars is within or at limit
      expect(exactlySeventyTwo.length).toBe(72);
      expect(validatePassword(exactlySeventyTwo)).toBeNull();
      // Note: This test passing documents that 72-char passwords are accepted
      // The security risk is that users may not know bcrypt will use all 72 bytes
    });

    test('rejects password with 73 characters (exceeds bcrypt limit by 1)', () => {
      // bcrypt silently truncates passwords > 72 bytes
      // Frontend should reject these to warn users about truncation
      const seventyThree = 'a'.repeat(73);

      // EXPECTED: Should return error about bcrypt's 72-byte limit
      // ACTUAL: Returns null (Bug #37 - no validation)
      // This is a SECURITY issue - user thinks they set 73-char password,
      // but bcrypt only hashes first 72 bytes
      expect(seventyThree.length).toBe(73);
      expect(validatePassword(seventyThree)).toBe('Password must be 72 characters or less (bcrypt limit)');
      // When this test FAILS, it exposes Bug #37 (no maximum length validation)
    });

    test('rejects password with 100 characters (reasonable maximum)', () => {
      // A reasonable maximum length for password validation
      // Prevents excessively long passwords while allowing strong passphrases
      const oneHundred = 'a'.repeat(100);

      // EXPECTED: Should return error about maximum length
      // ACTUAL: Returns null (Bug #37 - no validation)
      expect(oneHundred.length).toBe(100);
      expect(validatePassword(oneHundred)).toBe('Password must be 72 characters or less (bcrypt limit)');
      // Test failure documents security gap - no maximum length enforcement
    });

    test('rejects password with 200 characters (very long password)', () => {
      // Very long passwords that significantly exceed bcrypt's limit
      // Should definitely be rejected to prevent user confusion
      const twoHundred = 'a'.repeat(200);

      // EXPECTED: Should return error about maximum length
      // ACTUAL: Returns null (Bug #37 - no validation)
      // Users entering 200-char passwords think they're super secure,
      // but bcrypt only uses first 72 bytes - massive confusion
      expect(twoHundred.length).toBe(200);
      expect(validatePassword(twoHundred)).toBe('Password must be 72 characters or less (bcrypt limit)');
      // Test failure demonstrates user experience problem with silent truncation
    });

    test('rejects password with 1000 characters (DoS prevention)', () => {
      // Extremely long passwords could cause DoS via bcrypt processing time
      // Even though bcrypt only uses first 72 bytes, validation should reject early
      const oneThousand = 'a'.repeat(1000);

      // EXPECTED: Should return error about maximum length (fail-fast before bcrypt)
      // ACTUAL: Returns null (Bug #37 - no validation)
      // This is a DoS prevention test - extremely long inputs should be rejected
      expect(oneThousand.length).toBe(1000);
      expect(validatePassword(oneThousand)).toBe('Password must be 72 characters or less (bcrypt limit)');
      // Test failure demonstrates DoS vulnerability - no length cap
    });

    test('rejects password with 10000 characters (massive DoS attempt)', () => {
      // Massive input that would definitely cause performance issues
      // Should be rejected immediately to prevent resource exhaustion
      const tenThousand = 'a'.repeat(10000);

      // EXPECTED: Should return error about maximum length
      // ACTUAL: Returns null (Bug #37 - no validation)
      // Massive DoS risk - processing 10k character password wastes resources
      expect(tenThousand.length).toBe(10000);
      expect(validatePassword(tenThousand)).toBe('Password must be 72 characters or less (bcrypt limit)');
      // Test failure demonstrates critical security gap (no DoS protection)
    });
  });
});

describe('Utils - validateProblem', () => {
  test('returns null for valid math problem', () => {
    expect(validateProblem('2+2')).toBeNull();
    expect(validateProblem('5*10')).toBeNull();
    expect(validateProblem('(10+5)*2')).toBeNull();
    expect(validateProblem('100 - 50')).toBeNull();
  });

  test('returns error for invalid characters', () => {
    expect(validateProblem('abc')).toBe('Problem contains invalid characters');
    expect(validateProblem('2+2 hello')).toBe('Problem contains invalid characters');
    expect(validateProblem('x+y')).toBe('Problem contains invalid characters');
  });

  test('returns error for empty problem', () => {
    expect(validateProblem('')).toBe('Please enter a math problem');
    expect(validateProblem('   ')).toBe('Please enter a math problem');
    expect(validateProblem(null)).toBe('Please enter a math problem');
  });

  // Length validation tests - testing Bug #35 (no length validation exists)
  // Database limits problem_text to VARCHAR(500), but frontend has no length check
  // These tests will FAIL (expose Bug #35) because validateProblem doesn't check length

  describe('length validation tests', () => {
    test('accepts problem with exactly 500 characters (boundary - database limit)', () => {
      // Create a valid math expression with exactly 500 characters
      // Using '2+' (2 chars) repeated 250 times = 500 chars total
      const exactlyFiveHundred = '2+'.repeat(250);

      // Should pass validation (null) - this is the boundary condition
      // Currently PASSES because no length validation exists (Bug #35)
      // After Bug #35 is fixed, should still PASS (500 is within limit)
      expect(exactlyFiveHundred.length).toBe(500);
      expect(validateProblem(exactlyFiveHundred)).toBeNull();
    });

    test('rejects problem with 501 characters (exceeds database limit)', () => {
      // Create a valid math expression with exactly 501 characters
      // Using '2+' repeated 250 times (500 chars) + '2' (1 char) = 501 chars total
      const fiveHundredOne = '2+'.repeat(250) + '2';

      // EXPECTED: Should return error about length limit
      // ACTUAL: Returns null (Bug #35 - no validation)
      // Test will FAIL exposing Bug #35
      expect(fiveHundredOne.length).toBe(501);
      expect(validateProblem(fiveHundredOne)).toBe('Problem must be 500 characters or less');
      // When this test FAILS, it proves Bug #35 exists (no length validation)
    });

    test('rejects problem with 1000 characters (extreme DoS prevention)', () => {
      // Create a math expression with 1000 characters
      // Using '2+' repeated 500 times = 1000 chars total
      const oneThousand = '2+'.repeat(500);

      // EXPECTED: Should return error about length limit
      // ACTUAL: Returns null (Bug #35 - no validation)
      // This is a DoS prevention test - extremely long inputs should be rejected
      expect(oneThousand.length).toBe(1000);
      expect(validateProblem(oneThousand)).toBe('Problem must be 500 characters or less');
      // When this test FAILS, it documents a security gap (DoS risk)
    });

    test('rejects problem with 5000 characters (massive DoS attempt)', () => {
      // Create a math expression with 5000 characters (massive input)
      // Using '2+' repeated 2500 times = 5000 chars total
      const fiveThousand = '2+'.repeat(2500);

      // EXPECTED: Should return error about length limit
      // ACTUAL: Returns null (Bug #35 - no validation)
      // This tests fail-fast rejection of massive inputs
      expect(fiveThousand.length).toBe(5000);
      expect(validateProblem(fiveThousand)).toBe('Problem must be 500 characters or less');
      // Test failure demonstrates critical security gap (no DoS protection)
    });
  });
});

describe('Utils - formatDate', () => {
  test('formats date string to locale string', () => {
    const dateString = '2024-01-15T10:30:00Z';
    const result = formatDate(dateString);
    expect(result).toContain('2024');
    expect(typeof result).toBe('string');
  });

  test('handles invalid date gracefully', () => {
    const result = formatDate('invalid-date');
    expect(result).toBe('Invalid Date');
  });
});
