# apitest
Test the behavior of an HTTP-based REST API [from the command line](#command-line), in a container based pipeline like [GitHub Actions](#github-actions), or as a running service with continuous monitoring and [Prometheus metrics](#prometheus-usage) collection.

* [Defining test specs in YAML](#yaml-test-specs)
  * [Complete example](#complete-example)
* [Test specs syntax](#test-spec-properties)
* [Logging in / retrieving tokens](#logging-in)
* [jq style queries (for nested JSON)](#jq-style-json-parsing)
* [Command line usage](#command-line)
* [GitHub Actions usage](#github-actions)
* [Prometheus usage](#prometheus-usage)

## Usage

### YAML test specs

Define requests in YAML.  [See the test spec properties](#test-spec-properties) for more details on individual fields.

#### Complete example

`apitest -e token=secret123 test.yaml`

```yaml
# test.yaml
environment:
  vars:
    host: http://localhost:8000 # {{host}} will be replaced with this value
  headers: # all requests will include headers defined here
    Authorization: Bearer {{token}} # token was passed with the -e flag.
requests:
  - name: Add a comment
    url: "{{host}}/comments"
    method: post
    body: # this will be submitted as JSON in the request body
      comment: This is my comment! 
    expect:
      status: 201 # request will report as failed if 201 is not returned
    set:
      # set variables based on a field from the JSON response
      - var: created_comment # {{created_comment}} will be set/updated for further requests to use
        from: id
  - name: Get single comment
    url: "{{host}}/comments/{{created_comment}}"
    method: get
    expect:
      status: 200  
      values:
        comment: This is my comment! # the JSON response `comment` field must match this value
```

#### Test spec properties

`environment`: define defaults like request headers or starting values of variables.

  * `headers`: key/value pairs with any headers that should be added to each request.
  * `vars`: variables that can be accessed through template tags; e.g. `host: example.com` will be available as `{{host}}` in request URLs.  Currently only URLs and headers will accept variables, and strings starting with `{{ }}` may need to be surrounded by quotes to make sure they are parsed as a string.

```yaml
environment:
  vars:
    host: https://www.example.com/api/v1
    token: secret123
  headers:
    Authorization: Bearer {{token}}
```

`requests`: a list of requests to make as part of the test run.  Each request in the list can have the following properties:

  * `name`: a name for your request
  * `url`: the URL to make a request to
  * `method`: HTTP method e.g. GET, POST
  * `body`: key/value pairs that will be sent in the request body as JSON

```yaml
requests:
  - name: Add a joke
    url: "{{host}}/jokes"
    method: post
    body:
      joke: How did the Vikings send secret messages?
      punchline: By norse code!
```

  * `expect`: add simple checks to an expect block:  
    * `status`: HTTP status code  
    * `values`: key/value pairs 
    * `strict`: use `strict: true` to require expect & response type to be exactly the same (e.g. the integer `10` is not equal to the string "10"). Default is `false`.

```yaml
requests:
  - name: Get pizza
    url: "{{host}}/pizzas/1"
    method: get
    expect:
      status: 200
      values:
        size: Large
        type: Pepperoni
        quantity: 10
      strict: true # quantity must be a number 10, not a string "10".  Use false if not important.
```

  * `set`: a list of env variables to set from the response. Each item should have a `var` (the variable to be set) and `from` (a field in the response). This will be helpful for capturing the ID of a created resource to use in a later request.

```yaml
requests:
  - name: Order fast food
    url: "{{host}}/orders"
    method: post
    body:
      type: hamburder
      quantity: 1000
    set:
      - var: created_order # can now use urls like example.com/api/orders/{{created_order}}
        from: order_id
```

[See the full example](#complete-example) for more on how test specs can be defined using these properties.


### Logging in

Your first request can be to a token endpoint:

```sh
apitest -f todos.yaml -e auth_id=$AUTH_ID -e auth_secret=$AUTH_SECRET
```

```yaml
environment:
  vars:
    host: https://example.com
    auth_url: https://example.com/oauth/token
    auth_audience: https://example.com
  headers:
    Content-Type: application/json
    Authorization: Bearer {{auth_token}}
requests:
  - name: Log in
    url: "{{auth_url}}"
    method: post
    body:
      client_id: "{{auth_id}}"
      client_secret: "{{auth_secret}}"
      audience: "{{auth_audience}}"
      grant_type: client_credentials
    expect:
      status: 200
    set:
      - var: auth_token # set the {{auth_token}} here
        from: access_token
  - name: Create todo as authenticated user
    url: "{{host}}/api/v1/todos"
    method: post
    body:
      title: "Pick up groceries"
    expect:
      status: 201
```

### jq style JSON parsing

Response body checking (the `expect` block) now supports jq style selectors:

* `.foo` value at key
* `.foo.bar` value at a nested key
* `.foo.[0]` value at specified index of array

For convenience, the leading `.` can be omitted.

Example:

```yaml
  - name: Get customer orders
    url: "{{host}}/api/v1/orders"
    method: get
    expect:
      values:
        customer.name: Bill
```

### Command line

Example: `apitest input.yaml`

Arguments:

* `--file` `-f`: specify a file containing test specs. Example: `-f test/test.yaml`. Note: the file may be also be the first non-flag argument e.g. `apitest --monitor --delay=60 test.yaml`
* `--env` `-e`: define variables for the test environment. Example: `-e myvar=test123`
* `--test` `-t`: specify the name of a single test to run (use quotes if the name contains spaces). Example: `-t "Todo list"`
* `--verbose` `-v`: verbose request & response logging.  Output is currently not pretty.

The following arguments apply to monitoring/metrics mode:
* `--monitor` `-m`: enable monitoring mode (with metrics)
* `--port` `-p`: port for metrics endpoint (monitoring mode only). The metrics endpoint is `/metrics`. Default: `2112`
* `--delay` `-d`: delay (seconds) between automated test runs. Default: `300`

### GitHub Actions

Add a step to your workflow like this:
```
action "Run API tests" {
  uses = "stephenhillier/apitest@master"
  args = ["test/test.yaml"]
}
```

Replace `test/test.yaml` with the path to your yaml specs.
See the `.github/main.workflow` file in this repo for a working example.


### Prometheus usage

apitest can be used with prometheus by setting the `--monitor` (or `-m`) flag.  `monitor` will
set apitest to run in continuous mode, keeping track of request and error counts and request duration for every test.

By default, the metrics are available at `localhost:2112/metrics`. The port is configurable with `--listenPort`.

The following metrics are available with `hostname`, `path`, `method` and `name` (the test name in the yaml spec) labels:

```
apitest_requests_duration
apitest_requests_duration_sum
apitest_requests_duration_count
apitest_requests_errors_total
apitest_requests_total
```

**Note**: the errors recorded denote assertion errors & tests that fail to run.  A request
returning status 500 would be considered successful if the test spec had expect `status: 500`.

## Developing
`go get github.com/stephenhillier/apitest`

`go test`

## Credits

* https://github.com/savaki/jq - jq syntax
* gopkg.in/yaml.v2 - yaml parsing
* github.com/spf13/pflag
