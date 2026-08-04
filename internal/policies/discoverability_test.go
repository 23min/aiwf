package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

// TestDiscoverabilityHaystack_BannerSourceDeclaresPrintHelp asserts
// against the real repository that bannerSourceRel is the file
// declaring printHelp.
//
// The haystack names that file by a fixed path, so the banner channel
// is only in it while the two agree. A synthetic fixture cannot check
// this: it writes its own tree, so it passes whatever the real layout
// is. Moving printHelp to a sibling file while the named path still
// exists drops the channel with every other test still green — the
// policy then reports a code documented only in the banner as
// documented nowhere.
func TestDiscoverabilityHaystack_BannerSourceDeclaresPrintHelp(t *testing.T) {
	t.Parallel()

	files, err := WalkGoFiles(repoRoot(t), true)
	if err != nil {
		t.Fatalf("walking repo: %v", err)
	}

	var declaring []string
	fset := token.NewFileSet()
	for _, f := range files {
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.SkipObjectResolution)
		if perr != nil {
			continue
		}
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Recv == nil && fn.Name.Name == "printHelp" {
				declaring = append(declaring, f.Path)
			}
		}
	}

	switch len(declaring) {
	case 1:
		if declaring[0] != bannerSourceRel {
			t.Errorf("printHelp is declared in %q but the haystack reads the banner from %q; "+
				"repoint bannerSourceRel, or the banner channel is absent from the haystack",
				declaring[0], bannerSourceRel)
		}
	case 0:
		t.Errorf("no file declares printHelp; the banner channel has no source and %q is read for nothing",
			bannerSourceRel)
	default:
		t.Errorf("printHelp is declared in %d files (%v); the haystack reads only %q, so the rest are absent",
			len(declaring), declaring, bannerSourceRel)
	}
}

// TestDiscoverabilityHaystack_CoversBannerChannel pins that the
// binary's help banner is one of the channels
// PolicyFindingCodesAreDiscoverable searches. The policy's doc comment
// and the remediation text it emits both name four channels; a
// singleton path that no longer carries the banner leaves it searching
// three, and a missing channel produces no error of its own — it
// reports a code documented only in the banner as documented nowhere.
//
// Both arms are needed. The "documented only in the banner" arm is what
// fails when the banner path is wrong; the "documented nowhere" arm is
// what proves the first arm passed because the channel was read, not
// because the policy had stopped firing at all.
func TestDiscoverabilityHaystack_CoversBannerChannel(t *testing.T) {
	t.Parallel()

	const code = "zzz-banner-only-code"

	// A kernel code declared in check/ and mentioned in no doc channel
	// but the banner. Held in a string literal inside printHelp, which
	// is how the real banner carries its codes.
	bannerWithCode := "package cli\n\nfunc printHelp() {\n\tprint(\"" + code + ": what it means\\n\")\n}\n"
	bannerWithoutCode := "package cli\n\nfunc printHelp() {\n\tprint(\"no codes here\\n\")\n}\n"

	cases := []struct {
		name      string
		banner    string
		wantFires bool
	}{
		{
			name:      "documented only in the banner passes",
			banner:    bannerWithCode,
			wantFires: false,
		},
		{
			name:      "documented in no channel fires",
			banner:    bannerWithoutCode,
			wantFires: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			files := discoverabilityScaffold()
			files["internal/cli/root.go"] = tc.banner
			files["internal/check/x.go"] = "package check\n\nvar _ = Finding{Code: \"" + code + "\"}\n"

			root := t.TempDir()
			for rel, content := range files {
				mustWrite(t, filepath.Join(root, rel), content)
			}

			vs, err := PolicyFindingCodesAreDiscoverable(root)
			if err != nil {
				t.Fatalf("policy returned error: %v", err)
			}
			got := hasPolicyViolation(vs, "finding-codes-are-discoverable")
			if got != tc.wantFires {
				t.Errorf("policy fired = %v, want %v; violations: %+v", got, tc.wantFires, vs)
			}
		})
	}
}
