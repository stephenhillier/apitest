workflow "Run tests" {
  on = "push"
  resolves = ["Run sample tests"]
}

action "Lint" {
  uses = "stefanprodan/gh-actions/golang"
  args = "fmt"
}

action "Test" {
  needs = ["Lint"]
  uses = "stefanprodan/gh-actions/golang"
  args = "test"
}

action "Run sample tests" {
  needs = ["Test"]
  uses = "stephenhillier/apitest@master"
  args = ["-f", "test/test.yaml"]
}
