package main

import (
	"bytes"
	"encoding/binary"
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
