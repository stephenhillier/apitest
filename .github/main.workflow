workflow "Sample API tests" {
  on = "push"
  resolves = ["Run sample tests"]
}

action "Run sample tests" {
  uses = "stephenhillier/apitest@master"
  args = ["-f", "test/test.yaml"]
}
