# apitest
A lightweight API testing tool

## Usage

### YAML test specs

Define requests in YAML.

`environment`: define variables that can be accessed through template tags e.g. `host: example.com` will be available as `{{host}}` in request URLs.  Currently only the URL field will accept environment variables, and entries containing `{{ }}` may need to be surrounded by quotes to make sure they are parsed as a string.

```yaml
environment:
  host: https://www.example.com/api/v1
```

`requests`: a list of requests to make as part of the test run.  Each request in the list can have the following properties:

  * `name`: a name for your request
  * `url`: the URL to make a request to
  * `method`: HTTP method e.g. GET, POST
  * `body`: key/value pairs that will be sent in the request body as JSON

```yaml
requests:
  - name: Add a pizza
    url: "{{host}}/pizzas"
    method: post
    body:
      size: Large
      type: Pepperoni
```

  * `expect`: add simple checks to an expect block:  
    * `status`: HTTP status code  
    * `values`: key/value pairs 

```yaml
requests:
  - name: My test request
    url: "{{host}}/jokes/1"
    method: get
    expect:
      status: 200
      values:
        joke: How did the Vikings send secret messages?
        punchline: By norse code!
```

  * `set`: a list of items with `var` (the variable to be set) and `from` (a field in the response)


#### Complete example

```yaml
environment:
  host: http://localhost:8000
requests:
  - name: Add a comment
    url: "{{host}}/comments"
    method: post
    expect:
      status: 201
    body:
      content: This is my comment!
    set:
      - var: created_comment
        from: id
  - name: Get single comment
    url: "{{host}}/comments/{{created_comment}}"
    method: get
    expect:
      status: 200
      values:
        id: 1
        content: This is my comment!
```


### Command line

`apitest -f input.yaml`

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
