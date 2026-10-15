package config

import "testing"

func TestPreservePathValidation(t *testing.T) {
	if err := validateRewriteRule(Route{Path: "/ws", PreservePath: true, RewriteFrom: "/a", RewriteTo: "/b"}); err == nil {
		t.Error("expected conflict error")
	}
	if err := validateRewriteRule(Route{Path: "/ws", PreservePath: true, Backends: []string{"http://h:1/x"}}); err == nil {
		t.Error("expected backend path error")
	}
	if err := validateRewriteRule(Route{Path: "/ws", PreservePath: true, Backends: []string{"http://h:1", "http://h:2/"}}); err != nil {
		t.Errorf("unexpected: %v", err)
	}
}
