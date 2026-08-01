package testsupport

import (
	"fmt"
	"os"
	"syscall"
)

// executablePerm is the mode every test stand-in is written with. It is
// fixed rather than a parameter because the hazard WriteExecutable
// guards is the executable bit itself: a file no one execs cannot be
// caught by ETXTBSY, and a stand-in run only by the test that wrote it
// has no reason to want a narrower mode. Where the mode itself is what
// a test is asserting on, the policy's //exec:ok escape applies.
const executablePerm = 0o755

// WriteExecutable writes an executable stand-in — a shell script posing
// as a binary — so that a concurrently forking test process cannot make
// the subsequent exec of it fail with ETXTBSY ("text file busy").
//
// The hazard is a property of the writing process, not of the file. A
// plain os.WriteFile holds a writable descriptor on the new file between
// its open and its close. A fork anywhere else in the process during
// that window gives the child a copy of that descriptor, and the child
// keeps it until its own execve closes it. While any descriptor holds a
// file open for writing, execve on that file fails with ETXTBSY — so a
// stand-in written by one parallel test can be rejected when a different
// test execs it. In a package whose tests spawn subprocesses in
// parallel, the colliding forks are the suite's own (G-0491).
//
// Holding syscall.ForkLock for reading across the write closes the
// window: syscall's forkExec takes the same lock for writing, so no fork
// this process starts can overlap the descriptor's lifetime. Measured
// under deliberate fork pressure, the guarded write reports zero ETXTBSY
// across 9,600 cycles at baseline pressure and 19,200 more at four times
// it; the unguarded write reports 12% and 17% respectively.
//
// Writing to a temp name and renaming into place does not help and is
// not what this does: ETXTBSY is enforced against the inode, and a
// rename carries the same inode to the new path, leaking descriptor and
// all. Measured, it is indistinguishable from the unguarded write.
//
// Sustained forking does not starve the write: sync.RWMutex wakes every
// blocked reader when a writer unlocks, so a write waiting on forks in
// flight is admitted between them. (Linux and the BSDs additionally
// reference-count the lock and carry an explicit branch admitting
// waiting readers; darwin and aix use a plain Lock/Unlock. The property
// holds on both.)
//
// Point this only at a regular file. Go's own ForkLock documentation
// lists Open as the operation *not* to hold the lock across, because an
// open can block for an unbounded time; a target that blocks — a FIFO
// with no reader, a hung mount — stalls every fork in the process for
// as long as it blocks. The bound here is that callers write a small
// regular file into their own t.TempDir.
//
// On Windows ForkLock exists but goes unused, and there is no ETXTBSY
// either, so the helper is simply a plain write.
func WriteExecutable(path string, data []byte) error {
	syscall.ForkLock.RLock()
	defer syscall.ForkLock.RUnlock()
	if err := os.WriteFile(path, data, executablePerm); err != nil { //nolint:gosec // an executable stand-in is the point; callers write into their own t.TempDir
		return fmt.Errorf("writing executable %s: %w", path, err)
	}
	return nil
}
