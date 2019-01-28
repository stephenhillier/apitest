# apitest
Test the behavior of an HTTP-based REST API [from the command line](#command-line) or in a container based pipeline like [GitHub Actions](#github-actions).

## Usage

### YAML test specs

Define requests in YAML.  [See the test spec properties](#test-spec-properties) for more details on individual fields.

#### Complete example

`apitest -f test.yaml -e token=secret123`

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


### Logging in / Retrieving tokens

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


### Limitations

Response body assertions:  only flat JSON schemas are supported

### Command line

`apitest -f input.yaml`

Arguments:

* `--file` `-f`: specify a file containing test specs. Example: `-f test/test.yaml`
* `--env` `-e`: define variables for the test environment. Example: `-e myvar=test123`
* `--test` `-t`: specify the name of a single test to run (use quotes if the name contains spaces). Example: `-t "Todo list"`
* `--verbose` `-v`: verbose request & response logging.  Output is currently not pretty.

### GitHub Actions

Add a step to your workflow like this:
```
action "Run API tests" {
  uses = "stephenhillier/apitest@master"
  args = ["-f", "test/test.yaml"]
}
```

Replace `test/test.yaml` with the path to your yaml specs.
See the `.github/main.workflow` file in this repo for a working example.

## Developing
`go get github.com/stephenhillier/apitest`

`go test`
