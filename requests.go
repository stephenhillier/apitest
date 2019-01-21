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
func request(request Request, count int, env Environment) error {

	method := strings.ToUpper(request.Method)
	expect := request.Expect

	// setRequestVars will use Go's text templates to replace values in the URL and expect specs
	// with provided values in the envMap
	url, headers, err := setRequestVars(request.URL, env.Headers, env.Vars)
	if err != nil {
		return err
	}

	log.Printf("%v.  %s %s", count, method, url)

	reqBody, err := json.Marshal(request.Body)
	if err != nil {
		return errors.New("error serializing request body as JSON")
	}

	bodyBuffer := bytes.NewBuffer(reqBody)
	client := &http.Client{}
	req, err := http.NewRequest(method, url, bodyBuffer)
	if err != nil {
		return err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
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
	for k, v := range expect.Values {

		jsonValue := body[k]

		err := checkJSONResponse(k, jsonValue, v)
		if err != nil {
			failCount++
			log.Println("  FAIL,", k, err)
		} else {
			log.Printf("  ✓  %v equal to: %v", k, v)
		}

	}

	// Set user vars (defined by a `set:` block in the request spec)
	for _, v := range request.SetVars {
		env.Vars[v.Name] = fmt.Sprintf("%v", body[v.Key])
	}

	if failCount > 0 {
		return fmt.Errorf("  %v failing conditions", failCount)
	}

	// request tests passed, return nil error
	return nil
}

// setRequestVars takes a url, header set and a map of variables and modifies them according to the
// variable map, which contains some user defined values (or automatically updated values).
// it returns back a new url and headers with any "template tags", e.g. {{ }}, replaced.
func setRequestVars(url string, headers map[string]string, vars map[string]interface{}) (string, map[string]string, error) {

	var urlBuffer bytes.Buffer
	var headerBuffer bytes.Buffer

	// URL template tag variable replacement
	// parse URL string with text/template, and return a new
	// string with any {{ variables }} replaced with the values in the
	// vars map.
	urlTemplate, err := template.New("url").Parse(url)
	if err != nil {
		return url, headers, err
	}

	err = urlTemplate.Execute(&urlBuffer, vars)
	if err != nil {
		return url, headers, err
	}
	url = urlBuffer.String()

	// Replace all variables in each header.
	// the headers map is stringified first, then variables are replaced,
	// and then the headers are marshalled back to a map[string]string.
	// This is probably inefficient but is flexible
	headerJSON, err := json.Marshal(headers)
	if err != nil {
		return url, headers, err
	}

	headerTemplate, err := template.New("header").Parse(string(headerJSON))
	if err != nil {
		return url, headers, err
	}

	err = headerTemplate.Execute(&headerBuffer, vars)
	if err != nil {
		return url, headers, err
	}

	err = json.Unmarshal(headerBuffer.Bytes(), &headers)
	if err != nil {
		return url, headers, err
	}

	return url, headers, nil
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
