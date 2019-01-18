# apitest
A lightweight API testing tool

## Usage

### Request definitions

Define requests in YAML.

`name`: a name for your request  
`url`: the URL to make a request to  
`method`: HTTP method e.g. GET, POST

`expect`: add simple checks to an expect block:  

 * `status`: HTTP status code  
 * `values`: key/value pairs

```yaml
requests:
  - name: Todo list
    url: http://localhost:8000/api/v1/todos
    method: get
    expect:
      status: 200
  - name: Create a todo item
    url: http://localhost:8000/api/v1/todos
    method: post
    expect:
      status: 201
      values:
        - key: id
          value: "1231"
        - key: todo_title
          value: Clean the house

```


### Command line
Coming soon!

### In a container-based CI/CD pipeline (GitHub Actions)
Coming soon!



## Developing
`go get github.com/stephenhillier/apitest`

`go test`
