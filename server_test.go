package main

import (
	"testing"

	"gopkg.in/dedis/cothority.v0/dbg"
)

func TestMain(m *testing.M) {
	dbg.MainTest(m)
}

func TestIsPublicIP(t *testing.T) {
	tests := []struct {
		ip     string
		public bool
	}{
		{"11.0.0.0", true},
		{"10.0.0.0", false},
		{"10.1.2.3", false},
		{"127.0.1.2", false},
		{"8.8.8.8", true},
	}
	for _, te := range tests {
		if isPublicIP(te.ip) != te.public {
			t.Fatal(te.ip, "should be", te.public)
		}
	}
}
