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
	}

	data := todo{
		ID:          1,
		Title:       "delectus aut autem",
		Description: "something to do",
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
	total, fails := runRequests(set.Requests, set.Environment)

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

	parsedURL, parsedHeaders, err := setRequestVars(url, testHeaders, testVars)

	if err != nil {
		t.Error("error trying to parse url and headers")
	}

	if parsedHeaders["Authorization"] != expectedParsedHeader {
		t.Errorf("Expected '%v', received '%v'", expectedParsedHeader, parsedHeaders["Authorization"])
	}

	if parsedURL != expectedParsedURL {
		t.Errorf("Expected '%v', received '%v'", expectedParsedURL, parsedURL)
	}
}
