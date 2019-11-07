package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"

	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
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
	Name        string                 `yaml:"name"`
	URL         string                 `yaml:"url"`
	Method      string                 `yaml:"method"`
	ContentType string                 `yaml:"contentType"`
	Body        map[string]interface{} `yaml:"body"`
	Expect      Expect                 `yaml:"expect"`
	SetVars     []UserVar              `yaml:"set"`
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
func runRequests(requests []Request, env Environment, testname string, verbose bool, monitor bool) (int, int) {
	failCount := 0
	currentRequest := 0

	// iterate through requests and keep track of test fails
	for _, r := range requests {
		// if a test name was provided, skip this test request if it does not match.
		if testname != "" && testname != r.Name {
			continue
		}
		method := strings.ToUpper(r.Method)

		currentRequest++

		// make the request.
		// the hostname/path is parsed immediately so it's available for both
		// error handling and the "happy path"
		rawURL, duration, err := request(r, currentRequest, env, verbose)
		hostname, path := processURL(rawURL)
		if err != nil {
			// actions to take for unsuccessful requests
			log.Println("  ", err)
			failCount++
			if monitor {
				recordError(r.Name, hostname, path, method)
			}
		}

		durationSeconds := duration.Seconds()
		recordRequest(r.Name, hostname, path, method)
		recordDuration(r.Name, hostname, path, method, durationSeconds)

	}
	return currentRequest, failCount
}

func processURL(rawURL string) (string, string) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", ""
	}
	return u.Hostname(), u.EscapedPath()
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
	var monitor bool
	var listenPort int
	var delay int
	flag.StringVarP(&filename, "file", "f", "", "yaml file containing a list of test requests")
	flag.StringVarP(&testname, "test", "t", "", "the name of a single test to run (use quotes if name has spaces)")
	flag.BoolVarP(&verbose, "verbose", "v", false, "verbose mode: print response body")
	flag.BoolVarP(&monitor, "monitor", "m", false, "turn on monitor mode to continually run checks")
	flag.IntVarP(&listenPort, "port", "p", 2112, "port to start listener on (used with --monitor)")
	flag.StringSliceVarP(&userVars, "env", "e", []string{}, "variables to add to the test environment e.g. myvar=test123")
	flag.IntVarP(&delay, "delay", "d", 300, "delay (in seconds) between monitoring runs (used with --monitor). Default 300")
	flag.Parse()

	// user can enter filename as the first argument, or with the -f flag
	if flag.NArg() > 0 && filename == "" {
		filename = flag.Args()[0]
	}

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

	if !monitor {
		// run the set of tests and exit the program.
		// additional output will be provided by each request.
		// TODO: handle multiple test suites
		log.Println("Running tests...")
		totalRequests, failCount := runRequests(set.Requests, set.Environment, testname, verbose, monitor)

		log.Println("Total requests:", totalRequests)

		if failCount > 0 {
			log.Fatalf("FAIL  %s (%v requests, %v failed)", filename, totalRequests, failCount)
		}
		log.Printf("PASSED  %s (%v requests)", filename, totalRequests)
		os.Exit(0)
	}

	// using continuous monitor mode - set up metrics handler for Prometheus scraping
	handler := NewMetricsHandler()
	m := http.NewServeMux()
	m.Handle("/metrics", handler)
	h := http.Server{
		Addr:    fmt.Sprintf(":%v", listenPort),
		Handler: m,
	}

	// listen until receive an interrupt signal
	go func() {
		if err = h.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	log.Println("Listening on port", listenPort)

	// run monitoring loop
	go runMonitor(set.Requests, set.Environment, testname, verbose, filename, delay)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop
	log.Println("Shutting down...")
	h.Shutdown(context.Background())
	log.Println("Server stopped")

}

// runMonitor is used for monitoring mode and runs a continuous loop, checking the same
// test suite over and over for the purpose of collecting metrics and monitoring endpoints.
func runMonitor(requests []Request, env Environment, testname string, verbose bool, filename string, delay int) {
	for {
		totalRequests, failCount := runRequests(requests, env, testname, verbose, true)

		log.Println("Total requests:", totalRequests)

		if failCount > 0 {
			log.Printf("FAIL  %s (%v requests, %v failed)", filename, totalRequests, failCount)
		} else {
			log.Printf("PASSED  %s (%v requests)", filename, totalRequests)
		}

		time.Sleep(time.Duration(delay) * time.Second)
	}
}
