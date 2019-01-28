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
func request(request Request, count int, env Environment, verbose bool) error {

	method := strings.ToUpper(request.Method)
	expect := request.Expect

	// replace template tags/variables in the URL
	url, err := replaceURLVars(request.URL, env.Vars)
	if err != nil {
		return err
	}

	// copy original headers into a new map
	headers := make(map[string]string)
	for k, v := range env.Headers {
		headers[k] = v
	}

	// replace variables in the headers
	headers, err = setRequestHeaders(headers, env.Vars)
	if err != nil {
		return err
	}

	log.Printf("%v. %s", count, request.Name)
	log.Println(" ", method, url)

	// process template tags/variables in the request body and
	// store as a new variable
	bodyJSON, err := replaceBodyVars(request.Body, env.Vars)
	if err != nil {
		return err
	}

	reqBody, err := json.Marshal(bodyJSON)
	if err != nil {
		return errors.New("error serializing request body as JSON")
	}

	// replace variables in the request body
	bodyBuffer := bytes.NewBuffer(reqBody)
	client := &http.Client{}
	req, err := http.NewRequest(method, url, bodyBuffer)
	if err != nil {
		return err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	if verbose {
		log.Println(req)
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

		err := checkJSONResponse(jsonValue, v, request.Expect.Strict)
		if err != nil {
			failCount++
			log.Println("  FAIL,", k, err)
		} else {
			log.Printf("  ✓  %v equal to: %v", k, v)
		}

	}

	if verbose {
		log.Println(body)
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

// replaceVars takes a string with template tags and a map of variables and uses the
// text/template package to replace the template variables.
// It returns back a new string.
func replaceURLVars(url string, vars map[string]interface{}) (string, error) {

	var urlBuffer bytes.Buffer

	// URL template tag variable replacement
	// parse URL string with text/template, and return a new
	// string with any {{ variables }} replaced with the values in the
	// vars map.
	urlTemplate, err := template.New("url").Parse(url)
	if err != nil {
		return url, err
	}

	err = urlTemplate.Execute(&urlBuffer, vars)
	if err != nil {
		return url, err
	}
	url = urlBuffer.String()

	return url, nil
}

// setRequestHeaders replaces all variables in each header.
// the headers map is stringified first, then variables are replaced,
// and then the headers are marshalled back to a map[string]string.
func setRequestHeaders(headers map[string]string, vars map[string]interface{}) (map[string]string, error) {

	var headerBuffer bytes.Buffer

	headerJSON, err := json.Marshal(headers)
	if err != nil {
		return headers, err
	}

	headerTemplate, err := template.New("header").Parse(string(headerJSON))
	if err != nil {
		return headers, err
	}

	err = headerTemplate.Execute(&headerBuffer, vars)
	if err != nil {
		return headers, err
	}

	err = json.Unmarshal(headerBuffer.Bytes(), &headers)
	if err != nil {
		return headers, err
	}

	return headers, nil
}

// replaceBodyVars replaces all variables in the request body.
// interface{} is used here due to the unknown schema in the test spec file.
func replaceBodyVars(body map[string]interface{}, vars map[string]interface{}) (map[string]interface{}, error) {

	var bodyBuffer bytes.Buffer

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return body, err
	}

	headerTemplate, err := template.New("body").Parse(string(bodyJSON))
	if err != nil {
		return body, err
	}

	err = headerTemplate.Execute(&bodyBuffer, vars)
	if err != nil {
		return body, err
	}

	err = json.Unmarshal(bodyBuffer.Bytes(), &body)
	if err != nil {
		return body, err
	}

	return body, nil
}

// checkJSONResponse compares two values of arbitrary type.
// The values are considered equal if their string representation is the same (no type comparison)
// This could be made more strict by directly comparing the interface{} values.
func checkJSONResponse(value interface{}, expectedValue interface{}, strict bool) error {

	// use the Sprintf method to convert our value and expectedValue to strings so they can be
	// directly compared.
	sValue := fmt.Sprintf("%v", value)
	sExpected := fmt.Sprintf("%v", expectedValue)

	if strict {
		if value != expectedValue {
			return fmt.Errorf("expected: %v (%T) received: %v (%T)", expectedValue, expectedValue, value, value)
		}

		return nil
	}

	// not strict: compare against string representation of value

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
