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
		ID              int    `json:"id"`
		TodoTitle       string `json:"todo_title"`
		TodoDescription string `json:"todo_description"`
	}

	data := todo{
		ID:              123,
		TodoTitle:       "Clean the house",
		TodoDescription: "It's time to clean up around here",
	}

	switch req.Method {
	case "POST":

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

	// rewrite the URL for each request in the test yaml file to point at the httptest server
	for i := range set.Requests {
		set.Requests[i].URL = server.URL
	}

	// this is fragile, and will fail if more requests are added to the test.yaml file
	// todo:  rework test to focus more on logic, less on yaml file staying the same.
	expectedTotal, expectedFails := 2, 0
	total, fails := runRequests(set.Requests)

	if total != expectedTotal {
		t.Errorf("Expected '%v', received '%v'", expectedTotal, total)
	}

	if fails != expectedFails {
		t.Errorf("Expected '%v', received '%v'", expectedFails, fails)
	}

}
