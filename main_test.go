package main

import "testing"

// test processing env variable cli input
func TestProcessEnvVars(t *testing.T) {
	env := Environment{}
	env.Vars = make(map[string]interface{})

	vars := []string{
		"test1=foo",
		"test2=bar",
	}

	env.processEnvVars(vars)

	if env.Vars["test1"] != "foo" {
		t.Errorf("expected %s, received %s", "foo", env.Vars["test1"])
	}
	if env.Vars["test2"] != "bar" {
		t.Errorf("expected %s, received %s", "bar", env.Vars["test2"])
	}
}
