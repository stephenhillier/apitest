# apitest
A lightweight API testing tool

## Usage

### YAML test specs

Define requests in YAML.

`name`: a name for your request  
`url`: the URL to make a request to  
`method`: HTTP method e.g. GET, POST

`expect`: add simple checks to an expect block:  

 * `status`: HTTP status code  
 * `values`: key/value pairs

```yaml
requests:
  - name: Todo 1
    url: https://jsonplaceholder.typicode.com/todos/1
    method: get
    expect:
      status: 200
      values:
        - key: id
          value: 1
        - key: title
          value: delectus aut autem
  - name: Create a todo item
    url: https://jsonplaceholder.typicode.com/todos
    method: post
    expect:
      status: 201
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
