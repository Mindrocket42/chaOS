package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	maxReadBytes       = 1 << 20
	maxWriteBytes      = 4 << 20
	maxSearchFileBytes = 2 << 20
	maxProcessBytes    = 1 << 20
	maxProcessChunks   = 2000
)

type actionError struct {
	Code string
	Err  error
}

func (e *actionError) Error() string { return e.Err.Error() }
func failAction(code, format string, args ...any) error {
	return &actionError{Code: code, Err: fmt.Errorf(format, args...)}
}

type processChunk struct {
	Seq    int64  `json:"seq"`
	Stream string `json:"stream"`
	Text   string `json:"text"`
	bytes  int
}

type processEntry struct {
	ID        int64
	Cmd       *exec.Cmd
	Stdin     io.WriteCloser
	Workdir   string
	StartedAt int64
	ExitedAt  int64
	ExitCode  *int
	Signal    string
	NextSeq   int64
	Dropped   int64
	Bytes     int
	Chunks    []processChunk
	mu        sync.Mutex
}

var processTable = struct {
	sync.Mutex
	next int64
	m    map[int64]*processEntry
}{next: 1, m: map[int64]*processEntry{}}

func objectArgs(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return map[string]any{}, nil
	}
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil || args == nil {
		return nil, failAction("BAD_ARGS", "args must be a JSON object")
	}
	return args, nil
}

func stringArg(args map[string]any, key string, required bool, max int) (string, error) {
	value, ok := args[key]
	if !ok {
		if required {
			return "", failAction("BAD_ARGS", "%s is required", key)
		}
		return "", nil
	}
	text, ok := value.(string)
	if !ok || (required && text == "") || len(text) > max {
		return "", failAction("BAD_ARGS", "%s must be a string up to %d characters", key, max)
	}
	return text, nil
}

func intArg(args map[string]any, key string, fallback, min, max int) (int, error) {
	value, ok := args[key]
	if !ok {
		return fallback, nil
	}
	n, ok := value.(float64)
	if !ok || n != float64(int(n)) || int(n) < min || int(n) > max {
		return 0, failAction("BAD_ARGS", "%s must be an integer between %d and %d", key, min, max)
	}
	return int(n), nil
}

func boolArg(args map[string]any, key string) bool {
	value, _ := args[key].(bool)
	return value
}

func requireCapability(cfg Config, name string) error {
	enabled := false
	switch name {
	case "read":
		enabled = cfg.Capabilities.Read
	case "search":
		enabled = cfg.Capabilities.Search
	case "write":
		enabled = cfg.Capabilities.Write
	case "command":
		enabled = cfg.Capabilities.Command
	}
	if !enabled {
		return failAction("CAPABILITY_DISABLED", "%s capability is disabled", name)
	}
	return nil
}

func runAction(action string, raw json.RawMessage) (any, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, failAction("CONFIG_ERROR", "%v", err)
	}
	args, err := objectArgs(raw)
	if err != nil {
		return nil, err
	}
	switch action {
	case "health":
		roots := make([]string, 0, len(cfg.Roots))
		for _, root := range cfg.Roots {
			roots = append(roots, "/"+root.Name)
		}
		return map[string]any{
			"app": "chaOS",
			"version": hostVersion,
			"transport": "chrome-native-messaging",
			"mcp_tunnel": false,
			"roots": roots,
			"capabilities": cfg.Capabilities,
			"command_scope": "logged-in OS user; command execution is not confined to approved roots",
			"actions": []string{"health", "fs.list", "fs.read", "fs.search", "fs.write", "fs.replace", "proc.run", "proc.read", "proc.write", "proc.stop"},
		}, nil
	case "fs.list":
		return listDirectory(cfg, args)
	case "fs.read":
		return readFile(cfg, args)
	case "fs.search":
		return searchFiles(cfg, args)
	case "fs.write":
		return writeFile(cfg, args)
	case "fs.replace":
		return replaceText(cfg, args)
	case "proc.run":
		return runProcess(cfg, args)
	case "proc.read":
		return readProcess(args)
	case "proc.write":
		return writeProcess(cfg, args)
	case "proc.stop":
		return stopProcess(args)
	default:
		return nil, failAction("UNKNOWN_ACTION", "unknown action %q", action)
	}
}

func listDirectory(cfg Config, args map[string]any) (any, error) {
	if err := requireCapability(cfg, "read"); err != nil { return nil, err }
	virtual, err := stringArg(args, "path", true, 4096); if err != nil { return nil, err }
	limit, err := intArg(args, "limit", 200, 1, 500); if err != nil { return nil, err }
	real, canonical, err := resolveVirtual(cfg, virtual, false); if err != nil { return nil, failAction("PATH_ERROR", "%v", err) }
	entries, err := os.ReadDir(real); if err != nil { return nil, err }
	sort.Slice(entries, func(i, j int) bool { return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name()) })
	out := make([]map[string]any, 0, min(limit, len(entries)))
	for _, entry := range entries[:min(limit, len(entries))] {
		kind := "other"
		if entry.IsDir() { kind = "directory" } else if entry.Type().IsRegular() { kind = "file" } else if entry.Type()&os.ModeSymlink != 0 { kind = "link" }
		out = append(out, map[string]any{"path": strings.TrimSuffix(canonical, "/") + "/" + entry.Name(), "type": kind})
	}
	return map[string]any{"path": canonical, "entries": out, "truncated": len(entries) > len(out)}, nil
}

func readFile(cfg Config, args map[string]any) (any, error) {
	if err := requireCapability(cfg, "read"); err != nil { return nil, err }
	virtual, err := stringArg(args, "path", true, 4096); if err != nil { return nil, err }
	offset, err := intArg(args, "offset", 0, 0, 1<<30); if err != nil { return nil, err }
	limit, err := intArg(args, "limit", 256<<10, 1, maxReadBytes); if err != nil { return nil, err }
	real, canonical, err := resolveVirtual(cfg, virtual, false); if err != nil { return nil, failAction("PATH_ERROR", "%v", err) }
	file, err := os.Open(real); if err != nil { return nil, err }; defer file.Close()
	stat, err := file.Stat(); if err != nil { return nil, err }
	if !stat.Mode().IsRegular() { return nil, failAction("NOT_FILE", "%s is not a regular file", canonical) }
	if int64(offset) >= stat.Size() { return map[string]any{"path": canonical, "size": stat.Size(), "offset": offset, "bytes": 0, "encoding": "utf8", "data": "", "eof": true}, nil }
	buf := make([]byte, min(limit, int(stat.Size()-int64(offset))))
	n, err := file.ReadAt(buf, int64(offset)); if err != nil && !errors.Is(err, io.EOF) { return nil, err }; buf = buf[:n]
	binary := bytes.IndexByte(buf[:min(len(buf), 8192)], 0) >= 0
	encoding, data := "utf8", string(buf)
	if binary { encoding, data = "base64", base64.StdEncoding.EncodeToString(buf) }
	return map[string]any{"path": canonical, "size": stat.Size(), "offset": offset, "bytes": n, "encoding": encoding, "data": data, "eof": int64(offset+n) >= stat.Size()}, nil
}

var defaultSearchSkips = map[string]bool{".git": true, "node_modules": true, "dist": true, "build": true, ".next": true, "target": true}

type searchHit struct {
	Path string `json:"path"`
	Line int    `json:"line,omitempty"`
	Text string `json:"text,omitempty"`
}

func searchFiles(cfg Config, args map[string]any) (any, error) {
	if err := requireCapability(cfg, "search"); err != nil { return nil, err }
	virtual, err := stringArg(args, "path", true, 4096); if err != nil { return nil, err }
	query, err := stringArg(args, "query", true, 2000); if err != nil { return nil, err }
	mode, err := stringArg(args, "mode", false, 20); if err != nil { return nil, err }; if mode == "" { mode = "content" }
	if mode != "content" && mode != "name" { return nil, failAction("BAD_ARGS", "mode must be content or name") }
	maxResults, err := intArg(args, "max_results", 50, 1, 200); if err != nil { return nil, err }
	caseSensitive, regexMode := boolArg(args, "case_sensitive"), boolArg(args, "regex")
	real, canonical, err := resolveVirtual(cfg, virtual, false); if err != nil { return nil, failAction("PATH_ERROR", "%v", err) }
	var matcher *regexp.Regexp
	needle := query
	if regexMode {
		expr := query; if !caseSensitive { expr = "(?i)" + expr }
		matcher, err = regexp.Compile(expr); if err != nil { return nil, failAction("BAD_REGEX", "%v", err) }
	} else if !caseSensitive { needle = strings.ToLower(query) }
	matches := func(text string) bool { if matcher != nil { return matcher.MatchString(text) }; if !caseSensitive { text = strings.ToLower(text) }; return strings.Contains(text, needle) }
	hits := []searchHit{}
	truncated := false
	walkErr := filepath.WalkDir(real, func(hostPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil { return nil }
		if hostPath != real && entry.IsDir() && defaultSearchSkips[entry.Name()] { return filepath.SkipDir }
		if entry.IsDir() || !entry.Type().IsRegular() { return nil }
		rel, relErr := filepath.Rel(real, hostPath); if relErr != nil { return nil }
		virtualPath := strings.TrimSuffix(canonical, "/") + "/" + filepath.ToSlash(rel)
		if mode == "name" {
			if matches(entry.Name()) { hits = append(hits, searchHit{Path: virtualPath}) }
		} else {
			info, statErr := entry.Info(); if statErr != nil || info.Size() > maxSearchFileBytes { return nil }
			file, openErr := os.Open(hostPath); if openErr != nil { return nil }
			scanner := bufio.NewScanner(file); scanner.Buffer(make([]byte, 64<<10), 1<<20)
			line := 0
			for scanner.Scan() {
				line++
				text := scanner.Text()
				if matches(text) { hits = append(hits, searchHit{Path: virtualPath, Line: line, Text: truncate(text, 500)}) }
				if len(hits) >= maxResults { truncated = true; break }
			}
			file.Close()
		}
		if len(hits) >= maxResults { return errors.New("chaos-search-limit") }
		return nil
	})
	if walkErr != nil && walkErr.Error() != "chaos-search-limit" { return nil, walkErr }
	return map[string]any{"path": canonical, "query": query, "mode": mode, "hits": hits, "truncated": truncated}, nil
}

func writeFile(cfg Config, args map[string]any) (any, error) {
	if err := requireCapability(cfg, "write"); err != nil { return nil, err }
	virtual, err := stringArg(args, "path", true, 4096); if err != nil { return nil, err }
	content, err := stringArg(args, "content", false, maxWriteBytes); if err != nil { return nil, err }
	if len([]byte(content)) > maxWriteBytes { return nil, failAction("TOO_LARGE", "content exceeds %d bytes", maxWriteBytes) }
	real, canonical, err := resolveVirtual(cfg, virtual, true); if err != nil { return nil, failAction("PATH_ERROR", "%v", err) }
	_, statErr := os.Stat(real); created := errors.Is(statErr, os.ErrNotExist)
	if statErr != nil && !created { return nil, statErr }
	if err := os.MkdirAll(filepath.Dir(real), 0o755); err != nil { return nil, err }
	if err := os.WriteFile(real, []byte(content), 0o644); err != nil { return nil, err }
	return map[string]any{"path": canonical, "bytes": len([]byte(content)), "created": created}, nil
}

func replaceText(cfg Config, args map[string]any) (any, error) {
	if err := requireCapability(cfg, "write"); err != nil { return nil, err }
	virtual, err := stringArg(args, "path", true, 4096); if err != nil { return nil, err }
	before, err := stringArg(args, "before", true, maxWriteBytes); if err != nil { return nil, err }
	after, err := stringArg(args, "after", false, maxWriteBytes); if err != nil { return nil, err }
	expected, err := intArg(args, "expected", 1, 1, 1000); if err != nil { return nil, err }
	real, canonical, err := resolveVirtual(cfg, virtual, false); if err != nil { return nil, failAction("PATH_ERROR", "%v", err) }
	data, err := os.ReadFile(real); if err != nil { return nil, err }
	if len(data) > maxWriteBytes { return nil, failAction("TOO_LARGE", "file exceeds %d bytes", maxWriteBytes) }
	text := string(data); count := strings.Count(text, before)
	if count != expected { return nil, failAction("REPLACE_COUNT_MISMATCH", "expected %d exact match(es), found %d", expected, count) }
	updated := strings.ReplaceAll(text, before, after)
	if len([]byte(updated)) > maxWriteBytes { return nil, failAction("TOO_LARGE", "updated file exceeds %d bytes", maxWriteBytes) }
	if err := os.WriteFile(real, []byte(updated), 0o644); err != nil { return nil, err }
	return map[string]any{"path": canonical, "replacements": count, "bytes": len([]byte(updated))}, nil
}

func shellCommand(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		if pwsh, err := exec.LookPath("pwsh.exe"); err == nil { return pwsh, []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command", command} }
		if ps, err := exec.LookPath("powershell.exe"); err == nil { return ps, []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command", command} }
		return os.Getenv("COMSPEC"), []string{"/d", "/s", "/c", command}
	}
	shell := os.Getenv("SHELL"); if shell == "" { shell = "/bin/sh" }
	return shell, []string{"-lc", command}
}

func runProcess(cfg Config, args map[string]any) (any, error) {
	if err := requireCapability(cfg, "command"); err != nil { return nil, err }
	command, err := stringArg(args, "command", true, 32000); if err != nil { return nil, err }
	workdir, err := stringArg(args, "workdir", true, 4096); if err != nil { return nil, err }
	real, canonical, err := resolveVirtual(cfg, workdir, false); if err != nil { return nil, failAction("PATH_ERROR", "%v", err) }
	stat, err := os.Stat(real); if err != nil || !stat.IsDir() { return nil, failAction("NOT_DIRECTORY", "%s is not a directory", canonical) }
	file, argv := shellCommand(command); if file == "" { return nil, failAction("NO_SHELL", "no shell is available") }
	cmd := exec.Command(file, argv...); cmd.Dir = real; cmd.Env = os.Environ()
	stdin, err := cmd.StdinPipe(); if err != nil { return nil, err }
	stdout, err := cmd.StdoutPipe(); if err != nil { return nil, err }
	stderr, err := cmd.StderrPipe(); if err != nil { return nil, err }
	if err := cmd.Start(); err != nil { return nil, err }
	processTable.Lock(); id := processTable.next; processTable.next++; entry := &processEntry{ID: id, Cmd: cmd, Stdin: stdin, Workdir: canonical, StartedAt: time.Now().UnixMilli(), NextSeq: 1}; processTable.m[id] = entry; processTable.Unlock()
	go captureProcess(entry, "stdout", stdout); go captureProcess(entry, "stderr", stderr)
	go func() { waitErr := cmd.Wait(); entry.mu.Lock(); defer entry.mu.Unlock(); entry.ExitedAt = time.Now().UnixMilli(); code := -1; if cmd.ProcessState != nil { code = cmd.ProcessState.ExitCode() }; entry.ExitCode = &code; if waitErr != nil && code == -1 { entry.Signal = waitErr.Error() } }()
	return map[string]any{"process_id": id, "pid": cmd.Process.Pid, "workdir": canonical, "running": true, "started_at": entry.StartedAt}, nil
}

func captureProcess(entry *processEntry, stream string, reader io.Reader) {
	buf := make([]byte, 16<<10)
	for { n, err := reader.Read(buf); if n > 0 { entry.appendChunk(stream, string(buf[:n])) }; if err != nil { return } }
}

func (entry *processEntry) appendChunk(stream, text string) {
	entry.mu.Lock(); defer entry.mu.Unlock()
	chunk := processChunk{Seq: entry.NextSeq, Stream: stream, Text: text, bytes: len([]byte(text))}; entry.NextSeq++; entry.Chunks = append(entry.Chunks, chunk); entry.Bytes += chunk.bytes
	for len(entry.Chunks) > maxProcessChunks || entry.Bytes > maxProcessBytes { removed := entry.Chunks[0]; entry.Chunks = entry.Chunks[1:]; entry.Bytes -= removed.bytes; entry.Dropped = removed.Seq }
}

func processFor(args map[string]any) (*processEntry, error) {
	id, err := intArg(args, "process_id", 0, 1, 1<<30); if err != nil { return nil, err }
	processTable.Lock(); entry := processTable.m[int64(id)]; processTable.Unlock(); if entry == nil { return nil, failAction("NO_SUCH_PROCESS", "no chaOS process %d", id) }; return entry, nil
}

func readProcess(args map[string]any) (any, error) {
	entry, err := processFor(args); if err != nil { return nil, err }
	after, err := intArg(args, "after", 0, 0, 1<<30); if err != nil { return nil, err }
	entry.mu.Lock(); defer entry.mu.Unlock(); chunks := []processChunk{}; for _, chunk := range entry.Chunks { if chunk.Seq > int64(after) { chunks = append(chunks, processChunk{Seq: chunk.Seq, Stream: chunk.Stream, Text: chunk.Text}) } }
	return map[string]any{"process_id": entry.ID, "running": entry.ExitedAt == 0, "exit_code": entry.ExitCode, "signal": entry.Signal, "started_at": entry.StartedAt, "exited_at": nullableMillis(entry.ExitedAt), "dropped_through": entry.Dropped, "last_seq": entry.NextSeq - 1, "chunks": chunks}, nil
}

func writeProcess(cfg Config, args map[string]any) (any, error) {
	if err := requireCapability(cfg, "command"); err != nil { return nil, err }
	entry, err := processFor(args); if err != nil { return nil, err }
	text, err := stringArg(args, "text", false, 256<<10); if err != nil { return nil, err }
	entry.mu.Lock(); exited := entry.ExitedAt != 0; stdin := entry.Stdin; entry.mu.Unlock(); if exited { return nil, failAction("PROCESS_EXITED", "process %d has exited", entry.ID) }
	n, err := io.WriteString(stdin, text); if err != nil { return nil, err }; return map[string]any{"process_id": entry.ID, "bytes": n}, nil
}

func stopProcess(args map[string]any) (any, error) {
	entry, err := processFor(args); if err != nil { return nil, err }
	entry.mu.Lock(); exited := entry.ExitedAt != 0; pid := entry.Cmd.Process.Pid; entry.mu.Unlock()
	if !exited { _ = killProcessTree(pid) }
	return map[string]any{"process_id": entry.ID, "stopped": true}, nil
}

func killProcessTree(pid int) error {
	if runtime.GOOS == "windows" { return exec.Command("taskkill.exe", "/PID", fmt.Sprint(pid), "/T", "/F").Run() }
	process, err := os.FindProcess(pid); if err != nil { return err }; return process.Kill()
}

func shutdownProcesses() {
	processTable.Lock(); entries := make([]*processEntry, 0, len(processTable.m)); for _, entry := range processTable.m { entries = append(entries, entry) }; processTable.Unlock()
	for _, entry := range entries { entry.mu.Lock(); exited := entry.ExitedAt != 0; pid := entry.Cmd.Process.Pid; entry.mu.Unlock(); if !exited { _ = killProcessTree(pid) } }
}

func nullableMillis(value int64) any { if value == 0 { return nil }; return value }
func truncate(value string, limit int) string { if len(value) <= limit { return value }; return value[:limit] + "…" }

// Keep context imported for future bounded operations without introducing another framework.
var _ = context.Background
