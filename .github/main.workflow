workflow "New workflow" {
  on = "push"
  resolves = ["stephenhillier/apitest"]
}

action "stephenhillier/apitest" {
  uses = "stephenhillier/apitest@master"
  args = ["-f", "test/test.yaml"]
}
