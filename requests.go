package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"text/template"
)

// request makes an http client request and checks the response body and response status
// against any Expect conditions provided
func request(request Request, count int, envMap map[string]interface{}) error {

	method := strings.ToUpper(request.Method)
	expect := request.Expect

	// setRequestEnvironment will use Go's text templates to replace values in the URL and expect specs
	// with provided values in the envMap
	url, err := setRequestEnvironment(request.URL, envMap)
	if err != nil {
		return err
	}

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
	if resp.StatusCode != expect.Status {
		log.Printf("  FAIL expected: %v received: %v", expect.Status, resp.StatusCode)
	} else {
		log.Printf("  ✓  status is %v", resp.StatusCode)
	}

	// start checking JSON values, first checking that content type is application/json
	if !contains(resp.Header["Content-Type"], "application/json") {
		return errors.New("response body not JSON, skipping JSON value checks")
	}

	body := make(map[string]interface{})
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		log.Println(err)
		return fmt.Errorf("ERROR %s %s could not decode response body", method, url)
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
			log.Printf("  ✓  %v equal to: %v", v.Key, v.Value)
		}

	}

	// Set user vars (defined by a `set:` block in the request spec)
	for _, v := range request.SetVars {
		envMap[v.Name] = fmt.Sprintf("%v", body[v.Key])
	}

	if failCount > 0 {
		return fmt.Errorf("  %v failing conditions", failCount)
	}

	// request tests passed, return nil error
	return nil
}

// setRequestEnvironment takes a url and an Expect struct and modifies them according to the
// values in the envMap, which contains some user defined values (or automatically updated values).
// it returns back a new url and Expect struct with any "template tags", e.g. {{ }}, replaced.
func setRequestEnvironment(url string, envMap map[string]interface{}) (string, error) {

	var urlBuffer bytes.Buffer
	urlTemplate, err := template.New("url").Parse(url)
	if err != nil {
		return "", err
	}

	err = urlTemplate.Execute(&urlBuffer, envMap)
	if err != nil {
		return "", err
	}
	url = urlBuffer.String()

	return url, nil
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

// contains is a helper function to check if a slice of strings contains a particular string.
// each string in the slice need only contain a substring, a full match is not necessary
func contains(s []string, substring string) bool {
	for _, item := range s {
		if strings.Contains(item, substring) {
			return true
		}
	}
	return false
}
