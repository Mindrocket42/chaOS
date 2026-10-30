package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

const receiptSchemaVersion = 1

type CallerContext struct {
	Surface string `json:"surface"`
	TabID   int    `json:"tab_id,omitempty"`
	FrameID int    `json:"frame_id,omitempty"`
	Host    string `json:"host,omitempty"`
}

type Receipt struct {
	Schema                int               `json:"schema"`
	RequestID             string            `json:"request_id"`
	Action                string            `json:"action"`
	Caller                *CallerContext    `json:"caller,omitempty"`
	StartedAt             string            `json:"started_at"`
	FinishedAt            string            `json:"finished_at"`
	DurationMS            int64             `json:"duration_ms"`
	CapabilityFingerprint string            `json:"capability_fingerprint,omitempty"`
	Evidence              map[string]any    `json:"evidence,omitempty"`
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
	if caller == nil || caller.Surface != "chatgpt-tab" || caller.TabID < 0 || caller.FrameID < 0 {
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
	return fmt.Sprintf("%s|%d|%d|%s", caller.Surface, caller.TabID, caller.FrameID, strings.ToLower(caller.Host))
}

func capabilityFingerprint(cfg Config) string {
	type fingerprintRoot struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	type fingerprintPayload struct {
		ConfigVersion int          `json:"config_version"`
		HostVersion   string       `json:"host_version"`
		Roots         []fingerprintRoot `json:"roots"`
		Capabilities  Capabilities `json:"capabilities"`
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

func attachHealthFingerprint(result any, fingerprint string) any {
	if object, ok := result.(map[string]any); ok {
		object["capability_fingerprint"] = fingerprint
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
