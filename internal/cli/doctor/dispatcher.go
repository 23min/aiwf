package doctor

// Dispatcher is the in-process verb dispatcher used by the
// --self-check mode to drive every aiwf verb end-to-end against a
// throwaway repo. It must be wired by cmd/aiwf/main.go's init at
// startup; the doctor package cannot import cmd/aiwf, so the seam
// is a package-level variable.
//
// Default nil. Doctor's --self-check mode refuses with ExitInternal
// when the dispatcher is unset, naming the wiring step so the
// failure points at the right fix.
//
// When cmd/aiwf/main.go itself moves to internal/cli/root in M-0118,
// the wiring becomes a `doctor.Dispatcher = cli.Execute` call inside
// the root package's init.
var Dispatcher func(args []string) int
