package gitops

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

// ErrBlobMissing signals the requested commit:path doesn't resolve to
// a blob — typically because the file didn't exist at that commit, or
// the rev / path string is malformed. Callers use
// `errors.Is(err, ErrBlobMissing)` to branch this case off from real
// protocol / subprocess errors.
//
// The "missing" semantics match git cat-file --batch's `<input>
// missing\n` response shape: not a subprocess crash, just a "no such
// object" answer to a well-formed request.
var ErrBlobMissing = errors.New("gitops: blob missing at commit:path")

// errBlobReaderClosed is returned by Read after Close. Sentinel so a
// caller that wants to distinguish "the reader closed" from "git
// subprocess died" can branch on it via errors.Is.
var errBlobReaderClosed = errors.New("gitops: BlobReader closed")

// BlobReader is a long-lived `git cat-file --batch` pump. One
// subprocess is launched at construction and reused for all Read
// calls, replacing N short-lived `git show <commit>:<path>`
// invocations elsewhere in the kernel (notably
// `internal/check/fsm_history_consistent.go:351`'s per-(commit, path)
// status reads).
//
// Not safe for concurrent use — git's batch protocol is request /
// response one-at-a-time over a single stdin/stdout pair. Consumers
// serialize Read calls; a future M-NNN that wants concurrency adds a
// worker-pool front-end on top.
//
// Lifetime: callers MUST defer Close to terminate the subprocess.
// Leaking a BlobReader leaves a long-lived `git cat-file --batch`
// process attached to the parent — observable in `ps`, eventual
// fd-exhaustion risk in long-running daemons.
type BlobReader struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	closed bool
}

// NewBlobReader spawns the `git cat-file --batch` subprocess in
// root. Returns an error when root is empty, isn't a git repo, or
// the subprocess can't be started.
//
// Callers MUST defer Close after a successful NewBlobReader. The
// subprocess inherits the parent's environment; identity / config
// considerations follow git's normal layering.
func NewBlobReader(ctx context.Context, root string) (*BlobReader, error) {
	if root == "" {
		return nil, errors.New("gitops: NewBlobReader: root is empty")
	}
	if !IsRepo(ctx, root) {
		return nil, fmt.Errorf("gitops: NewBlobReader: %s is not a git repo", root)
	}
	cmd := exec.CommandContext(ctx, "git", "cat-file", "--batch")
	cmd.Dir = root
	stdin, err := cmd.StdinPipe()
	if err != nil { //coverage:ignore exec.Cmd.StdinPipe documented to fail only when Stdin was already set explicitly; we never set it
		return nil, fmt.Errorf("gitops: NewBlobReader: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil { //coverage:ignore exec.Cmd.StdoutPipe documented to fail only when Stdout was already set explicitly; we never set it
		_ = stdin.Close()
		return nil, fmt.Errorf("gitops: NewBlobReader: stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil { //coverage:ignore exec.Cmd.Start fails only when the binary can't be found or fork fails; tests run in environments where `git` is on PATH (CI gates that elsewhere)
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, fmt.Errorf("gitops: NewBlobReader: start git cat-file: %w", err)
	}
	return &BlobReader{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}, nil
}

// Read fetches the blob content at the named commit:path.
//
// Returns (nil, ErrBlobMissing) when the path doesn't exist at the
// commit, the commit doesn't exist, or the input string is malformed
// — the same skip-this-pair signal `internal/check/fsm_history_
// consistent.go:statusAtCommitPath` returns "" for today.
//
// Returns (blob, nil) on success — the bytes are exactly the file's
// content at that commit, with no trailing newline injected by git's
// batch output (the protocol's framing newline is consumed by the
// parser).
//
// Returns (nil, err) for real failure modes: subprocess crash,
// protocol violation, post-Close call.
func (br *BlobReader) Read(commit, path string) ([]byte, error) {
	if br.closed {
		return nil, errBlobReaderClosed
	}
	request := commit + ":" + path + "\n"
	if _, err := io.WriteString(br.stdin, request); err != nil {
		return nil, fmt.Errorf("gitops: BlobReader.Read: write request: %w", err)
	}
	headerLine, err := br.stdout.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("gitops: BlobReader.Read: read header: %w", err)
	}
	header := strings.TrimRight(headerLine, "\n")
	missing, size, parseErr := parseBatchHeader(header)
	if parseErr != nil {
		return nil, parseErr
	}
	if missing {
		return nil, ErrBlobMissing
	}
	content := make([]byte, size)
	if _, err := io.ReadFull(br.stdout, content); err != nil {
		return nil, fmt.Errorf("gitops: BlobReader.Read: read content (%d bytes): %w", size, err)
	}
	// Consume the trailing LF git appends after content.
	if _, err := br.stdout.ReadByte(); err != nil {
		return nil, fmt.Errorf("gitops: BlobReader.Read: read trailing LF: %w", err)
	}
	return content, nil
}

// Close terminates the subprocess and reaps the exit status.
// Subsequent Read calls return errBlobReaderClosed. Close is
// idempotent — a second call is a no-op.
func (br *BlobReader) Close() error {
	if br.closed {
		return nil
	}
	br.closed = true
	// Closing stdin signals EOF to git cat-file, which exits cleanly.
	// We don't care about the close error if Wait reports the real one.
	closeErr := br.stdin.Close()
	waitErr := br.cmd.Wait()
	if waitErr != nil {
		return fmt.Errorf("gitops: BlobReader.Close: wait git cat-file: %w", waitErr)
	}
	if closeErr != nil { //coverage:ignore stdin.Close on a pipe owned by the cmd is documented to return only ErrClosed (which we swallow above via closed flag) or stdlib pipe errors that don't surface in tests
		return fmt.Errorf("gitops: BlobReader.Close: close stdin: %w", closeErr)
	}
	return nil
}

// parseBatchHeader parses a `git cat-file --batch` header line into
// (missing, size, err). The two shapes:
//
//   - `<sha1> <type> <size>` — found; size is decimal bytes of
//     content that follows the header line + LF.
//   - `<input> missing` — not found; no content follows the header.
//
// Returns err for any other shape (defensive — git's protocol is
// well-specified, but a future flag or a stderr leak into stdout
// would surface here rather than corrupt subsequent reads).
func parseBatchHeader(line string) (missing bool, size int, err error) {
	parts := strings.Fields(line)
	if len(parts) == 2 && parts[1] == "missing" {
		return true, 0, nil
	}
	if len(parts) != 3 {
		return false, 0, fmt.Errorf("gitops: malformed cat-file --batch header: %q", line)
	}
	n, err := strconv.Atoi(parts[2])
	if err != nil {
		return false, 0, fmt.Errorf("gitops: cat-file --batch size parse %q: %w", parts[2], err)
	}
	if n < 0 {
		return false, 0, fmt.Errorf("gitops: cat-file --batch negative size %d in %q", n, line)
	}
	return false, n, nil
}
