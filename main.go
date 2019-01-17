package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"gopkg.in/yaml.v2"
)

// TestSet is a set of requests and assertions contained within a file of tests
type TestSet struct {
	Requests []Request `yaml:"requests"`
}

// Request represents a request made against a URL.
// The response will be checked against any Expect instances contained
// within the Expect slice.
type Request struct {
	Name   string `yaml:"name"`
	URL    string `yaml:"url"`
	Method string `yaml:"method"`
	Expect Expect `yaml:"expect"`
}

// Expect is a test assertion.  The values provided will be checked against the request's response.
type Expect struct {
	Status int
}

func main() {

	set := TestSet{}

	file, err := ioutil.ReadFile("test/test.yaml")
	if err != nil {
		log.Fatalf("File open error %v ", err)
	}
	err = yaml.Unmarshal(file, &set)

	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	log.Println("Running tests...")

	totalRequests := len(set.Requests)
	currentRequest := 1
	failCount := 0

	for _, r := range set.Requests {
		err = request(r.URL, strings.ToUpper(r.Method), r.Expect, currentRequest)
		if err != nil {
			log.Println(err)
			failCount++
		}
		currentRequest++
	}

	log.Println("Total requests:", totalRequests)
	log.Println("Requests with failing assertions:", failCount)

}

// request makes an http client request and checks the response body and response status
// against any Expect conditions provided
func request(url string, method string, expect Expect, count int) error {
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

	if resp.StatusCode != expect.Status {
		return fmt.Errorf("%v.   FAIL  %s %s Expected: %v Received: %v", count, method, url, expect.Status, resp.StatusCode)
	}
	log.Printf("%v.   OK    %s %s %v", count, method, url, resp.StatusCode)

	// for _, e := range expect.JSONValues {
	// }

	return nil
}
