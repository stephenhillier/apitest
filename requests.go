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

	body := make(map[string]interface{})
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		log.Println(err)
		return fmt.Errorf("  ERROR %s %s could not decode response body", method, url)
	}

	failCount := 0

	// Check for JSON values
	for _, v := range expect.Values {

		jsonValue := body[v.Key]

		err := checkJSONResponse(v.Key, jsonValue, v.Value)
		if err != nil {
			failCount++
			log.Println("  FAIL,", v.Key, err)
		} else {
			log.Printf("  OK  %v equal to: %v", v.Key, v.Value)
		}

	}

	if failCount > 0 {
		return fmt.Errorf("  %v failing conditions", failCount)
	}

	// request tests passed, return nil error
	return nil
}

// checkJSONResponse compares a key and expected value to a map of a response body
func checkJSONResponse(key string, value interface{}, expectedValue interface{}) error {

	// use the Sprintf method to convert our value and expectedValue to strings so they can be
	// directly compared.
	sValue := fmt.Sprintf("%v", value)
	sExpected := fmt.Sprintf("%v", expectedValue)

	if sValue != sExpected {
		return fmt.Errorf("expected: %v received: %v", sExpected, sValue)
	}

	return nil
}
