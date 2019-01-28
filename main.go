package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strings"

	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

// TestSet is a set of requests and assertions
type TestSet struct {
	Requests    []Request   `yaml:"requests"`
	Environment Environment `yaml:"environment"`
}

// Environment stores defaults to use with each request.
// Vars holds variables to be inserted (using the text/template package)
// into request specs (e.g. http://{{hostname}}/api/posts). Vars may be updated
// after a request if the input request spec has a "set" block.
// Headers can contain variables.
type Environment struct {
	Vars    map[string]interface{} `yaml:"vars"`
	Headers map[string]string      `yaml:"headers"`
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
	Strict bool                   `yaml:"strict"`
}

// UserVar holds a value (string) and a type. The key/value pair will be copied to the
// Environment.Vars map. This allows users to store values from one request to the next
// (e.g. a token received after a login request, or the ID or other response value from
// a created resource)
type UserVar struct {
	Key  string `yaml:"from"`
	Name string `yaml:"var"`
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
	// so that the text/template package can replace them with variables.
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
func runRequests(requests []Request, env Environment, testname string, verbose bool) (int, int) {
	failCount := 0
	currentRequest := 0

	// iterate through requests and keep track of test fails
	for _, r := range requests {
		// if a test name was provided, skip this test request if it does not match.
		if testname != "" && testname != r.Name {
			continue
		}

		currentRequest++
		err := request(r, currentRequest, env, verbose)
		if err != nil {
			log.Println("  ", err)
			failCount++
		}
	}
	return currentRequest, failCount
}

// processEnvVars iterates through userVars provided through the command line
// (in the form -e myvar="my var" or -e token=$API_TOKEN) and puts them
// into the test environment. They can be accessed through template tags
// e.g. {{ myvar }}
func (env Environment) processEnvVars(userVars []string) error {

	for _, s := range userVars {
		// key/values separated by `=` are split and put into the map of env variables (env.Vars)
		pair := strings.Split(s, "=")
		if len(pair) != 2 {
			return errors.New("Error processing env vars.  Usage example: -e myvar=$MYVAR -e anothervar=$MYVAR2")
		}
		env.Vars[pair[0]] = pair[1]
	}
	return nil
}

func main() {
	var filename string
	var testname string
	var userVars []string
	var verbose bool
	flag.StringVarP(&filename, "file", "f", "", "yaml file containing a list of test requests")
	flag.StringVarP(&testname, "test", "t", "", "the name of a single test to run (use quotes if name has spaces)")
	flag.BoolVarP(&verbose, "verbose", "v", false, "verbose mode: print response body")
	flag.StringSliceVarP(&userVars, "env", "e", []string{}, "variables to add to the test environment e.g. myvar=test123")
	flag.Parse()

	if filename == "" {
		log.Fatal("No file specified. Usage:  apitest -f test.yaml")
	}

	// read in test definitions from a provided yaml file
	set, err := readTestDefinition(filename)
	if err != nil {
		log.Fatal(err)
	}

	// set variables in the test environment to values provided with the -e CLI flag.
	// these are starting values; it is possible to update them during a test run.
	err = set.Environment.processEnvVars(userVars)
	if err != nil {
		log.Fatal(err)
	}

	// run the set of tests and exit the program.
	// additional output will be provided by each request.
	// TODO: handle multiple test suites
	log.Println("Running tests...")
	totalRequests, failCount := runRequests(set.Requests, set.Environment, testname, verbose)

	log.Println("Total requests:", totalRequests)

	if failCount > 0 {
		log.Fatalf("FAIL  %s (%v requests, %v failed)", filename, totalRequests, failCount)
	}
	log.Printf("PASSED  %s (%v requests)", filename, totalRequests)
}
