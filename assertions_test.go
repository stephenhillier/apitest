package main

import "testing"

func TestEqualsAssertion(t *testing.T) {
	type testCase struct {
		Value      interface{}
		Comparison interface{}
		Expected   bool
	}

	cases := []testCase{
		testCase{Value: "123", Comparison: 123, Expected: true},
		testCase{Value: "123", Comparison: 124, Expected: false},
		testCase{Value: 123, Comparison: 123, Expected: true},
		testCase{Value: "the quick brown fox", Comparison: "the quick brown fox", Expected: true},
		testCase{Value: "quick", Comparison: 421421, Expected: false},
		testCase{Value: 123.4, Comparison: 123.4, Expected: true},
		testCase{Value: 123.4, Comparison: 123.5, Expected: false},
	}

	for _, c := range cases {
		err := checkAssertions(c.Value, map[string]interface{}{"equals": c.Comparison})
		if (err == nil) != c.Expected {
			t.Errorf("failed: expected %v == %v to have been %v; %v", c.Value, c.Comparison, c.Expected, err)
		}
	}
}

// TestLTAssertion tests the "less than" comparison rule
func TestLTAssertion(t *testing.T) {
	type testCase struct {
		Value      interface{}
		Comparison interface{}
		Expected   bool
	}

	cases := []testCase{
		testCase{Value: "123", Comparison: 123, Expected: false},
		testCase{Value: "123", Comparison: 124, Expected: true},
		testCase{Value: 123, Comparison: 124, Expected: true},
		testCase{Value: 123, Comparison: 122, Expected: false},
	}

	for _, c := range cases {
		err := checkAssertions(c.Value, map[string]interface{}{"lt": c.Comparison})
		if (err == nil) != c.Expected {
			t.Errorf("failed: expected %v < %v to have been %v; %v", c.Value, c.Comparison, c.Expected, err)
		}
	}
}

// TestGTAssertion tests the "greater than" comparison rule
func TestGTAssertion(t *testing.T) {
	type testCase struct {
		Value      interface{}
		Comparison interface{}
		Expected   bool
	}

	cases := []testCase{
		testCase{Value: "123", Comparison: 123, Expected: false},
		testCase{Value: "123", Comparison: 124, Expected: false},
		testCase{Value: 123, Comparison: 124, Expected: false},
		testCase{Value: 123, Comparison: 122, Expected: true},
	}

	for _, c := range cases {
		err := checkAssertions(c.Value, map[string]interface{}{"gt": c.Comparison})
		if (err == nil) != c.Expected {
			t.Errorf("failed: expected %v > %v to have been %v; %v", c.Value, c.Comparison, c.Expected, err)
		}
	}
}

// TestLEAssertion tests the "less than or equal to" comparison rule
func TestLEAssertion(t *testing.T) {
	type testCase struct {
		Value      interface{}
		Comparison interface{}
		Expected   bool
	}

	cases := []testCase{
		testCase{Value: "123", Comparison: 123, Expected: true},
		testCase{Value: "123", Comparison: 124, Expected: true},
		testCase{Value: 123, Comparison: 124, Expected: true},
		testCase{Value: 123, Comparison: 122, Expected: false},
		testCase{Value: 123, Comparison: 123, Expected: true},
	}

	for _, c := range cases {
		err := checkAssertions(c.Value, map[string]interface{}{"le": c.Comparison})
		if (err == nil) != c.Expected {
			t.Errorf("failed: expected %v <= %v to have been %v; %v", c.Value, c.Comparison, c.Expected, err)
		}
	}
}

// TestGEAssertion tests the "greater than or equal to" comparison rule
func TestGEAssertion(t *testing.T) {
	type testCase struct {
		Value      interface{}
		Comparison interface{}
		Expected   bool
	}

	cases := []testCase{
		testCase{Value: "123", Comparison: 123, Expected: true},
		testCase{Value: "123", Comparison: 124, Expected: false},
		testCase{Value: 123, Comparison: 124, Expected: false},
		testCase{Value: 123, Comparison: 122, Expected: true},
		testCase{Value: 123, Comparison: 123, Expected: true},
	}

	for _, c := range cases {
		err := checkAssertions(c.Value, map[string]interface{}{"ge": c.Comparison})
		if (err == nil) != c.Expected {
			t.Errorf("failed: expected %v >= %v to have been %v; %v", c.Value, c.Comparison, c.Expected, err)
		}
	}
}
