package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func withTempConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)
	t.Setenv("LOCALAPPDATA", dir)
	return dir
}

func frameRequest(t *testing.T, value any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil { t.Fatal(err) }
	var out bytes.Buffer
	if err := binary.Write(&out, binary.LittleEndian, uint32(len(body))); err != nil { t.Fatal(err) }
	out.Write(body)
	return out.Bytes()
}

func decodeResponse(t *testing.T, raw []byte) response {
	t.Helper()
	reader := bytes.NewReader(raw)
	var size uint32
	if err := binary.Read(reader, binary.LittleEndian, &size); err != nil { t.Fatal(err) }
	body := make([]byte, size)
	if _, err := io.ReadFull(reader, body); err != nil { t.Fatal(err) }
	if reader.Len() != 0 { t.Fatalf("unexpected trailing bytes: %d", reader.Len()) }
	var got response
	if err := json.Unmarshal(body, &got); err != nil { t.Fatal(err) }
	return got
}

func callBody(t *testing.T, id, action string, args map[string]any, caller *CallerContext) []byte {
	t.Helper()
	value := map[string]any{"type": "call", "id": id, "action": action, "args": args}
	if caller != nil { value["caller"] = caller }
	body, err := json.Marshal(value)
	if err != nil { t.Fatal(err) }
	return body
}

func attributedCaller() *CallerContext {
	return &CallerContext{Surface: "chatgpt-tab", TabID: 7, FrameID: 0, Host: "chatgpt.com"}
}

func TestNativeMessagingPingUsesRawFrame(t *testing.T) {
	withTempConfig(t)
	input := frameRequest(t, map[string]any{"type": "ping", "id": "wire-test"})
	var output bytes.Buffer
	err := serveNativeMessaging(bytes.NewReader(input), &output)
	if err != io.EOF { t.Fatalf("expected EOF after one frame, got %v", err) }
	got := decodeResponse(t, output.Bytes())
	if !got.OK || got.ID != "wire-test" || got.Protocol != protocolVersion { t.Fatalf("unexpected response: %#v", got) }
}

func TestFreshConfigStartsClosed(t *testing.T) {
	withTempConfig(t)
	cfg, err := loadConfig()
	if err != nil { t.Fatal(err) }
	if len(cfg.Roots) != 0 { t.Fatalf("fresh config exposed roots: %#v", cfg.Roots) }
	if cfg.Capabilities.Read || cfg.Capabilities.Search || cfg.Capabilities.Write || cfg.Capabilities.Command {
		t.Fatalf("fresh config widened capabilities: %#v", cfg.Capabilities)
	}
}

func TestVirtualRootsDoNotExposeHostPath(t *testing.T) {
	base := withTempConfig(t)
	approved := filepath.Join(base, "approved")
	if err := os.MkdirAll(approved, 0o755); err != nil { t.Fatal(err) }
	if err := addRoot("project", approved); err != nil { t.Fatal(err) }
	if err := setCapability("read", true); err != nil { t.Fatal(err) }
	result, err := runAction("health", nil)
	if err != nil { t.Fatal(err) }
	encoded, _ := json.Marshal(result)
	if bytes.Contains(encoded, []byte(approved)) { t.Fatalf("health leaked host path: %s", encoded) }
	if !bytes.Contains(encoded, []byte(`"/project"`)) { t.Fatalf("health omitted virtual root: %s", encoded) }
}

func TestPathTraversalCannotEscapeRoot(t *testing.T) {
	base := withTempConfig(t)
	approved := filepath.Join(base, "approved")
	if err := os.MkdirAll(approved, 0o755); err != nil { t.Fatal(err) }
	cfg := defaultConfig()
	cfg.Roots = []Root{{Name: "project", Path: approved}}
	if _, _, err := resolveVirtual(cfg, "/project/../../outside.txt", true); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
}

func TestCapabilityFingerprintChangesWithAuthoritySurface(t *testing.T) {
	withTempConfig(t)
	first := handleRequest(callBody(t, "health-a", "health", map[string]any{}, nil))
	if !first.OK { t.Fatalf("health failed: %#v", first) }
	firstResult, ok := first.Result.(map[string]any)
	if !ok { t.Fatalf("unexpected health result: %#v", first.Result) }
	firstFingerprint, _ := firstResult["capability_fingerprint"].(string)
	if firstFingerprint == "" { t.Fatal("health omitted capability fingerprint") }

	if err := setCapability("read", true); err != nil { t.Fatal(err) }
	second := handleRequest(callBody(t, "health-b", "health", map[string]any{}, nil))
	secondResult, ok := second.Result.(map[string]any)
	if !ok { t.Fatalf("unexpected health result: %#v", second.Result) }
	secondFingerprint, _ := secondResult["capability_fingerprint"].(string)
	if secondFingerprint == "" || secondFingerprint == firstFingerprint {
		t.Fatalf("fingerprint did not change with authority surface: %q -> %q", firstFingerprint, secondFingerprint)
	}
}

func TestMutationWithoutAttributedCallerFailsClosed(t *testing.T) {
	base := withTempConfig(t)
	approved := filepath.Join(base, "approved")
	if err := os.MkdirAll(approved, 0o755); err != nil { t.Fatal(err) }
	if err := addRoot("project", approved); err != nil { t.Fatal(err) }
	if err := setCapability("write", true); err != nil { t.Fatal(err) }

	got := handleRequest(callBody(t, "write-no-caller", "fs.write", map[string]any{"path": "/project/test.txt", "content": "blocked"}, nil))
	if got.OK || got.Error == nil || got.Error.Code != "CALLER_CONTEXT_REQUIRED" {
		t.Fatalf("unattributed mutation was not refused: %#v", got)
	}
	if got.Receipt == nil || got.Receipt.Action != "fs.write" { t.Fatalf("refusal omitted receipt: %#v", got) }
	if _, err := os.Stat(filepath.Join(approved, "test.txt")); !os.IsNotExist(err) {
		t.Fatalf("refused mutation touched disk: %v", err)
	}
}

func TestWriteReceiptCarriesCallerAndContentDigest(t *testing.T) {
	base := withTempConfig(t)
	approved := filepath.Join(base, "approved")
	if err := os.MkdirAll(approved, 0o755); err != nil { t.Fatal(err) }
	if err := addRoot("project", approved); err != nil { t.Fatal(err) }
	if err := setCapability("write", true); err != nil { t.Fatal(err) }
	caller := attributedCaller()

	got := handleRequest(callBody(t, "write-receipt", "fs.write", map[string]any{"path": "/project/test.txt", "content": "alpha"}, caller))
	if !got.OK { t.Fatalf("write failed: %#v", got) }
	if got.Receipt == nil { t.Fatal("write omitted receipt") }
	if got.Receipt.Schema != receiptSchemaVersion || got.Receipt.RequestID != "write-receipt" || got.Receipt.Action != "fs.write" {
		t.Fatalf("bad receipt identity: %#v", got.Receipt)
	}
	if got.Receipt.Caller == nil || got.Receipt.Caller.TabID != caller.TabID || got.Receipt.Caller.Host != caller.Host {
		t.Fatalf("receipt omitted transport caller: %#v", got.Receipt.Caller)
	}
	if got.Receipt.CapabilityFingerprint == "" { t.Fatal("receipt omitted capability fingerprint") }
	sum := sha256.Sum256([]byte("alpha"))
	wantHash := hex.EncodeToString(sum[:])
	if got.Receipt.Evidence["sha256"] != wantHash || got.Receipt.Evidence["path"] != "/project/test.txt" {
		t.Fatalf("write evidence mismatch: %#v", got.Receipt.Evidence)
	}
}

func TestDuplicateSuccessfulMutationIsNotReexecuted(t *testing.T) {
	base := withTempConfig(t)
	approved := filepath.Join(base, "approved")
	if err := os.MkdirAll(approved, 0o755); err != nil { t.Fatal(err) }
	if err := addRoot("project", approved); err != nil { t.Fatal(err) }
	if err := setCapability("write", true); err != nil { t.Fatal(err) }
	body := callBody(t, "write-once", "fs.write", map[string]any{"path": "/project/test.txt", "content": "first"}, attributedCaller())

	first := handleRequest(body)
	if !first.OK { t.Fatalf("first write failed: %#v", first) }
	target := filepath.Join(approved, "test.txt")
	if err := os.WriteFile(target, []byte("external-change"), 0o644); err != nil { t.Fatal(err) }
	second := handleRequest(body)
	if !second.OK { t.Fatalf("cached write failed: %#v", second) }
	data, err := os.ReadFile(target)
	if err != nil { t.Fatal(err) }
	if string(data) != "external-change" {
		t.Fatalf("duplicate request re-executed mutation: %q", data)
	}
	if second.Receipt == nil || first.Receipt == nil || second.Receipt.FinishedAt != first.Receipt.FinishedAt {
		t.Fatalf("duplicate did not return original receipt: first=%#v second=%#v", first.Receipt, second.Receipt)
	}
}
