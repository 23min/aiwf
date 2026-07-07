package logger

import (
	"log/slog"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestResolveConfigWithSources_ReportsWhichTierWon pins M-0238/AC-4:
// aiwf doctor needs to explain, per field, whether env, yaml, or the
// default supplied the resolved value — not just the merged result.
func TestResolveConfigWithSources_ReportsWhichTierWon(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		getenv     map[string]string
		yamlCfg    YAMLConfig
		wantSource Sources
	}{
		{
			name:       "env wins all three",
			getenv:     map[string]string{"AIWF_LOG": "info", "AIWF_LOG_FORMAT": "json", "AIWF_LOG_FILE": "stderr"},
			yamlCfg:    YAMLConfig{Level: "error", Format: "text", Destination: "/other.log"},
			wantSource: Sources{Level: SourceEnv, Format: SourceEnv, Destination: SourceEnv},
		},
		{
			name:       "yaml wins format and destination when env unset",
			getenv:     map[string]string{"AIWF_LOG": "info"},
			yamlCfg:    YAMLConfig{Format: "json", Destination: "/some.log"},
			wantSource: Sources{Level: SourceEnv, Format: SourceYAML, Destination: SourceYAML},
		},
		{
			name:       "level from yaml, format and destination default",
			getenv:     map[string]string{},
			yamlCfg:    YAMLConfig{Level: "debug"},
			wantSource: Sources{Level: SourceYAML, Format: SourceDefault, Destination: SourceDefault},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			getenv := func(key string) string { return tc.getenv[key] }
			_, sources, err := ResolveConfigWithSources(getenv, tc.yamlCfg)
			if err != nil {
				t.Fatalf("ResolveConfigWithSources: %v", err)
			}
			if sources != tc.wantSource {
				t.Errorf("sources = %+v, want %+v", sources, tc.wantSource)
			}
		})
	}
}

// TestResolveConfigWithSources_Disabled pins the disabled case: no
// level from any source reports the zero Sources value, not a
// misleading "default" label on fields that never resolved.
func TestResolveConfigWithSources_Disabled(t *testing.T) {
	t.Parallel()
	getenv := func(string) string { return "" }
	cfg, sources, err := ResolveConfigWithSources(getenv, YAMLConfig{})
	if err != nil {
		t.Fatalf("ResolveConfigWithSources: %v", err)
	}
	if cfg.Enabled {
		t.Error("cfg.Enabled = true, want false")
	}
	if sources != (Sources{}) {
		t.Errorf("sources = %+v, want the zero value", sources)
	}
}

// TestResolveConfig_MatchesResolveConfigWithSources pins that
// ResolveConfig (the pre-existing, still-used API) is a thin wrapper
// over ResolveConfigWithSources, not a second, divergent
// implementation of the same precedence rule.
func TestResolveConfig_MatchesResolveConfigWithSources(t *testing.T) {
	t.Parallel()
	getenv := func(key string) string {
		return map[string]string{"AIWF_LOG": "warn", "AIWF_LOG_FORMAT": "json"}[key]
	}
	yamlCfg := YAMLConfig{Destination: "/some.log"}
	got, err := ResolveConfig(getenv, yamlCfg)
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	want, _, err := ResolveConfigWithSources(getenv, yamlCfg)
	if err != nil {
		t.Fatalf("ResolveConfigWithSources: %v", err)
	}
	if got != want {
		t.Errorf("ResolveConfig() = %+v, want %+v (from ResolveConfigWithSources)", got, want)
	}
}

// TestResolveConfig_Precedence covers the full env/yaml/default matrix
// ADR-0017 Decision #3 specifies: AIWF_LOG/AIWF_LOG_FORMAT/AIWF_LOG_FILE
// each beat the corresponding aiwf.yaml logging: key, which beats the
// default — resolved independently per setting.
func TestResolveConfig_Precedence(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		env  map[string]string
		yaml YAMLConfig
		want Config
	}{
		{
			name: "neither set: disabled, zero value",
			want: Config{},
		},
		{
			name: "yaml only: level enables logging",
			yaml: YAMLConfig{Level: "debug"},
			want: Config{Enabled: true, Level: slog.LevelDebug, Format: "text"},
		},
		{
			name: "env only: level enables logging",
			env:  map[string]string{"AIWF_LOG": "info"},
			want: Config{Enabled: true, Level: slog.LevelInfo, Format: "text"},
		},
		{
			name: "both set: env level wins",
			env:  map[string]string{"AIWF_LOG": "warn"},
			yaml: YAMLConfig{Level: "debug"},
			want: Config{Enabled: true, Level: slog.LevelWarn, Format: "text"},
		},
		{
			name: "yaml format applies when env format unset",
			env:  map[string]string{"AIWF_LOG": "error"},
			yaml: YAMLConfig{Format: "json"},
			want: Config{Enabled: true, Level: slog.LevelError, Format: "json"},
		},
		{
			name: "env format wins over yaml format",
			env:  map[string]string{"AIWF_LOG": "error", "AIWF_LOG_FORMAT": "json"},
			yaml: YAMLConfig{Format: "text"},
			want: Config{Enabled: true, Level: slog.LevelError, Format: "json"},
		},
		{
			name: "yaml destination applies when env destination unset",
			env:  map[string]string{"AIWF_LOG": "error"},
			yaml: YAMLConfig{Destination: "/custom/path.log"},
			want: Config{Enabled: true, Level: slog.LevelError, Format: "text", Destination: "/custom/path.log"},
		},
		{
			name: "env destination wins over yaml destination",
			env:  map[string]string{"AIWF_LOG": "error", "AIWF_LOG_FILE": "stderr"},
			yaml: YAMLConfig{Destination: "/custom/path.log"},
			want: Config{Enabled: true, Level: slog.LevelError, Format: "text", Destination: "stderr"},
		},
		{
			name: "format alone does not opt in without a level",
			env:  map[string]string{"AIWF_LOG_FORMAT": "json"},
			yaml: YAMLConfig{Destination: "/custom/path.log"},
			want: Config{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			getenv := func(key string) string { return tc.env[key] }
			got, err := ResolveConfig(getenv, tc.yaml)
			if err != nil {
				t.Fatalf("ResolveConfig() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Fatalf("ResolveConfig() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestResolveConfig_InvalidValues confirms a level or format value
// outside ADR-0017's closed sets is rejected rather than silently
// accepted or defaulted, from either source.
func TestResolveConfig_InvalidValues(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		env     map[string]string
		yaml    YAMLConfig
		wantErr string
	}{
		{"invalid env level", map[string]string{"AIWF_LOG": "verbose"}, YAMLConfig{}, `invalid level "verbose"`},
		{"invalid yaml level", nil, YAMLConfig{Level: "trace"}, `invalid level "trace"`},
		{"invalid env format", map[string]string{"AIWF_LOG": "info", "AIWF_LOG_FORMAT": "xml"}, YAMLConfig{}, `invalid format "xml"`},
		{"invalid yaml format", map[string]string{"AIWF_LOG": "info"}, YAMLConfig{Format: "xml"}, `invalid format "xml"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			getenv := func(key string) string { return tc.env[key] }
			_, err := ResolveConfig(getenv, tc.yaml)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("ResolveConfig() error = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

// TestYAMLConfig_DecodesFromAiwfYAMLLoggingBlock pins YAMLConfig's yaml
// struct tags against a real aiwf.yaml logging: block — ResolveConfig's
// tests above construct YAMLConfig literals directly and so never
// exercise the tags themselves; a tag typo (e.g. yaml:"dest" instead of
// yaml:"destination") would silently break yaml-driven configuration
// with none of those tests catching it.
func TestYAMLConfig_DecodesFromAiwfYAMLLoggingBlock(t *testing.T) {
	t.Parallel()
	const doc = `
logging:
  level: debug
  format: json
  destination: /var/log/aiwf.log
`
	var root struct {
		Logging YAMLConfig `yaml:"logging"`
	}
	if err := yaml.Unmarshal([]byte(doc), &root); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	want := YAMLConfig{Level: "debug", Format: "json", Destination: "/var/log/aiwf.log"}
	if root.Logging != want {
		t.Fatalf("decoded YAMLConfig = %+v, want %+v", root.Logging, want)
	}
}

// TestYAMLConfig_AbsentBlockDecodesToZeroValue pins the "all three keys
// optional" half of ADR-0017 Decision #3: an aiwf.yaml with no logging:
// block at all decodes to the zero value, which ResolveConfig already
// treats as fully unset.
func TestYAMLConfig_AbsentBlockDecodesToZeroValue(t *testing.T) {
	t.Parallel()
	var root struct {
		Logging YAMLConfig `yaml:"logging"`
	}
	if err := yaml.Unmarshal([]byte("hosts: [claude-code]\n"), &root); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	if root.Logging != (YAMLConfig{}) {
		t.Fatalf("decoded YAMLConfig = %+v, want the zero value", root.Logging)
	}
}
