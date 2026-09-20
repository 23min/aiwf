package gitops

import "testing"

// TestRefusedCoauthorIn pins the decision the Co-Authored-By refusal rests
// on: which value, if any, names an address the caller declared it refuses.
func TestRefusedCoauthorIn(t *testing.T) {
	t.Parallel()

	refused := map[string]bool{"noreply@example.invalid": true}

	cases := []struct {
		name  string
		block string
		want  string
	}{
		{
			name:  "a refused address is named",
			block: "Co-Authored-By: Someone <noreply@example.invalid>\n",
			want:  "noreply@example.invalid",
		},
		{
			// Git compares trailer keys without regard to case and its own
			// tooling writes both spellings, so both name this trailer.
			name:  "the trailer key matches whatever its case",
			block: "Co-authored-by: Someone <noreply@example.invalid>\n",
			want:  "noreply@example.invalid",
		},
		{
			// Addresses compare case-insensitively, as mail addresses do.
			name:  "the address matches whatever its case",
			block: "Co-Authored-By: Someone <NoReply@Example.Invalid>\n",
			want:  "noreply@example.invalid",
		},
		{
			name:  "an address the caller did not list passes",
			block: "Co-Authored-By: A Person <person@example.invalid>\n",
			want:  "",
		},
		{
			name:  "a trailer block carrying no co-author passes",
			block: "aiwf-verb: promote\naiwf-entity: M-0001\n",
			want:  "",
		},
		{
			// The refusal reads the whole block, not only its last line.
			name:  "a co-author among other trailers is named",
			block: "aiwf-verb: promote\nCo-Authored-By: Someone <noreply@example.invalid>\naiwf-entity: M-0001\n",
			want:  "noreply@example.invalid",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := RefusedCoauthorIn(tc.block, refused); got != tc.want {
				t.Errorf("RefusedCoauthorIn() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestCoauthorAddress pins which span of a value is read as the address.
// Each case is a distinct decision in the parse, not a second spelling of
// one already covered above.
func TestCoauthorAddress(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "a display name and a bracketed address yields the address",
			in:   "Someone <a@example.invalid>",
			want: "a@example.invalid",
		},
		{
			// The harness that writes this line may omit the display name.
			// Taking the whole value here would yield "<a@example.invalid>",
			// which matches no configured address — a silent bypass.
			name: "a bracketed address with no display name yields the address",
			in:   "<a@example.invalid>",
			want: "a@example.invalid",
		},
		{
			name: "a bare address with no brackets yields the address",
			in:   "A@Example.Invalid",
			want: "a@example.invalid",
		},
		{
			name: "space inside and around the brackets is trimmed",
			in:   "  Someone <  a@example.invalid  >  ",
			want: "a@example.invalid",
		},
		{
			// The last pair wins, so an alias written ahead of the real
			// address cannot hide it from the refusal.
			name: "a value with two bracket pairs yields the last",
			in:   "A <alias@example.invalid> Name <real@example.invalid>",
			want: "real@example.invalid",
		},
		{
			// An unterminated bracket is not an address; the value is taken
			// whole so it matches nothing rather than matching a prefix.
			name: "an unclosed bracket yields the whole value",
			in:   "Someone <a@example.invalid",
			want: "someone <a@example.invalid",
		},
		{
			name: "empty brackets yield an empty address",
			in:   "Someone <>",
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := coauthorAddress(tc.in); got != tc.want {
				t.Errorf("coauthorAddress(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
