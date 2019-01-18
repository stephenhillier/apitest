package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

// TestSet is a set of requests and assertions
type TestSet struct {
	Requests []Request `yaml:"requests"`
}

// Request is a request made against a URL to test the response.
// The response will be checked against the conditions in the Expect struct
type Request struct {
	Name   string `yaml:"name"`
	URL    string `yaml:"url"`
	Method string `yaml:"method"`
	Expect Expect `yaml:"expect"`
}

// Expect is a test assertion.  The values provided will be checked against the request's response.
type Expect struct {
	// Status is the response status code, e.g. 200 for "OK", 404 for "Not Found"
	Status int         `yaml:"status"`
	Values []JSONValue `yaml:"values"`
}

// JSONValue is an expected value received as part of a JSON response
// e.g.  {"key": "value"}
type JSONValue struct {
	Key   string      `yaml:"key"`
	Value interface{} `yaml:"value"`
}

func main() {
	var filename string
	flag.StringVarP(&filename, "file", "f", "", "yaml file containing a list of test requests")
	flag.Parse()

	if filename == "" {
		log.Fatal("Usage:  apitest -f test.yaml")
	}

	// read in test definitions from a provided yaml file
	set, err := readTestDefinition(filename)
	if err != nil {
		log.Fatalln(err)
	}

	// run the set of tests and exit the program.
	// additional output will be provided by each request.
	// TODO: handle multiple test suites
	log.Println("Running tests...")
	totalRequests, failCount := runRequests(set.Requests)

	log.Println("Total requests:", totalRequests)

	if failCount > 0 {
		log.Fatalf("FAIL  %s (%v requests, %v failed)", filename, totalRequests, failCount)
	}
	log.Printf("OK  %s (%v requests)", filename, totalRequests)
}

// readTestDefinition reads a yaml file of test requests
// and returns a TestSet.  If an error occurs while reading the file
// or unmarshaling yaml, an empty test set and an error will be returned.
func readTestDefinition(filename string) (TestSet, error) {
	set := TestSet{}

	// Read in a yaml file containing
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return TestSet{}, fmt.Errorf("File open error %v ", err)
	}

	// convert yaml to structs.
	// the output should be a single TestSet with nested Request structs.
	err = yaml.Unmarshal(file, &set)
	if err != nil {
		return TestSet{}, fmt.Errorf("Unmarshal: %v", err)
	}

	return set, nil
}

// runRequests accepts a set of Request objects and calls the request() function
// for each one. Since requests are expected to fail often, errors are not passed
// up to the calling function, but instead reported to output, tallied
// and the total request & error counts returned at the end of the run.
func runRequests(requests []Request) (totalRequests int, failCount int) {
	totalRequests = len(requests)
	currentRequest := 1

	for _, r := range requests {
		err := request(r.URL, strings.ToUpper(r.Method), r.Expect, currentRequest)
		if err != nil {
			log.Println("  ", err)
			failCount++
		}
		currentRequest++
	}
	return totalRequests, failCount
}
