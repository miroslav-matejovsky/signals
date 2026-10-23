package collection

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Status is how a commit changed a path.
type Status string

// Statuses of a Change. Git type changes, such as a file becoming a symbolic
// link, are reported as Modified.
const (
	Added    Status = "added"
	Modified Status = "modified"
	Deleted  Status = "deleted"
)

// Commit is the basic metadata of one Git commit.
type Commit struct {
	Hash    string    `json:"hash"`
	Author  string    `json:"author"`
	Time    time.Time `json:"time"`
	Subject string    `json:"subject"`
}

// Change is a path added, modified, or deleted by the commit with hash Commit.
// Path is slash-separated and relative to the repository root.
type Change struct {
	Commit string
	Path   string
	Status Status
}

// ChangeSet groups the commits that implement one change.
//
// Commits whose subject starts with the same work-item key, such as
// "PAY-1421 add refunds" or "[PAY-1421] add refunds", form one ChangeSet
// identified by that key. Every other commit forms its own ChangeSet
// identified by its hash abbreviated to 12 characters.
type ChangeSet struct {
	ID string
	// Commits are ordered newest first.
	Commits []Commit
	// Files lists the file changes of all commits, in commit order.
	Files []Change
	// Directories lists the directories the commits created (Added) or
	// removed (Deleted), in commit order. Git tracks no empty directories, so
	// a directory is created by adding its first file and removed by deleting
	// its last file.
	Directories []Change
}

// Repository is the result of collecting one local Git repository.
type Repository struct {
	// Name is the base name of the repository's top-level directory. The
	// architecture declaration refers to repositories by this name.
	Name string
	// ChangeSets are ordered by their newest commit, newest first.
	ChangeSets []ChangeSet
}

// Collect reads the non-merge commits reachable from HEAD of the Git
// repository containing dir that were committed at or after since, and groups
// them into ChangeSets. Renames are reported as a deletion and an addition.
func Collect(ctx context.Context, dir string, since time.Time) (Repository, error) {
	top, err := git(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return Repository{}, err
	}
	out, err := git(ctx, dir,
		"-c", "core.quotePath=false",
		"log", "--no-merges", "--no-renames", "--no-color",
		// --raw -t also reports tree entries, which reveal created and removed directories.
		"--raw", "-t",
		"--since="+since.Format(time.RFC3339),
		"--format="+logFormat,
	)
	if err != nil {
		return Repository{}, err
	}
	commits, err := parseLog(out)
	if err != nil {
		return Repository{}, err
	}
	return Repository{Name: filepath.Base(strings.TrimSpace(top)), ChangeSets: group(commits)}, nil
}

func git(ctx context.Context, dir string, args ...string) (string, error) {
	//nolint:gosec // runs git only, with arguments built by this package and the repository directory chosen by the user.
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

// logFormat starts every commit with a record separator followed by
// unit-separated hash, author, author date, and subject. The raw diff lines of
// the commit follow on separate lines.
const (
	recordSeparator = "\x1e"
	fieldSeparator  = "\x1f"
	logFormat       = "%x1e%H%x1f%an%x1f%aI%x1f%s"
	treeMode        = "040000"
)

// loggedCommit is one parsed commit before grouping.
type loggedCommit struct {
	Commit
	files       []Change
	directories []Change
}

func parseLog(out string) ([]loggedCommit, error) {
	var commits []loggedCommit
	for _, record := range strings.Split(out, recordSeparator)[1:] {
		lines := strings.Split(record, "\n")
		c, err := parseHeader(lines[0])
		if err != nil {
			return nil, err
		}
		for _, line := range lines[1:] {
			if line == "" {
				continue
			}
			if err := c.addRaw(line); err != nil {
				return nil, fmt.Errorf("commit %s: %w", c.Hash, err)
			}
		}
		commits = append(commits, c)
	}
	return commits, nil
}

func parseHeader(line string) (loggedCommit, error) {
	fields := strings.Split(line, fieldSeparator)
	if len(fields) != 4 {
		return loggedCommit{}, fmt.Errorf("unexpected git log header %q", line)
	}
	t, err := time.Parse(time.RFC3339, fields[2])
	if err != nil {
		return loggedCommit{}, fmt.Errorf("commit %s: author date: %w", fields[0], err)
	}
	return loggedCommit{Hash: fields[0], Author: fields[1], Time: t, Subject: fields[3]}, nil
}

// addRaw records one line of git --raw output, for example
// ":100644 100644 533790e 485540d M\tpayments/api/openapi.yaml".
// Modified directories carry no information beyond their files and are skipped.
func (c *loggedCommit) addRaw(line string) error {
	meta, p, ok := strings.Cut(line, "\t")
	fields := strings.Fields(strings.TrimPrefix(meta, ":"))
	if !ok || len(fields) != 5 {
		return fmt.Errorf("unexpected raw diff line %q", line)
	}
	oldMode, newMode, code := fields[0], fields[1], fields[4]
	var status Status
	switch code {
	case "A":
		status = Added
	case "M", "T":
		status = Modified
	case "D":
		status = Deleted
	default:
		return fmt.Errorf("unexpected status %q for %s", code, p)
	}
	change := Change{Commit: c.Hash, Path: p, Status: status}
	switch {
	case oldMode != treeMode && newMode != treeMode:
		c.files = append(c.files, change)
	case status != Modified:
		c.directories = append(c.directories, change)
	}
	return nil
}

var workItemKey = regexp.MustCompile(`^\[?([A-Z][A-Z0-9]+-\d+)\b`)

func group(commits []loggedCommit) []ChangeSet {
	var sets []ChangeSet
	index := map[string]int{}
	for i := range commits {
		c := &commits[i]
		id := changeSetID(c.Commit)
		n, ok := index[id]
		if !ok {
			n = len(sets)
			index[id] = n
			sets = append(sets, ChangeSet{ID: id})
		}
		cs := &sets[n]
		cs.Commits = append(cs.Commits, c.Commit)
		cs.Files = append(cs.Files, c.files...)
		cs.Directories = append(cs.Directories, c.directories...)
	}
	return sets
}

func changeSetID(c Commit) string {
	if m := workItemKey.FindStringSubmatch(c.Subject); m != nil {
		return m[1]
	}
	return c.Hash[:12]
}
