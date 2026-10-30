package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSuccessfulRequestIDIsBoundToCallerContext(t *testing.T) {
	base := withTempConfig(t)
	approved := filepath.Join(base, "approved")
	if err := os.MkdirAll(approved, 0o755); err != nil { t.Fatal(err) }
	if err := addRoot("project", approved); err != nil { t.Fatal(err) }
	if err := setCapability("write", true); err != nil { t.Fatal(err) }

	firstCaller := attributedCaller()
	first := handleRequest(callBody(t, "caller-bound-id", "fs.write", map[string]any{
		"path": "/project/test.txt", "content": "alpha",
	}, firstCaller))
	if !first.OK { t.Fatalf("first write failed: %#v", first) }

	secondCaller := &CallerContext{Surface: "chatgpt-tab", TabID: 8, FrameID: 0, Host: "chatgpt.com"}
	second := handleRequest(callBody(t, "caller-bound-id", "fs.write", map[string]any{
		"path": "/project/test.txt", "content": "alpha",
	}, secondCaller))
	if second.OK || second.Error == nil || second.Error.Code != "REQUEST_ID_REUSED" {
		t.Fatalf("request id crossed caller boundary: %#v", second)
	}
}

func TestCapabilityFingerprintStableForEquivalentConfig(t *testing.T) {
	base := withTempConfig(t)
	approved := filepath.Join(base, "approved")
	if err := os.MkdirAll(approved, 0o755); err != nil { t.Fatal(err) }

	one := Config{
		Version: configVersion,
		Roots: []Root{{Name: "zeta", Path: approved}, {Name: "alpha", Path: approved}},
		Capabilities: Capabilities{Read: true, Search: true},
	}
	two := Config{
		Version: configVersion,
		Roots: []Root{{Name: "alpha", Path: approved}, {Name: "zeta", Path: approved}},
		Capabilities: Capabilities{Read: true, Search: true},
	}
	if capabilityFingerprint(one) != capabilityFingerprint(two) {
		t.Fatal("equivalent authority surfaces produced different fingerprints")
	}
}
