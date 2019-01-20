package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"

	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

// TestSet is a set of requests and assertions
type TestSet struct {
	Requests    []Request              `yaml:"requests"`
	Environment map[string]interface{} `yaml:"environment"`
}

// Request is a request made against a URL to test the response.
// The response will be checked against the conditions in the Expect struct
type Request struct {
	Name    string                 `yaml:"name"`
	URL     string                 `yaml:"url"`
	Method  string                 `yaml:"method"`
	Body    map[string]interface{} `yaml:"body"`
	Expect  Expect                 `yaml:"expect"`
	SetVars []UserVar              `yaml:"set"`
}

// Expect is a test assertion.  The values provided will be checked against the request's response.
type Expect struct {
	// Status is the response status code, e.g. 200 for "OK", 404 for "Not Found"
	Status int                    `yaml:"status"`
	Values map[string]interface{} `yaml:"values"`
}

// UserVar holds a value (string) and a type. It allows users to
// store values from one request to the next (e.g. a token received after a login request, or
// the ID or other response value from a created resource)
type UserVar struct {
	Key  string `yaml:"from"`
	Name string `yaml:"var"`
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
	totalRequests, failCount := runRequests(set.Requests, set.Environment)

	log.Println("Total requests:", totalRequests)

	if failCount > 0 {
		log.Fatalf("FAIL  %s (%v requests, %v failed)", filename, totalRequests, failCount)
	}
	log.Printf("PASSED  %s (%v requests)", filename, totalRequests)
}

// readTestDefinition reads a yaml file of test requests
// and returns a TestSet.  If an error occurs while reading the file
// or unmarshaling yaml, an empty test set and an error will be returned.
func readTestDefinition(filename string) (TestSet, error) {
	set := TestSet{}

	// Read in a yaml file containing test specs
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return TestSet{}, fmt.Errorf("File open error %v ", err)
	}

	// process template tags.  a period will be pre-pended to the argument {{ myVar }} becomes {{ .myVar }}
	// so that the text/template package can process them.
	r, err := regexp.Compile(`{{\s*(\w+)\s*}}`)
	if err != nil {
		return TestSet{}, errors.New("error processing {{ template }} tags. Please double check input file")
	}
	file = r.ReplaceAll(file, []byte(`{{.$1}}`))

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
func runRequests(requests []Request, envMap map[string]interface{}) (totalRequests int, failCount int) {
	totalRequests = len(requests)
	currentRequest := 1

	// create a mapping for user-defined variables within the test.
	// these are necessary for setting a value during one test (e.g, an auth token
	// or an ID for a created resource) and then referring to it during later tests.

	for _, r := range requests {
		err := request(r, currentRequest, envMap)
		if err != nil {
			log.Println("  ", err)
			failCount++
		}
		currentRequest++
	}
	return totalRequests, failCount
}
