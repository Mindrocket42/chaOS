package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	hostVersion      = "0.1.0"
	hostName         = "pro.ironhead.chaos"
	extensionID      = "oneodmlppiiihbbligdiloibjagnfach"
	extensionOrigin  = "chrome-extension://" + extensionID + "/"
	protocolVersion  = 1
	maxRequestBytes  = 8 << 20
	maxResponseBytes = 900 << 10
	maxCachedCalls   = 512
)

type request struct {
	Type   string          `json:"type"`
	ID     string          `json:"id"`
	Action string          `json:"action,omitempty"`
	Args   json.RawMessage `json:"args,omitempty"`
	Caller *CallerContext  `json:"caller,omitempty"`
}

type response struct {
	Type     string         `json:"type"`
	ID       string         `json:"id"`
	OK       bool           `json:"ok"`
	Protocol int            `json:"protocol"`
	Result   any            `json:"result,omitempty"`
	Error    *responseError `json:"error,omitempty"`
	Receipt  *Receipt       `json:"receipt,omitempty"`
}

type responseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type cachedCall struct {
	Signature string
	Response  response
}

var (
	idPattern = regexpID()
	cacheMu   sync.Mutex
	callCache = map[string]cachedCall{}
	cacheFIFO []string
)

func regexpID() func(string) bool {
	const allowed = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789._:-"
	return func(value string) bool {
		if len(value) < 1 || len(value) > 128 { return false }
		for _, r := range value { if !strings.ContainsRune(allowed, r) { return false } }
		return true
	}
}

func main() {
	if handled, code := handleCLI(os.Args[1:]); handled {
		os.Exit(code)
	}
	if !hasExtensionOrigin(os.Args[1:]) {
		fmt.Fprintln(os.Stderr, "chaOS native host refused launch: expected the registered Chrome extension origin")
		os.Exit(2)
	}
	defer shutdownProcesses()
	if err := serveNativeMessaging(os.Stdin, os.Stdout); err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintln(os.Stderr, "chaOS native host:", err)
		os.Exit(3)
	}
}

func handleCLI(args []string) (bool, int) {
	if len(args) == 0 { return false, 0 }
	switch args[0] {
	case "--register":
		if err := registerHost(); err != nil { fmt.Fprintln(os.Stderr, "register:", err); return true, 1 }
		fmt.Println("chaOS Chrome native host registered.")
		return true, 0
	case "--unregister":
		if err := unregisterHost(); err != nil { fmt.Fprintln(os.Stderr, "unregister:", err); return true, 1 }
		fmt.Println("chaOS Chrome native host unregistered.")
		return true, 0
	case "--status":
		cfg, err := loadConfig(); if err != nil { fmt.Fprintln(os.Stderr, "status:", err); return true, 1 }
		path, _ := configPath(); data, _ := json.MarshalIndent(cfg, "", "  "); fmt.Printf("Config: %s\n%s\n", path, data)
		return true, 0
	case "--add-root":
		if len(args) != 3 { fmt.Fprintln(os.Stderr, "usage: chaos-host.exe --add-root <name> <folder>"); return true, 2 }
		if err := addRoot(args[1], args[2]); err != nil { fmt.Fprintln(os.Stderr, "add-root:", err); return true, 1 }
		fmt.Printf("Approved root /%s -> %s\n", strings.ToLower(args[1]), args[2]); return true, 0
	case "--remove-root":
		if len(args) != 2 { fmt.Fprintln(os.Stderr, "usage: chaos-host.exe --remove-root <name>"); return true, 2 }
		if err := removeRoot(args[1]); err != nil { fmt.Fprintln(os.Stderr, "remove-root:", err); return true, 1 }
		fmt.Printf("Removed approved root /%s\n", strings.ToLower(args[1])); return true, 0
	case "--set-capability":
		if len(args) != 3 { fmt.Fprintln(os.Stderr, "usage: chaos-host.exe --set-capability <read|search|write|command> <on|off>"); return true, 2 }
		enabled, err := parseOnOff(args[2]); if err != nil { fmt.Fprintln(os.Stderr, err); return true, 2 }
		if err := setCapability(args[1], enabled); err != nil { fmt.Fprintln(os.Stderr, "set-capability:", err); return true, 1 }
		fmt.Printf("Capability %s = %t\n", strings.ToLower(args[1]), enabled); return true, 0
	case "--help", "-h", "/?":
		fmt.Print(`chaOS native host

Browser transport:
  Chrome launches this executable automatically through Native Messaging.

Explicit local controls:
  --register
  --unregister
  --status
  --add-root <name> <folder>
  --remove-root <name>
  --set-capability <read|search|write|command> <on|off>

Command permission is intentionally powerful: commands run with the logged-in OS user's normal access and are not confined to approved roots.
`)
		return true, 0
	}
	return false, 0
}

func parseOnOff(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "true", "1", "yes": return true, nil
	case "off", "false", "0", "no": return false, nil
	default: return false, fmt.Errorf("expected on or off, got %q", value)
	}
}

func hasExtensionOrigin(args []string) bool {
	for _, arg := range args { if arg == extensionOrigin { return true } }
	return false
}

func serveNativeMessaging(reader io.Reader, writer io.Writer) error {
	in := bufio.NewReader(reader)
	for {
		var length uint32
		if err := binary.Read(in, binary.LittleEndian, &length); err != nil { return err }
		if length < 2 || length > maxRequestBytes { return fmt.Errorf("invalid frame length %d", length) }
		body := make([]byte, length)
		if _, err := io.ReadFull(in, body); err != nil { return err }
		resp := handleRequest(body)
		encoded, err := json.Marshal(resp); if err != nil { return err }
		if len(encoded) > maxResponseBytes {
			encoded, _ = json.Marshal(errorResponse(resp.ID, "RESULT_TOO_LARGE", "native result exceeded the browser messaging limit"))
		}
		if err := binary.Write(writer, binary.LittleEndian, uint32(len(encoded))); err != nil { return err }
		if _, err := writer.Write(encoded); err != nil { return err }
	}
}

func handleRequest(body []byte) response {
	var req request
	if err := json.Unmarshal(body, &req); err != nil { return errorResponse("invalid", "BAD_JSON", "request was not valid JSON") }
	if !idPattern(req.ID) { return errorResponse("invalid", "BAD_REQUEST", "request id is missing or invalid") }
	if req.Type == "ping" {
		return successResponse(req.ID, map[string]any{"app": "chaOS", "transport": "chrome-native-messaging", "protocol": protocolVersion, "version": hostVersion})
	}
	if req.Type != "call" || req.Action == "" || len(req.Action) > 128 { return errorResponse(req.ID, "BAD_REQUEST", "expected a call request with an action") }

	started := time.Now().UTC()
	fingerprint := ""
	if cfg, err := loadConfig(); err == nil {
		fingerprint = capabilityFingerprint(cfg)
	}
	if isMutation(req.Action) && !callerIsAttributed(req.Caller) {
		resp := errorResponse(req.ID, "CALLER_CONTEXT_REQUIRED", "mutating actions require transport-generated attribution to a ChatGPT tab")
		resp.Receipt = buildReceipt(req, req.Caller, started, fingerprint, nil)
		return resp
	}

	signature := req.Action + "\x00" + string(req.Args) + "\x00" + callerSignature(req.Caller)
	cacheMu.Lock(); previous, exists := callCache[req.ID]; cacheMu.Unlock()
	if exists {
		if previous.Signature != signature {
			resp := errorResponse(req.ID, "REQUEST_ID_REUSED", "this successful action id was already used for different arguments or caller context")
			resp.Receipt = buildReceipt(req, req.Caller, started, fingerprint, nil)
			return resp
		}
		return previous.Response
	}

	result, err := runAction(req.Action, req.Args)
	if err != nil {
		var actionErr *actionError
		var resp response
		if errors.As(err, &actionErr) {
			resp = errorResponse(req.ID, actionErr.Code, actionErr.Error())
		} else {
			resp = errorResponse(req.ID, "INTERNAL_ERROR", truncate(err.Error(), 2000))
		}
		resp.Receipt = buildReceipt(req, req.Caller, started, fingerprint, nil)
		return resp
	}
	if req.Action == "health" && fingerprint != "" {
		result = attachHealthMetadata(result, fingerprint)
	}
	resp := successResponse(req.ID, result)
	resp.Receipt = buildReceipt(req, req.Caller, started, fingerprint, mutationEvidence(req.Action, req.Args))
	rememberCall(req.ID, signature, resp)
	return resp
}

func rememberCall(id, signature string, resp response) {
	cacheMu.Lock(); defer cacheMu.Unlock()
	callCache[id] = cachedCall{Signature: signature, Response: resp}; cacheFIFO = append(cacheFIFO, id)
	for len(cacheFIFO) > maxCachedCalls { oldest := cacheFIFO[0]; cacheFIFO = cacheFIFO[1:]; delete(callCache, oldest) }
}

func successResponse(id string, result any) response { return response{Type: "result", ID: id, OK: true, Protocol: protocolVersion, Result: result} }
func errorResponse(id, code, message string) response { return response{Type: "result", ID: id, OK: false, Protocol: protocolVersion, Error: &responseError{Code: code, Message: message}} }

func nativeManifestPath() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" { return "", fmt.Errorf("LOCALAPPDATA is not set") }
	return filepath.Join(base, "chaOS", "native-messaging", hostName+".json"), nil
}

func registerHost() error {
	if runtime.GOOS != "windows" { return fmt.Errorf("registration currently supports Windows only") }
	executable, err := os.Executable(); if err != nil { return err }; executable, err = filepath.Abs(executable); if err != nil { return err }
	manifestPath, err := nativeManifestPath(); if err != nil { return err }
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o700); err != nil { return err }
	manifest := map[string]any{"name": hostName, "description": "chaOS local capability host for ChatGPT in Chrome", "path": executable, "type": "stdio", "allowed_origins": []string{extensionOrigin}}
	data, _ := json.MarshalIndent(manifest, "", "  "); data = append(data, '\n')
	if err := os.WriteFile(manifestPath, data, 0o600); err != nil { return err }
	key := `HKCU\Software\Google\Chrome\NativeMessagingHosts\` + hostName
	cmd := exec.Command("reg.exe", "add", key, "/ve", "/t", "REG_SZ", "/d", manifestPath, "/f")
	if out, err := cmd.CombinedOutput(); err != nil { return fmt.Errorf("reg.exe: %v: %s", err, strings.TrimSpace(string(out))) }
	return nil
}

func unregisterHost() error {
	if runtime.GOOS != "windows" { return fmt.Errorf("unregistration currently supports Windows only") }
	key := `HKCU\Software\Google\Chrome\NativeMessagingHosts\` + hostName
	_ = exec.Command("reg.exe", "delete", key, "/f").Run()
	manifestPath, err := nativeManifestPath(); if err == nil { _ = os.Remove(manifestPath) }
	return nil
}

var _ = strconv.IntSize
