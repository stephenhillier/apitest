package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"

	"github.com/hashicorp/hcl"
	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

// TestSet is a set of requests and assertions
type TestSet struct {
	Requests []Request              `yaml:"requests" hcl:"request"`
	Vars     map[string]interface{} `yaml:"vars"`
	Headers  map[string]string      `yaml:"headers"`
}

// Request is a request made against a URL to test the response.
// The response will be checked against the conditions in the Expect struct
type Request struct {
	Name    string                 `yaml:"name"`
	URL     string                 `yaml:"url"`
	Method  string                 `yaml:"method"`
	Body    map[string]interface{} `yaml:"body"`
	Expect  Expect                 `yaml:"expect"`
	SetVars map[string]string      `yaml:"set" hcl:"set"`
}

// Expect is a test assertion.  The values provided will be checked against the request's response.
type Expect struct {
	// Status is the response status code, e.g. 200 for "OK", 404 for "Not Found"
	Status int                    `yaml:"status"`
	Values map[string]interface{} `yaml:"values"`
	Strict bool                   `yaml:"strict"`
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

	// process template tags.  a period will be pre-pended to the argument: {{ myVar }} becomes {{ .myVar }}
	// so that the text/template package can process them.
	r, err := regexp.Compile(`{{\s*(\w+)\s*}}`)
	if err != nil {
		return TestSet{}, errors.New("error processing {{ template }} tags. Please double check input file")
	}
	file = r.ReplaceAll(file, []byte(`{{.$1}}`))

	// check for HCL extension (note: testing if using HCL is practical)
	fileExt := filepath.Ext(filename)
	if fileExt == ".hcl" {
		err = hcl.Unmarshal(file, &set)
		if err != nil {
			return TestSet{}, fmt.Errorf("Unmarshal HCL: %v", err)
		}
	} else {
		// convert yaml to structs.
		// the output should be a single TestSet with nested Request structs.
		err = yaml.Unmarshal(file, &set)
		if err != nil {
			return TestSet{}, fmt.Errorf("Unmarshal YAML: %v", err)
		}
	}

	log.Println(set)

	return set, nil
}

// runRequests accepts a set of Request objects and calls the request() function
// for each one. Since requests are expected to fail often, errors are not passed
// up to the calling function, but instead reported to output, tallied
// and the total request & error counts returned at the end of the run.
func runRequests(requests []Request, vars map[string]interface{}, headers map[string]string, testname string) (int, int) {
	failCount := 0
	currentRequest := 0

	// iterate through requests and keep track of test fails
	for _, r := range requests {
		// if a test name was provided, skip this test request if it does not match.
		if testname != "" && testname != r.Name {
			continue
		}

		currentRequest++
		err := request(r, currentRequest, vars, headers)
		if err != nil {
			log.Println("  ", err)
			failCount++
		}
	}
	return currentRequest, failCount
}

func main() {
	var filename string
	var testname string
	flag.StringVarP(&filename, "file", "f", "", "yaml file containing a list of test requests")
	flag.StringVarP(&testname, "test", "t", "", "the name of a single test to run (use quotes if name has spaces)")
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
	totalRequests, failCount := runRequests(set.Requests, set.Vars, set.Headers, testname)

	log.Println("Total requests:", totalRequests)

	if failCount > 0 {
		log.Fatalf("FAIL  %s (%v requests, %v failed)", filename, totalRequests, failCount)
	}
	log.Printf("PASSED  %s (%v requests)", filename, totalRequests)
}
