package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// request makes an http client request and checks the response body and response status
// against any Expect conditions provided
func request(url string, method string, expect Expect, count int) error {
	log.Printf("%v.  %s %s", count, method, url)

	client := &http.Client{}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check that status code matches the expected value, return with an error message on fail
	// TODO: break this out into a more general function that handles different assertions
	if resp.StatusCode != expect.Status {
		log.Printf("  FAIL expected: %v received: %v", expect.Status, resp.StatusCode)
	}

	body := make(map[string]string)
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		log.Println(err)
		return fmt.Errorf("  ERROR %s %s could not decode response body", method, url)
	}

	failCount := 0

	// Check for JSON values
	for _, v := range expect.Values {
		err := checkJSONResponse(body[v.Key], v.Value)
		if err != nil {
			failCount++
			log.Printf("  FAIL  %s equal to: %s, received: %s", v.Key, v.Value, body[v.Key])
		} else {
			log.Printf("  OK  %s equal to: %s", v.Key, v.Value)
		}

	}

	if failCount > 0 {
		return fmt.Errorf("  %v failing conditions", failCount)
	}

	// request tests passed, return nil error
	return nil
}

// checkJSONResponse compares a key and expected value to a map of a response body
// TODO: this basic implementation only supports strings and flat JSON responses.
func checkJSONResponse(value string, expectedValue string) error {
	if value != expectedValue {
		return fmt.Errorf("expected: %v received: %v", expectedValue, value)
	}
	return nil
}
