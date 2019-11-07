package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// basicRequestHandler is a test handler that returns different responses
// to HTTP requests
func basicRequestHandler(w http.ResponseWriter, req *http.Request) {

	// todo contains some example data used for testing a "todo app"
	type todo struct {
		ID          int    `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		NumTasks    int    `json:"num_tasks"`
	}

	data := todo{
		ID:          1,
		Title:       "delectus aut autem",
		Description: "something to do",
		NumTasks:    2,
	}

	switch req.Method {
	case "POST":
		// mock route requiring authentication
		if !contains(req.Header["Authorization"], "Bearer secret123") {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(data)
		return
	case "GET":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(data)
		return
	}
	w.Header().Set("Allow", "GET, POST")
	http.Error(w, http.StatusText(405), 405)
	return
}

func TestRequestSet(t *testing.T) {
	// set up httptest server to handle test requests
	handler := http.HandlerFunc(basicRequestHandler)
	server := httptest.NewServer(handler)
	defer server.Close()

	set, err := readTestDefinition("test/test.yaml")
	if err != nil {
		t.Error("Error reading test yaml file:", err)
	}

	// rewrite the "host" variable to be the mock server
	set.Environment.Vars["host"] = server.URL

	// this is fragile, and will fail if more requests are added to the test.yaml file
	// todo:  rework test to focus more on logic, less on yaml file staying the same.
	expectedTotal, expectedFails := 2, 0
	// the third argument is the test request name to run, and an empty string means all tests.
	total, fails := runRequests(set.Requests, set.Environment, "", false, false)

	if total != expectedTotal {
		t.Errorf("Expected '%v', received '%v'", expectedTotal, total)
	}

	if fails != expectedFails {
		t.Errorf("Expected '%v', received '%v'", expectedFails, fails)
	}
}

func TestRequestSingle(t *testing.T) {
	// set up httptest server to handle test requests
	handler := http.HandlerFunc(basicRequestHandler)
	server := httptest.NewServer(handler)
	defer server.Close()

	set, err := readTestDefinition("test/test.yaml")
	if err != nil {
		t.Error("Error reading test yaml file:", err)
	}

	// rewrite the "host" variable to be the mock server
	set.Environment.Vars["host"] = server.URL

	// name of the test in test.yaml to run.  All other tests not matching
	// this name should be skipped.
	testName := "Todo list"

	// this is fragile, and will fail if more requests are added to the test.yaml file
	// todo:  rework test to focus more on logic, less on yaml file staying the same.
	expectedTotal, expectedFails := 1, 0
	total, fails := runRequests(set.Requests, set.Environment, testName, false, false)

	if total != expectedTotal {
		t.Errorf("Expected '%v', received '%v'", expectedTotal, total)
	}

	if fails != expectedFails {
		t.Errorf("Expected '%v', received '%v'", expectedFails, fails)
	}
}

func TestSetRequestVars(t *testing.T) {

	testVars := make(map[string]interface{})
	testHeaders := make(map[string]string)

	url := `{{.host}}/api/v1/posts`
	testVars["host"] = "localhost:8000"

	expectedParsedURL := `localhost:8000/api/v1/posts`

	testHeaders["Authorization"] = `Bearer {{.token}}`
	testVars["token"] = "token"
	expectedParsedHeader := `Bearer token`

	parsedURL, err := replaceURLVars(url, testVars)

	if err != nil {
		t.Error("error trying to parse url")
	}

	parsedHeaders, err := setRequestHeaders(testHeaders, testVars)
	if err != nil {
		t.Error("error trying to parse headers")
	}

	if parsedHeaders["Authorization"] != expectedParsedHeader {
		t.Errorf("Expected '%v', received '%v'", expectedParsedHeader, parsedHeaders["Authorization"])
	}

	if parsedURL != expectedParsedURL {
		t.Errorf("Expected '%v', received '%v'", expectedParsedURL, parsedURL)
	}
}

func TestStrictJSONComparison(t *testing.T) {
	type testCase struct {
		JSON        []byte
		Key         string
		Expected    interface{}
		ExpectEqual bool
	}

	cases := []testCase{
		// NOTE: entering typed values here is not the same as unmarshalling typed values
		// from JSON/YAML. Numbers unmarshal to float64 (todo: verify) so for these test cases, they should
		// be entered as floats, not integers.
		testCase{JSON: []byte(`{"foo":123}`), Key: "foo", Expected: 123., ExpectEqual: true},
		testCase{JSON: []byte(`{"foo":"234"}`), Key: "foo", Expected: "234", ExpectEqual: true},
		testCase{JSON: []byte(`{"foo":345}`), Key: "foo", Expected: "345", ExpectEqual: false},
		testCase{JSON: []byte(`{"foo":"1234"}`), Key: "foo", Expected: "12345", ExpectEqual: false},
	}

	for _, c := range cases {
		err := checkJSONResponse(c.JSON, c.Key, c.Expected, true)
		if (err == nil) != c.ExpectEqual {
			t.Errorf("failed: %s; expected key %s == %v to have been %v; %v", c.JSON, c.Key, c.Expected, c.ExpectEqual, err)
		}
	}
}

func TestNonStrictJSONComparison(t *testing.T) {
	type testCase struct {
		JSON        []byte
		Key         string
		Expected    interface{}
		ExpectEqual bool
	}

	cases := []testCase{
		testCase{JSON: []byte(`{"foo":123}`), Key: "foo", Expected: 123, ExpectEqual: true},
		testCase{JSON: []byte(`{"foo":"123"}`), Key: "foo", Expected: "123", ExpectEqual: true},
		testCase{JSON: []byte(`{"foo":123}`), Key: "foo", Expected: "123", ExpectEqual: true},
		testCase{JSON: []byte(`{"foo":"1234"}`), Key: "foo", Expected: "12345", ExpectEqual: false},
		testCase{JSON: []byte(`{"foo":{"bar":"12345", "unused":"54321"}}`), Key: "foo.bar", Expected: "12345", ExpectEqual: true},
		testCase{JSON: []byte(`{"foo":{"bar":"12345"}}`), Key: "foo.bar", Expected: "asdf", ExpectEqual: false},
	}

	for _, c := range cases {
		err := checkJSONResponse(c.JSON, c.Key, c.Expected, false)
		if (err == nil) != c.ExpectEqual {
			t.Errorf("failed: %s; expected key %s == %v to have been %v; %v", c.JSON, c.Key, c.Expected, c.ExpectEqual, err)
		}
	}
}
