package gitops

import (
	"context"
	"errors"
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
	// Stub for AC-2 red phase. Implementation lands in green.
}

// NewBlobReader spawns the `git cat-file --batch` subprocess in
// root. Returns an error when root is empty, isn't a git repo, or
// the subprocess can't be started.
//
// Callers MUST defer Close after a successful NewBlobReader. The
// subprocess inherits the parent's environment; identity / config
// considerations follow git's normal layering.
func NewBlobReader(ctx context.Context, root string) (*BlobReader, error) {
	// Stub for AC-2 red phase. Implementation lands in green.
	_ = ctx
	_ = root
	return nil, errors.New("BlobReader not implemented (M-0137/AC-2 red phase)")
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
	// Stub for AC-2 red phase. Implementation lands in green.
	_ = commit
	_ = path
	return nil, errors.New("BlobReader.Read not implemented (M-0137/AC-2 red phase)")
}

// Close terminates the subprocess and reaps the exit status.
// Subsequent Read calls return an error. Close is idempotent — a
// second call is a no-op.
func (br *BlobReader) Close() error {
	// Stub for AC-2 red phase. Implementation lands in green.
	return nil
}
