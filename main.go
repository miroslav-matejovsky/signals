// Command signals surfaces significant architectural changes in a local Git
// repository.
//
// Usage:
//
//	signals scan --repo <path> --architecture <file> --since <duration> [--format text|json]
//
// It collects the commits of the period ending now, groups them into
// ChangeSets, detects signals against the declared architecture, and prints
// the ChangeSets that deserve architect attention.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/miroslav-matejovsky/signals/architecture"
	"github.com/miroslav-matejovsky/signals/attention"
	"github.com/miroslav-matejovsky/signals/collection"
	"github.com/miroslav-matejovsky/signals/detection"
)

const usage = "usage: signals scan --repo <path> --architecture <file> --since <duration> [--format text|json]"

const (
	formatText = "text"
	formatJSON = "json"
)

func main() {
	if err := run(context.Background(), os.Args[1:], time.Now(), os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "signals:", err)
		os.Exit(1)
	}
}

// run executes the command line args, without the program name, for the scan
// period ending at now, and writes the report to out.
func run(ctx context.Context, args []string, now time.Time, out io.Writer) error {
	if len(args) == 0 || args[0] != "scan" {
		return errors.New(usage)
	}
	opts, err := parseScanArgs(args[1:])
	if err != nil {
		return err
	}
	arch, err := architecture.Load(opts.architecture)
	if err != nil {
		return err
	}
	since := now.Add(-opts.since)
	repo, err := collection.Collect(ctx, opts.repo, since)
	if err != nil {
		return err
	}
	signals, err := detection.Detect(repo, arch)
	if err != nil {
		return err
	}
	candidates, err := attention.Select(repo.ChangeSets, signals)
	if err != nil {
		return err
	}
	r := &report{
		Repository:    repo.Name,
		Since:         since,
		ChangeSets:    len(repo.ChangeSets),
		WithoutSignal: len(repo.ChangeSets) - len(candidates),
		Candidates:    candidates,
	}
	if opts.format == formatJSON {
		return writeJSON(out, r)
	}
	return writeText(out, r)
}

type scanOptions struct {
	repo         string
	architecture string
	since        time.Duration
	format       string
}

func parseScanArgs(args []string) (scanOptions, error) {
	var opts scanOptions
	var since string
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.repo, "repo", "", "path to a local Git repository")
	fs.StringVar(&opts.architecture, "architecture", "", "path to the architecture YAML file")
	fs.StringVar(&since, "since", "", "length of the scan period ending now, such as 7d or 36h")
	fs.StringVar(&opts.format, "format", formatText, "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return scanOptions{}, fmt.Errorf("%w\n%s", err, usage)
	}
	if fs.NArg() > 0 {
		return scanOptions{}, fmt.Errorf("unexpected arguments %q\n%s", fs.Args(), usage)
	}
	if opts.repo == "" || opts.architecture == "" || since == "" {
		return scanOptions{}, fmt.Errorf("--repo, --architecture, and --since are required\n%s", usage)
	}
	if opts.format != formatText && opts.format != formatJSON {
		return scanOptions{}, fmt.Errorf("unknown format %q, use text or json", opts.format)
	}
	d, err := parseSince(since)
	if err != nil {
		return scanOptions{}, err
	}
	opts.since = d
	return opts, nil
}

// parseSince parses a positive duration given in whole days, such as 7d, or
// in time.ParseDuration syntax, such as 36h.
func parseSince(s string) (time.Duration, error) {
	var d time.Duration
	if days, ok := strings.CutSuffix(s, "d"); ok {
		n, err := strconv.Atoi(days)
		if err != nil {
			return 0, fmt.Errorf("invalid --since %q: %w", s, err)
		}
		d = time.Duration(n) * 24 * time.Hour
	} else {
		var err error
		if d, err = time.ParseDuration(s); err != nil {
			return 0, fmt.Errorf("invalid --since %q: %w", s, err)
		}
	}
	if d <= 0 {
		return 0, fmt.Errorf("invalid --since %q: must be positive", s)
	}
	return d, nil
}

// report is the outcome of one scan.
type report struct {
	Repository string    `json:"repository"`
	Since      time.Time `json:"since"`
	ChangeSets int       `json:"changeSets"`
	// WithoutSignal counts suppressed ChangeSets. Their architectural impact
	// is unknown, not assessed as safe.
	WithoutSignal int                   `json:"withoutSignal"`
	Candidates    []attention.Candidate `json:"candidates"`
}

func writeJSON(w io.Writer, r *report) error {
	if r.Candidates == nil {
		r.Candidates = []attention.Candidate{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func writeText(w io.Writer, r *report) error {
	var b strings.Builder
	for _, c := range r.Candidates {
		fmt.Fprintf(&b, "%s  %s\n\nCommits:\n", c.Level, c.ChangeSet)
		for _, cm := range c.Commits {
			fmt.Fprintf(&b, "  %s  %s  %s  %s\n", short(cm.Hash), cm.Time.Format(time.DateOnly), cm.Author, cm.Subject)
		}
		b.WriteString("\nSignals:\n")
		for _, s := range c.Signals {
			fmt.Fprintf(&b, "  %-28s  %s\n", s.Type, s.Summary)
		}
		b.WriteString("\nEvidence:\n")
		for _, s := range c.Signals {
			fmt.Fprintf(&b, "  %s\n", s.Type)
			for _, e := range s.Evidence {
				fmt.Fprintf(&b, "    %s %s %s (%s)\n", short(e.Commit), e.Path, e.Status, e.Reason)
			}
		}
		b.WriteString("\n\n")
	}
	fmt.Fprintf(&b, "Repository %s: %d change sets since %s.\n", r.Repository, r.ChangeSets, r.Since.Format(time.RFC3339))
	fmt.Fprintf(&b, "%d attention candidates. %d change sets without architectural signal: impact unknown, not assessed as safe.\n",
		len(r.Candidates), r.WithoutSignal)
	_, err := io.WriteString(w, b.String())
	return err
}

func short(hash string) string {
	return hash[:min(len(hash), 7)]
}
