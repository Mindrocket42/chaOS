package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const receiptSchemaVersion = 1

var hostInstanceID = newHostInstanceID()

type processOwner struct {
	Caller string
	Token  string
}

var processOwners = struct {
	sync.RWMutex
	byID map[int64]processOwner
}{byID: map[int64]processOwner{}}

type CallerContext struct {
	Surface    string `json:"surface"`
	TabID      int    `json:"tab_id,omitempty"`
	FrameID    int    `json:"frame_id,omitempty"`
	DocumentID string `json:"document_id,omitempty"`
	Host       string `json:"host,omitempty"`
}

type Receipt struct {
	Schema                int            `json:"schema"`
	HostInstance          string         `json:"host_instance"`
	RequestID             string         `json:"request_id"`
	Action                string         `json:"action"`
	Caller                *CallerContext `json:"caller,omitempty"`
	StartedAt             string         `json:"started_at"`
	FinishedAt            string         `json:"finished_at"`
	DurationMS            int64          `json:"duration_ms"`
	CapabilityFingerprint string         `json:"capability_fingerprint,omitempty"`
	Evidence              map[string]any `json:"evidence,omitempty"`
}

func newHostInstanceID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err == nil {
		return hex.EncodeToString(buf)
	}
	return fmt.Sprintf("fallback-%d-%d", os.Getpid(), time.Now().UnixNano())
}

func newProcessToken() string {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return hex.EncodeToString(buf)
}

func isMutation(action string) bool {
	switch action {
	case "fs.write", "fs.replace", "proc.run", "proc.write", "proc.stop":
		return true
	default:
		return false
	}
}

func callerIsAttributed(caller *CallerContext) bool {
	if caller == nil || caller.Surface != "chatgpt-tab" || caller.TabID < 0 || caller.FrameID < 0 || strings.TrimSpace(caller.DocumentID) == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(caller.Host)) {
	case "chatgpt.com", "chat.openai.com":
		return true
	default:
		return false
	}
}

func callerSignature(caller *CallerContext) string {
	if caller == nil {
		return "none"
	}
	return fmt.Sprintf("%s|%d|%d|%s|%s", caller.Surface, caller.TabID, caller.FrameID, caller.DocumentID, strings.ToLower(caller.Host))
}

func processIDFromArgs(raw json.RawMessage) (int64, map[string]any, error) {
	args, err := objectArgs(raw)
	if err != nil {
		return 0, nil, err
	}
	id, err := intArg(args, "process_id", 0, 1, 1<<30)
	if err != nil {
		return 0, nil, err
	}
	return int64(id), args, nil
}

func authorizeProcessCaller(action string, raw json.RawMessage, caller *CallerContext) error {
	if action != "proc.read" && action != "proc.write" && action != "proc.stop" {
		return nil
	}
	if !callerIsAttributed(caller) {
		return failAction("CALLER_CONTEXT_REQUIRED", "process handles require transport-generated attribution to a ChatGPT browser document")
	}
	id, args, err := processIDFromArgs(raw)
	if err != nil {
		return err
	}
	processOwners.RLock()
	owner, ok := processOwners.byID[id]
	processOwners.RUnlock()
	if !ok {
		return failAction("PROCESS_OWNER_UNKNOWN", "process %d has no chaOS caller owner in this host lifetime", id)
	}
	if owner.Caller == callerSignature(caller) {
		return nil
	}
	provided, _ := args["process_token"].(string)
	if provided == "" || owner.Token == "" {
		return failAction("PROCESS_CAPABILITY_REQUIRED", "process %d belongs to another browser document; provide the process_token returned by proc.run", id)
	}
	if len(provided) != len(owner.Token) || subtle.ConstantTimeCompare([]byte(provided), []byte(owner.Token)) != 1 {
		return failAction("PROCESS_CAPABILITY_MISMATCH", "process %d capability did not match", id)
	}
	return nil
}

func rememberProcessCaller(action string, result any, caller *CallerContext) {
	if action != "proc.run" || !callerIsAttributed(caller) {
		return
	}
	object, ok := result.(map[string]any)
	if !ok {
		return
	}
	var id int64
	switch value := object["process_id"].(type) {
	case int64:
		id = value
	case int:
		id = int64(value)
	case float64:
		id = int64(value)
	}
	if id <= 0 {
		return
	}
	token := newProcessToken()
	processOwners.Lock()
	processOwners.byID[id] = processOwner{Caller: callerSignature(caller), Token: token}
	processOwners.Unlock()
	if token != "" {
		object["process_token"] = token
	}
}

func capabilityFingerprint(cfg Config) string {
	type fingerprintRoot struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	type fingerprintPayload struct {
		ConfigVersion int               `json:"config_version"`
		HostVersion   string            `json:"host_version"`
		Roots         []fingerprintRoot `json:"roots"`
		Capabilities  Capabilities      `json:"capabilities"`
	}
	roots := make([]fingerprintRoot, 0, len(cfg.Roots))
	for _, root := range cfg.Roots {
		roots = append(roots, fingerprintRoot{Name: root.Name, Path: root.Path})
	}
	sort.Slice(roots, func(i, j int) bool {
		if roots[i].Name == roots[j].Name {
			return roots[i].Path < roots[j].Path
		}
		return roots[i].Name < roots[j].Name
	})
	payload, _ := json.Marshal(fingerprintPayload{
		ConfigVersion: cfg.Version,
		HostVersion:   hostVersion,
		Roots:         roots,
		Capabilities:  cfg.Capabilities,
	})
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func attachHealthMetadata(result any, fingerprint string) any {
	if object, ok := result.(map[string]any); ok {
		object["capability_fingerprint"] = fingerprint
		object["host_instance"] = hostInstanceID
		object["receipt_schema"] = receiptSchemaVersion
	}
	return result
}

func mutationEvidence(action string, raw json.RawMessage) map[string]any {
	if action != "fs.write" && action != "fs.replace" {
		return nil
	}
	cfg, err := loadConfig()
	if err != nil {
		return map[string]any{"verification_error": truncate(err.Error(), 500)}
	}
	args, err := objectArgs(raw)
	if err != nil {
		return map[string]any{"verification_error": truncate(err.Error(), 500)}
	}
	virtual, err := stringArg(args, "path", true, 4096)
	if err != nil {
		return map[string]any{"verification_error": truncate(err.Error(), 500)}
	}
	real, canonical, err := resolveVirtual(cfg, virtual, false)
	if err != nil {
		return map[string]any{"verification_error": truncate(err.Error(), 500)}
	}
	data, err := os.ReadFile(real)
	if err != nil {
		return map[string]any{"path": canonical, "verification_error": truncate(err.Error(), 500)}
	}
	sum := sha256.Sum256(data)
	return map[string]any{
		"path":   canonical,
		"bytes":  len(data),
		"sha256": hex.EncodeToString(sum[:]),
	}
}

func buildReceipt(req request, caller *CallerContext, started time.Time, fingerprint string, evidence map[string]any) *Receipt {
	finished := time.Now().UTC()
	duration := finished.Sub(started).Milliseconds()
	if duration < 0 {
		duration = 0
	}
	return &Receipt{
		Schema:                receiptSchemaVersion,
		HostInstance:          hostInstanceID,
		RequestID:             req.ID,
		Action:                req.Action,
		Caller:                caller,
		StartedAt:             started.UTC().Format(time.RFC3339Nano),
		FinishedAt:            finished.Format(time.RFC3339Nano),
		DurationMS:            duration,
		CapabilityFingerprint: fingerprint,
		Evidence:              evidence,
	}
}
