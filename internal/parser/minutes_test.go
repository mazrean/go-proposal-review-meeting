package parser_test

import (
	"testing"
	"time"

	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
)

func TestMinutesParser_Parse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		comment     string
		commentedAt time.Time
		want        []struct {
			changedAt     time.Time
			title         string
			currentStatus parser.Status
			issueNumber   int
		}
		wantErr bool
	}{
		{
			name: "single proposal accepted with emoji",
			comment: `**2019-08-20** / @rsc, @griesemer, @andybons, @ianlancetaylor

- #25530 **cmd/go: secure releases with transparency log**
  - **no final comments; accepted 🎉**
`,
			commentedAt: time.Date(2019, 8, 20, 12, 0, 0, 0, time.UTC),
			want: []struct {
				changedAt     time.Time
				title         string
				currentStatus parser.Status
				issueNumber   int
			}{
				{
					issueNumber:   25530,
					title:         "cmd/go: secure releases with transparency log",
					currentStatus: parser.StatusAccepted,
					changedAt:     time.Date(2019, 8, 20, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "single proposal declined",
			comment: `**2019-08-20** / @rsc, @griesemer

- #32405 **errors: simplified error inspection**
  - **no final comments; declined**
`,
			commentedAt: time.Date(2019, 8, 20, 12, 0, 0, 0, time.UTC),
			want: []struct {
				changedAt     time.Time
				title         string
				currentStatus parser.Status
				issueNumber   int
			}{
				{
					issueNumber:   32405,
					title:         "errors: simplified error inspection",
					currentStatus: parser.StatusDeclined,
					changedAt:     time.Date(2019, 8, 20, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "likely accept",
			comment: `**2019-08-13** / @rsc, @griesemer

- #25530 **cmd/go: secure releases with transparency log**
  - commented
  - **likely accept; last call for comments**
`,
			commentedAt: time.Date(2019, 8, 13, 12, 0, 0, 0, time.UTC),
			want: []struct {
				changedAt     time.Time
				title         string
				currentStatus parser.Status
				issueNumber   int
			}{
				{
					issueNumber:   25530,
					title:         "cmd/go: secure releases with transparency log",
					currentStatus: parser.StatusLikelyAccept,
					changedAt:     time.Date(2019, 8, 13, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "likely decline",
			comment: `**2019-08-13** / @rsc, @griesemer

- #32405 **errors: simplified error inspection**
  - commented
  - **likely decline; last call for comments**
`,
			commentedAt: time.Date(2019, 8, 13, 12, 0, 0, 0, time.UTC),
			want: []struct {
				changedAt     time.Time
				title         string
				currentStatus parser.Status
				issueNumber   int
			}{
				{
					issueNumber:   32405,
					title:         "errors: simplified error inspection",
					currentStatus: parser.StatusLikelyDecline,
					changedAt:     time.Date(2019, 8, 13, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "closed as declined",
			comment: `**2019-08-06** / @rsc, @griesemer

- #33454 **log: modify Logger struct**
  - **closed** (backwards-incompatible change)
`,
			commentedAt: time.Date(2019, 8, 6, 12, 0, 0, 0, time.UTC),
			want: []struct {
				changedAt     time.Time
				title         string
				currentStatus parser.Status
				issueNumber   int
			}{
				{
					issueNumber:   33454,
					title:         "log: modify Logger struct",
					currentStatus: parser.StatusDeclined,
					changedAt:     time.Date(2019, 8, 6, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "put on hold",
			comment: `**2019-08-13** / @rsc, @griesemer

- #32456 **net/url: add FromFilePath and ToFilePath**
  - asked for design doc
  - put on hold for design doc
`,
			commentedAt: time.Date(2019, 8, 13, 12, 0, 0, 0, time.UTC),
			want: []struct {
				changedAt     time.Time
				title         string
				currentStatus parser.Status
				issueNumber   int
			}{
				{
					issueNumber:   32456,
					title:         "net/url: add FromFilePath and ToFilePath",
					currentStatus: parser.StatusHold,
					changedAt:     time.Date(2019, 8, 13, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "multiple proposals",
			comment: `**2019-08-20** / @rsc, @griesemer

- #25530 **cmd/go: secure releases with transparency log**
  - **no final comments; accepted 🎉**
- #32405 **errors: simplified error inspection**
  - **no final comments; declined**
- #31572 **net/http: add constant for content type**
  - **no final comments; declined**
`,
			commentedAt: time.Date(2019, 8, 20, 12, 0, 0, 0, time.UTC),
			want: []struct {
				changedAt     time.Time
				title         string
				currentStatus parser.Status
				issueNumber   int
			}{
				{
					issueNumber:   25530,
					title:         "cmd/go: secure releases with transparency log",
					currentStatus: parser.StatusAccepted,
					changedAt:     time.Date(2019, 8, 20, 0, 0, 0, 0, time.UTC),
				},
				{
					issueNumber:   32405,
					title:         "errors: simplified error inspection",
					currentStatus: parser.StatusDeclined,
					changedAt:     time.Date(2019, 8, 20, 0, 0, 0, 0, time.UTC),
				},
				{
					issueNumber:   31572,
					title:         "net/http: add constant for content type",
					currentStatus: parser.StatusDeclined,
					changedAt:     time.Date(2019, 8, 20, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "no status change - only comments",
			comment: `**2019-08-06** / @rsc, @griesemer

- #33388 **cmd/go: add build tags for 32-bit and 64-bit architectures**
  - retitled, commented
- #33203 **cmd/go: add package search functionality**
  - commented
`,
			commentedAt: time.Date(2019, 8, 6, 12, 0, 0, 0, time.UTC),
			want:        nil,
		},
		{
			name: "discussion ongoing",
			comment: `**2019-08-06** / @rsc, @griesemer

- #32816 **cmd/fix: automate migrations for simple deprecations**
  - discussion ongoing (no action taken)
`,
			commentedAt: time.Date(2019, 8, 6, 12, 0, 0, 0, time.UTC),
			want: []struct {
				changedAt     time.Time
				title         string
				currentStatus parser.Status
				issueNumber   int
			}{
				{
					issueNumber:   32816,
					title:         "cmd/fix: automate migrations for simple deprecations",
					currentStatus: parser.StatusDiscussions,
					changedAt:     time.Date(2019, 8, 6, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name:        "empty comment",
			comment:     "",
			commentedAt: time.Date(2019, 8, 6, 12, 0, 0, 0, time.UTC),
			want:        nil,
		},
		{
			name: "invalid format - no date header",
			comment: `This is not a valid minutes format
Just some random text
`,
			commentedAt: time.Date(2019, 8, 6, 12, 0, 0, 0, time.UTC),
			want:        nil,
		},
		{
			name: "no change in consensus accepted",
			comment: `**2020-03-04** / @rsc, @griesemer

- #35667 **cmd/go: add compiler flags, relevant env vars to 'go version -m' output**
  - no change in consensus; **accepted** 🎉
`,
			commentedAt: time.Date(2020, 3, 4, 12, 0, 0, 0, time.UTC),
			want: []struct {
				changedAt     time.Time
				title         string
				currentStatus parser.Status
				issueNumber   int
			}{
				{
					issueNumber:   35667,
					title:         "cmd/go: add compiler flags, relevant env vars to 'go version -m' output",
					currentStatus: parser.StatusAccepted,
					changedAt:     time.Date(2020, 3, 4, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "no change in consensus declined",
			comment: `**2020-03-04** / @rsc, @griesemer

- #34594 **crypto/cipher: Specify nonce and tag sizes for GCM**
  - no change in consensus; **declined**
`,
			commentedAt: time.Date(2020, 3, 4, 12, 0, 0, 0, time.UTC),
			want: []struct {
				changedAt     time.Time
				title         string
				currentStatus parser.Status
				issueNumber   int
			}{
				{
					issueNumber:   34594,
					title:         "crypto/cipher: Specify nonce and tag sizes for GCM",
					currentStatus: parser.StatusDeclined,
					changedAt:     time.Date(2020, 3, 4, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "likely accept with emoji",
			comment: `**2020-03-04** / @rsc, @griesemer

- #35499 **crypto/tls expose names for CurveID and SignatureScheme**
  - **likely accept**; last call for comments ⏳
`,
			commentedAt: time.Date(2020, 3, 4, 12, 0, 0, 0, time.UTC),
			want: []struct {
				changedAt     time.Time
				title         string
				currentStatus parser.Status
				issueNumber   int
			}{
				{
					issueNumber:   35499,
					title:         "crypto/tls expose names for CurveID and SignatureScheme",
					currentStatus: parser.StatusLikelyAccept,
					changedAt:     time.Date(2020, 3, 4, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "likely decline with emoji",
			comment: `**2020-03-04** / @rsc, @griesemer

- #36887 **sort: add InvertSlice**
  - **likely decline**; last call for comments ⏳
`,
			commentedAt: time.Date(2020, 3, 4, 12, 0, 0, 0, time.UTC),
			want: []struct {
				changedAt     time.Time
				title         string
				currentStatus parser.Status
				issueNumber   int
			}{
				{
					issueNumber:   36887,
					title:         "sort: add InvertSlice",
					currentStatus: parser.StatusLikelyDecline,
					changedAt:     time.Date(2020, 3, 4, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "proposal retracted declined",
			comment: `**2020-03-11** / @rsc, @griesemer

- #37642 **runtime: make print/println print interfaces**
  - proposal retracted by author; **declined**
`,
			commentedAt: time.Date(2020, 3, 11, 12, 0, 0, 0, time.UTC),
			want: []struct {
				changedAt     time.Time
				title         string
				currentStatus parser.Status
				issueNumber   int
			}{
				{
					issueNumber:   37642,
					title:         "runtime: make print/println print interfaces",
					currentStatus: parser.StatusDeclined,
					changedAt:     time.Date(2020, 3, 11, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := parser.NewMinutesParser()
			got, err := p.Parse(tt.comment, tt.commentedAt)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(got) != len(tt.want) {
				t.Fatalf("Parse() returned %d changes, want %d", len(got), len(tt.want))
			}

			for i, want := range tt.want {
				if got[i].IssueNumber != want.issueNumber {
					t.Errorf("changes[%d].IssueNumber = %d, want %d", i, got[i].IssueNumber, want.issueNumber)
				}
				if got[i].Title != want.title {
					t.Errorf("changes[%d].Title = %q, want %q", i, got[i].Title, want.title)
				}
				if got[i].CurrentStatus != want.currentStatus {
					t.Errorf("changes[%d].CurrentStatus = %s, want %s", i, got[i].CurrentStatus, want.currentStatus)
				}
				if !got[i].ChangedAt.Equal(want.changedAt) {
					t.Errorf("changes[%d].ChangedAt = %v, want %v", i, got[i].ChangedAt, want.changedAt)
				}
			}
		})
	}
}

// TestMinutesParser_Parse_RealComment tests with actual GitHub comment data.
func TestMinutesParser_Parse_RealComment(t *testing.T) {
	t.Parallel()

	// This is an actual comment from https://github.com/golang/go/issues/33502
	realComment := `**2020-04-08 / @rsc, @griesemer, @ianlancetaylor, @bradfitz, @andybons, @spf13**

- #37503 **all: add bare metal ARM support**
  - commented
- #34527 **cmd/go: add GOMODCACHE**
  - no change in consensus; **accepted** 🎉
- #25348 **cmd/go: allow && and || operators and parentheses in build tags**
  - put on hold
- #37641 **cmd/go: reserve specific path prefixes for local (user-defined) modules**
  - discussion ongoing
- #37475 **cmd/go: stamp git/vcs current HEAD hash/commit hash/dirty bit in binaries**
  - no change in consensus; **accepted** 🎉
- #34544 **cmd/vet: detect defer rows.Close()**
  - cc'ed owners
- #37168 **crypto: new assembly policy**
  - discussion ongoing
- #24171 **crypto/cipher: allow short tags in NewGCMWithNonceAndTagSize**
  - no change in consensus; **declined**
- #38158 **dep: officially deprecate the "dep" experiment**
  - commented
- #32779 **encoding/json: memoize strings during decode**
  - no change in consensus; **accepted** 🎉
- #37974 **go/doc: drop //go:* comments from extracted docs**
  - commented
- #36450 **index/suffixarray: added functionality via longest common prefix array**
  - discussion ongoing
- #37776 **net/url: add URL.RawFragment, like RawPath**
  - no change in consensus; **accepted** 🎉
- #36141 **runtime: "real time" timer semantics**
  - commented
- #37112 **runtime: API for unstable metrics**
  - **likely accept**; last call for comments ⏳
- #36771 **strconv: add ParseComplex**
  - no change in consensus; **accepted** 🎉
- #33194 **testing: add B.Lap for phases of benchmarks**
  - commented
- #31107 **text/template, html/template: add ExecuteContext methods**
  - no change in consensus; **declined**
- #34652 **text/template/parse: add CommentNode to the parse tree**
  - cc'ed owners
- #33273 **text/template: allow template and block outputs to be chained**
  - commented
- #38017 **time: add time/tzdata package and timetzdata tag to embed tzdata in program**
  - no change in consensus; **accepted** 🎉
- #29390 **x/crypto: add implementation of Diffie-Hellman x448**
  - discussion ongoing
- #34508 **x/tools/go/analysis: add tags or codes to diagnostics**
  - cc'ed owners
- #35921 **x/tools/go/packages: add modinfo.ModulePublic to packages.Package**
  - cc'ed owners
- #33595 **x/tools/gopls: support for per-.go file builds**
  - commented
`
	commentedAt := time.Date(2020, 4, 8, 18, 26, 15, 0, time.UTC)

	p := parser.NewMinutesParser()
	changes, err := p.Parse(realComment, commentedAt)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Expected status changes from this comment
	expectedChanges := map[int]parser.Status{
		34527: parser.StatusAccepted,     // cmd/go: add GOMODCACHE
		25348: parser.StatusHold,         // cmd/go: allow && and ||
		37641: parser.StatusDiscussions,  // cmd/go: reserve specific path
		37475: parser.StatusAccepted,     // cmd/go: stamp git/vcs
		37168: parser.StatusDiscussions,  // crypto: new assembly policy
		24171: parser.StatusDeclined,     // crypto/cipher: allow short tags
		32779: parser.StatusAccepted,     // encoding/json: memoize strings
		36450: parser.StatusDiscussions,  // index/suffixarray
		37776: parser.StatusAccepted,     // net/url: add URL.RawFragment
		37112: parser.StatusLikelyAccept, // runtime: API for unstable metrics
		36771: parser.StatusAccepted,     // strconv: add ParseComplex
		31107: parser.StatusDeclined,     // text/template: add ExecuteContext
		38017: parser.StatusAccepted,     // time: add time/tzdata
		29390: parser.StatusDiscussions,  // x/crypto: add x448
	}

	// Build map of actual changes
	actualChanges := make(map[int]parser.Status)
	for _, c := range changes {
		actualChanges[c.IssueNumber] = c.CurrentStatus
	}

	// Verify all expected changes are present
	for issueNum, expectedStatus := range expectedChanges {
		actualStatus, ok := actualChanges[issueNum]
		if !ok {
			t.Errorf("missing change for issue #%d, expected status %s", issueNum, expectedStatus)
			continue
		}
		if actualStatus != expectedStatus {
			t.Errorf("issue #%d: got status %s, want %s", issueNum, actualStatus, expectedStatus)
		}
	}

	// Verify date is correct
	expectedDate := time.Date(2020, 4, 8, 0, 0, 0, 0, time.UTC)
	for _, c := range changes {
		if !c.ChangedAt.Equal(expectedDate) {
			t.Errorf("issue #%d: got date %v, want %v", c.IssueNumber, c.ChangedAt, expectedDate)
		}
	}
}
