vars = {
  host = "http://jsonplaceholder.typicode.com"
  token = "secret123"
}

headers = {
  Authorization = "Bearer {{token}}"
}

request {
  name = "Create a todo item"
  url = "{{host}}/todos"
  method = "post"
  expect = {
    status = 201
  }
  set = {
    created_todo = "id"
  }
}

request {
  name = "Get todo"
  url = "{{host}}/todos/1"
  method = "get"
  expect = {
    status = 200
    values = {
      id = 1
      title = "delectus aut autem"
    }
  }
}
