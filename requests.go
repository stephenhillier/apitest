package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/savaki/jq"
)

// request makes an http client request and checks the response body and response status
// against any Expect conditions provided
func request(request Request, count int, env Environment, verbose bool) error {

	method := strings.ToUpper(request.Method)
	expect := request.Expect

	// replace template tags/variables in the URL
	reqURL, err := replaceURLVars(request.URL, env.Vars)
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
	log.Println(" ", method, reqURL)

	// set up request and client
	var req *http.Request
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Do not follow redirects
			return http.ErrUseLastResponse
		},
	}

	// If values are passed in by the "urlencoded" field, treat the request
	// as x-www-form-urlencoded
	if request.ContentType == "urlencoded" ||
		request.ContentType == "x-www-form-urlencoded" ||
		request.ContentType == "application/x-www-form-urlencoded" {

		headers["Content-Type"] = "application/x-www-form-urlencoded"

		form, err := replaceBodyVars(request.Body, env.Vars)
		if err != nil {
			return err
		}
		formData := url.Values{}
		for k, v := range form {
			formData.Set(k, fmt.Sprintf("%s", v))
		}

		req, err = http.NewRequest(method, reqURL, strings.NewReader(formData.Encode()))
		if err != nil {
			return err
		}
	} else {
		headers["Content-Type"] = "application/json"

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
		req, err = http.NewRequest(method, reqURL, bodyBuffer)
		if err != nil {
			return err
		}
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return fmt.Errorf("ERROR %s %s could not read response body", method, reqURL)
	}

	failCount := 0

	// Check that status code matches the expected value, return with an error message on fail
	if resp.StatusCode != expect.Status {
		if verbose {
			log.Printf("%s", body)
		}
		return fmt.Errorf("  FAIL expected: %v received: %v", expect.Status, resp.StatusCode)
	} else {
		log.Printf("  OK status is %v", resp.StatusCode)
	}

	// if the response is not JSON, end the request here.
	if !contains(resp.Header["Content-Type"], "application/json") {
		if verbose {
			log.Printf("%s", body)
		}
		return nil
	}

	// Handle verbose output (-v or --verbose flag) by unmarshalling to interface then marshalling
	// to indented JSON format
	var respBodyJSON interface{}
	err = json.Unmarshal(body, &respBodyJSON)
	if err != nil {
		log.Println(err)
		return fmt.Errorf("ERROR %s %s could not decode response body", method, reqURL)
	}

	if verbose {
		out, err := json.MarshalIndent(respBodyJSON, "", "  ")
		if err != nil {
			return fmt.Errorf("ERROR %s %s could not print response body in verbose mode", method, reqURL)
		}
		log.Printf("%s", out)
	}

	// Check for JSON values
	for k, v := range expect.Values {

		err := checkJSONResponse(body, k, v, request.Expect.Strict)
		if err != nil {
			failCount++
			log.Println("  FAIL,", k, err)
		} else {
			log.Printf("  ✓  %v equal to: %v", k, v)
		}

	}

	// Set user vars (defined by a `set:` block in the request spec)
	for _, v := range request.SetVars {
		selector := v.Key
		if c := fmt.Sprintf("%c", selector[0]); c != "." {
			selector = "." + selector
		}

		op, err := jq.Parse(selector)
		if err != nil {
			return fmt.Errorf("error setting variable from selector %s. Use jq format: e.g. foo or .foo.bar or foo.bar (all valid)", selector)
		}

		value, err := op.Apply(body)
		if err != nil {
			return fmt.Errorf("error finding value for key %s to use as variable. Key may not exist. Hint: Use jq format: e.g. foo or .foo.bar or foo.bar (all valid)", selector)
		}

		var setValue interface{}
		json.Unmarshal(value, &setValue)
		env.Vars[v.Name] = setValue
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
func checkJSONResponse(body []byte, selector string, expectedValue interface{}, strict bool) error {

	if c := fmt.Sprintf("%c", selector[0]); c != "." {
		selector = "." + selector
	}

	op, err := jq.Parse(selector)
	if err != nil {
		return fmt.Errorf("error processing selector %s. Use jq format: e.g. foo or .foo.bar or foo.bar (all valid)", selector)
	}

	value, err := op.Apply(body)
	if err != nil {
		return fmt.Errorf("error finding value for key selector %s. Key may not exist. Hint: Use jq format: e.g. foo or .foo.bar or foo.bar (all valid)", selector)
	}

	if strict {
		var strictIValue interface{}
		if err := json.Unmarshal(value, &strictIValue); err != nil {
			return fmt.Errorf("could not decode value from key %s", selector)
		}

		strictValue := fmt.Sprintf("%s", strictIValue)
		strictExpected := fmt.Sprintf("%s", expectedValue)

		if strictValue != strictExpected {
			return fmt.Errorf("expected: %v received: %v", strictExpected, strictValue)
		}

		return nil
	}

	// not strict: compare against string representation of value

	var iValue interface{}
	if err := json.Unmarshal(value, &iValue); err != nil {
		return fmt.Errorf("could not decode value from key %s", selector)
	}

	sValue := fmt.Sprintf("%v", iValue)
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

// toBytes accepts any value and returns the byte representation
func toBytes(key interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(key)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
