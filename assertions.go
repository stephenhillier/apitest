package main

import (
	"fmt"
	"strconv"
)

// checkAssertions takes the rules provided in the test spec and iterates over them,
// returning an error if at any point a comparison is false.
// value comparisons:
// "equals"
// "lt" (less than)
// "gt" (greater than)
// "le" (less than or equal to)
// "ge" (greater than or equal to)
// "exists" (key exists in the JSON response body)
func checkAssertions(value interface{}, rules map[string]interface{}) error {
	for k, comparisonValue := range rules {
		switch k {
		case "equals":
			// compare values using string formatting. This is equivalent to "strict=false"
			// e.g.:
			// 123 == 123 > true
			// 123 == "123" > true

			if !equals(value, comparisonValue) {
				return fmt.Errorf("expected: %v received: %v", comparisonValue, value)
			}
		case "lt":
			// convert to floats, returning an error if that's not possible (bad input)
			val1, val2, err := asFloat(value, comparisonValue)
			if err != nil {
				return err
			}

			// perform comparison
			if val1 >= val2 {
				return fmt.Errorf("expected %v less than %v", value, comparisonValue)
			}
		case "gt":
			val1, val2, err := asFloat(value, comparisonValue)
			if err != nil {
				return err
			}

			if val1 <= val2 {
				return fmt.Errorf("expected %v greater than %v", value, comparisonValue)
			}
		case "le":
			val1, val2, err := asFloat(value, comparisonValue)
			if err != nil {
				return err
			}

			if val1 > val2 {
				return fmt.Errorf("expected %v less than or equal to %v", value, comparisonValue)
			}
		case "ge":
			val1, val2, err := asFloat(value, comparisonValue)
			if err != nil {
				return err
			}

			if val1 < val2 {
				return fmt.Errorf("expected %v greater than or equal to %v", value, comparisonValue)
			}
		case "exists":
			// check whether this key was received as part of the body, even if null.
			// not elegant, but due to the jq parsing in checkJSONResponse(), this api test case will
			// fail earlier in checkJSONResponse if the key doesn't exist. Therefore, if the test case
			// gets this far, we already know the key exists. This is here to provide a means to check
			// the "exists" case without having to provide a comparison value. In the future, I need
			// to refactor to allow `exists: false`.
		default:
			// invalid rule (not defined above)
			return fmt.Errorf("invalid rule: %s", k)
		}
	}
	return nil
}

// equals returns true if two values are equal, when cast to strings.
// this means that 123 == "123" (int vs string).
func equals(val interface{}, comparison interface{}) bool {
	return fmt.Sprintf("%v", val) == fmt.Sprintf("%v", comparison)
}

// asFloat converts two values (received value and comparison value) to floats.
// returns an error if not possible.
func asFloat(val1 interface{}, val2 interface{}) (float64, float64, error) {
	float1, err := strconv.ParseFloat(fmt.Sprintf("%v", val1), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to parse %v as a float", val1)
	}
	float2, err := strconv.ParseFloat(fmt.Sprintf("%v", val2), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to parse %v as a float", val2)
	}
	return float1, float2, nil
}
