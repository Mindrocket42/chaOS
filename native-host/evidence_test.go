package main

import (
	"encoding/json"
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

	secondCaller := &CallerContext{Surface: "chatgpt-tab", TabID: 8, FrameID: 0, DocumentID: "document-8", Host: "chatgpt.com"}
	second := handleRequest(callBody(t, "caller-bound-id", "fs.write", map[string]any{
		"path": "/project/test.txt", "content": "alpha",
	}, secondCaller))
	if second.OK || second.Error == nil || second.Error.Code != "REQUEST_ID_REUSED" {
		t.Fatalf("request id crossed caller boundary: %#v", second)
	}
	if second.Receipt == nil || second.Receipt.Caller == nil || second.Receipt.Caller.DocumentID != "document-8" {
		t.Fatalf("rejected replay omitted current caller receipt: %#v", second.Receipt)
	}
}

func TestMutationWithoutDocumentIdentityFailsClosed(t *testing.T) {
	base := withTempConfig(t)
	approved := filepath.Join(base, "approved")
	if err := os.MkdirAll(approved, 0o755); err != nil { t.Fatal(err) }
	if err := addRoot("project", approved); err != nil { t.Fatal(err) }
	if err := setCapability("write", true); err != nil { t.Fatal(err) }

	caller := &CallerContext{Surface: "chatgpt-tab", TabID: 7, FrameID: 0, Host: "chatgpt.com"}
	got := handleRequest(callBody(t, "missing-document-id", "fs.write", map[string]any{
		"path": "/project/test.txt", "content": "blocked",
	}, caller))
	if got.OK || got.Error == nil || got.Error.Code != "CALLER_CONTEXT_REQUIRED" {
		t.Fatalf("mutation without document identity was not refused: %#v", got)
	}
	if _, err := os.Stat(filepath.Join(approved, "test.txt")); !os.IsNotExist(err) {
		t.Fatalf("refused mutation touched disk: %v", err)
	}
}

func TestRetainedProcessHandleUsesDocumentOwnershipAndRecoveryCapability(t *testing.T) {
	firstCaller := attributedCaller()
	secondCaller := &CallerContext{Surface: "chatgpt-tab", TabID: 8, FrameID: 0, DocumentID: "document-8", Host: "chatgpt.com"}
	const processID int64 = 424242

	processOwners.Lock()
	delete(processOwners.byID, processID)
	processOwners.Unlock()
	result := map[string]any{"process_id": processID}
	rememberProcessCaller("proc.run", result, firstCaller)
	token, _ := result["process_token"].(string)
	if token == "" {
		t.Fatal("proc.run ownership did not mint a recovery capability")
	}

	ownerArgs, _ := json.Marshal(map[string]any{"process_id": processID})
	if err := authorizeProcessCaller("proc.read", ownerArgs, firstCaller); err != nil {
		t.Fatalf("starting document could not read its process: %v", err)
	}

	otherArgs, _ := json.Marshal(map[string]any{"process_id": processID})
	err := authorizeProcessCaller("proc.read", otherArgs, secondCaller)
	actionErr, ok := err.(*actionError)
	if !ok || actionErr.Code != "PROCESS_CAPABILITY_REQUIRED" {
		t.Fatalf("other document was not challenged for capability: %#v", err)
	}

	wrongArgs, _ := json.Marshal(map[string]any{"process_id": processID, "process_token": "wrong"})
	err = authorizeProcessCaller("proc.read", wrongArgs, secondCaller)
	actionErr, ok = err.(*actionError)
	if !ok || actionErr.Code != "PROCESS_CAPABILITY_MISMATCH" {
		t.Fatalf("wrong process capability was not rejected: %#v", err)
	}

	recoveryArgs, _ := json.Marshal(map[string]any{"process_id": processID, "process_token": token})
	if err := authorizeProcessCaller("proc.read", recoveryArgs, secondCaller); err != nil {
		t.Fatalf("valid recovery capability was rejected: %v", err)
	}
}

func TestUnknownProcessOwnerFailsClosed(t *testing.T) {
	const processID int64 = 424243
	processOwners.Lock()
	delete(processOwners.byID, processID)
	processOwners.Unlock()

	raw, _ := json.Marshal(map[string]any{"process_id": processID})
	err := authorizeProcessCaller("proc.read", raw, attributedCaller())
	actionErr, ok := err.(*actionError)
	if !ok || actionErr.Code != "PROCESS_OWNER_UNKNOWN" {
		t.Fatalf("unknown process owner was not rejected: %#v", err)
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
