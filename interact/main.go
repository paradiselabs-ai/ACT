// interact v0 — dumb PTY harness for `act`.
//
// Launches `act --project <name>` in a pseudo-terminal, types a prompt,
// captures output for a fixed duration, then prints a one-screen report.
// No LLM, no decisions. Just: launch, type, wait, dump, grade.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

func main() {
	project := flag.String("project", "interact-test", "project name (also used as run dir)")
	prompt := flag.String("prompt", "", "prompt to type into act (or use -prompt-file)")
	promptFile := flag.String("prompt-file", "", "read prompt from this file")
	timeout := flag.Duration("timeout", 90*time.Second, "max time to capture after typing the prompt")
	actBin := flag.String("act", "act", "path to the act binary")
	serverLog := flag.String("server-log", "../server/data/coordination-log.jsonl", "ACT server coordination log path")
	flag.Parse()

	if *promptFile != "" {
		b, err := os.ReadFile(*promptFile)
		if err != nil {
			die("read prompt file: %v", err)
		}
		*prompt = string(b)
	}
	if strings.TrimSpace(*prompt) == "" {
		die("need -prompt or -prompt-file")
	}

	runDir := filepath.Join("/tmp/interact-runs", *project)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		die("mkdir run dir: %v", err)
	}
	rawPath := filepath.Join(runDir, "run.log")
	txtPath := filepath.Join(runDir, "run.txt")

	rawF, err := os.Create(rawPath)
	if err != nil {
		die("create run.log: %v", err)
	}
	defer rawF.Close()

	fmt.Printf("interact: project=%s runDir=%s\n", *project, runDir)
	fmt.Printf("interact: spawning %s --project %s\n", *actBin, *project)

	cmd := exec.Command(*actBin, "--project", *project)
	cmd.Dir = runDir
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		die("pty start: %v", err)
	}
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 40, Cols: 120})

	var (
		mu         sync.Mutex
		buf        strings.Builder
		lastByteAt = time.Now()
	)

	// Reader goroutine: copy PTY → file + buffer.
	done := make(chan struct{})
	go func() {
		defer close(done)
		b := make([]byte, 4096)
		for {
			n, err := ptmx.Read(b)
			if n > 0 {
				rawF.Write(b[:n])
				mu.Lock()
				buf.Write(b[:n])
				lastByteAt = time.Now()
				mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()

	// Wait for TUI to settle: 2s of no new bytes after first output.
	settleStart := time.Now()
	for {
		time.Sleep(250 * time.Millisecond)
		mu.Lock()
		idle := time.Since(lastByteAt)
		hasOutput := buf.Len() > 0
		mu.Unlock()
		if hasOutput && idle > 2*time.Second {
			break
		}
		if time.Since(settleStart) > 15*time.Second {
			fmt.Println("interact: settle timeout (15s) — typing anyway")
			break
		}
	}

	// New-project init dialog: a single Yes/No "Initialize Project?" prompt.
	// "Yes" runs the hardcoded `init` command which asks the Planner to analyze
	// the existing codebase and write ACT.md — that's wrong for a brand-new
	// empty project (nothing to analyze). We dismiss with `n`, then the very
	// first chat message becomes the project requirement (Snake Arena, etc.).
	fmt.Println("interact: dismissing init dialog (n)")
	ptmx.Write([]byte{'n'})
	// Brief pause for the dialog to close and the chat input to focus.
	time.Sleep(1 * time.Second)

	fmt.Printf("interact: typing prompt (%d bytes)\n", len(*prompt))
	io.WriteString(ptmx, *prompt)
	io.WriteString(ptmx, "\r")

	// Capture window.
	deadline := time.Now().Add(*timeout)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		mu.Lock()
		s := buf.String()
		mu.Unlock()
		stripped := ansiRE.ReplaceAllString(s, "")
		if strings.Contains(stripped, "SYNTHESIS_COMPLETE") || strings.Contains(stripped, "QA_COMPLETE") {
			fmt.Println("interact: detected completion sentinel — stopping early")
			break
		}
	}

	// Shutdown: Ctrl+C, Ctrl+C, SIGKILL.
	fmt.Println("interact: sending Ctrl+C")
	ptmx.Write([]byte{0x03})
	time.Sleep(2 * time.Second)
	if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
		ptmx.Write([]byte{0x03})
		time.Sleep(1 * time.Second)
	}
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		_ = cmd.Process.Kill()
	}
	ptmx.Close()
	<-done
	cmd.Wait()

	// Write stripped txt.
	mu.Lock()
	raw := buf.String()
	mu.Unlock()
	stripped := ansiRE.ReplaceAllString(raw, "")
	os.WriteFile(txtPath, []byte(stripped), 0o644)

	// Report.
	exitCode := -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	createTaskCount := strings.Count(stripped, "CREATE_TASK")
	roleCounts := map[string]int{
		"Planner":   countCI(stripped, "planner"),
		"Observer":  countCI(stripped, "observer"),
		"Assurance": countCI(stripped, "assurance"),
		"QA":        countCI(stripped, "qa"),
	}
	panics := findLines(stripped, "panic")
	errors := findLines(stripped, "error")
	fatals := findLines(stripped, "fatal")
	lines := strings.Count(stripped, "\n")

	fmt.Println()
	fmt.Println("════════ interact report ════════")
	fmt.Printf(" exit code:        %d\n", exitCode)
	fmt.Printf(" raw bytes:        %d\n", len(raw))
	fmt.Printf(" stripped lines:   %d\n", lines)
	fmt.Printf(" CREATE_TASK seen: %d\n", createTaskCount)
	fmt.Printf(" role tags:        Planner=%d Observer=%d Assurance=%d QA=%d\n",
		roleCounts["Planner"], roleCounts["Observer"], roleCounts["Assurance"], roleCounts["QA"])
	fmt.Printf(" panics:           %d %v\n", len(panics), truncList(panics, 5))
	fmt.Printf(" 'error' lines:    %d\n", len(errors))
	fmt.Printf(" 'fatal' lines:    %d %v\n", len(fatals), truncList(fatals, 5))
	fmt.Printf(" run.log:          %s\n", rawPath)
	fmt.Printf(" run.txt:          %s\n", txtPath)
	fmt.Println()
	fmt.Println("──── coordination-log.jsonl tail (20) ────")
	tailFile(*serverLog, 20)
	fmt.Println()
	fmt.Println("──── project dir contents ────")
	listDir(runDir)
	fmt.Println("══════════════════════════════════")

	if len(panics) > 0 || createTaskCount == 0 {
		os.Exit(1)
	}
}

func die(f string, a ...any) {
	fmt.Fprintf(os.Stderr, "interact: "+f+"\n", a...)
	os.Exit(2)
}

func countCI(s, sub string) int {
	return strings.Count(strings.ToLower(s), strings.ToLower(sub))
}

func findLines(s, needle string) []int {
	var out []int
	low := strings.ToLower(s)
	low_needle := strings.ToLower(needle)
	for i, line := range strings.Split(low, "\n") {
		if strings.Contains(line, low_needle) {
			out = append(out, i+1)
		}
	}
	return out
}

func truncList(xs []int, n int) []int {
	if len(xs) <= n {
		return xs
	}
	return xs[:n]
}

func tailFile(path string, n int) {
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf(" (could not read %s: %v)\n", path, err)
		return
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	for _, l := range lines {
		fmt.Println(" " + l)
	}
}

func listDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf(" (could not read %s: %v)\n", dir, err)
		return
	}
	for _, e := range entries {
		fmt.Println(" " + e.Name())
	}
}
