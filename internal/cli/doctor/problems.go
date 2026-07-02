package doctor

// Severity is a doctor problem's severity, in the health-file schema's
// vocabulary. Only warn and error are produced: info-level context
// (binary version, env, ok lines) is not a problem and yields nothing.
type Severity string

const (
	// SeverityWarn is an advisory problem that does not block a push.
	SeverityWarn Severity = "warn"
	// SeverityError is a blocking problem — the class doctor's exit code
	// counts.
	SeverityError Severity = "error"
)

// Problem is one doctor warning or error: a severity and a human
// message (the same text the report shows after its label). It is what
// aiwf writes to .claude/health.aiwf.json and what the doctor exit code
// counts; ok/info report lines produce no Problem.
type Problem struct {
	Severity Severity
	Message  string
}

// Problems returns the warnings and errors doctor found for the repo at
// rootDir — the report's problem states without the ok/info context.
func Problems(rootDir string, opts DoctorOptions) []Problem {
	_, p := DoctorReport(rootDir, opts)
	return p
}
