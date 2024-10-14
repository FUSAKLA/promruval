package validate

import (
	"testing"

)

func TestGenerateExcludedRules(t *testing.T) {
	type testCase struct {
		input    string
		expected []string
	}
	testCases := []testCase{
		{
			input:    "check-test-label,check-testy-label",
			expected: []string{"check-test-label", "check-testy-label"},
		},
		{
			input:    "check-test-label,check-test-label",
			expected: []string{"check-test-label"},
		},
		{
			input:    "check-test-label, check-testy-label",
			expected: []string{"check-test-label", "check-testy-label"},
		},
		{
			input:    "check-test-label, check-testy-label,",
			expected: []string{"check-test-label", "check-testy-label"},
		},
		{
			input:    "check-test-label ,check-test-label",
			expected: []string{"check-test-label"},
		},
		{
			input:    "check-test-label,check-test-label, ",
			expected: []string{"check-test-label"},
		},
		{
			input:    "check-test-label  ,  check testy label , ",
			expected: []string{"check testy label", "check-test-label"},
		},
		{
			input:    "check-test-label , check-testy-label    ",
			expected: []string{"check-test-label", "check-testy-label"},
		},
		{
			input:    "check-test-label , check-testy-label, ,    ",
			expected: []string{"check-test-label", "check-testy-label"},
		},
		{
			input:    " check-test-label , check-testy-label , ,    ",
			expected: []string{"check-test-label", "check-testy-label"},
		},
	}
	for i, tc := range testCases {
		t.Run(string(rune(i)), func(t *testing.T) {
			result := generateExcludedRules(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
