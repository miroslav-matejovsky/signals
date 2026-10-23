package collection_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/miroslav-matejovsky/signals/collection"
)

// fixture is a Git repository in a temporary directory named "commerce",
// isolated from the user's and the system's Git configuration.
type fixture struct {
	t      *testing.T
	dir    string
	writes int
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	config := filepath.Join(t.TempDir(), "gitconfig")
	require.NoError(t, os.WriteFile(config, nil, 0o600))
	t.Setenv("GIT_CONFIG_GLOBAL", config)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Jane Doe")
	t.Setenv("GIT_AUTHOR_EMAIL", "jane@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Jane Doe")
	t.Setenv("GIT_COMMITTER_EMAIL", "jane@example.com")

	f := &fixture{t: t, dir: filepath.Join(t.TempDir(), "commerce")}
	require.NoError(t, os.Mkdir(f.dir, 0o750))
	f.git(nil, "init", "--quiet")
	return f
}

// write creates or modifies the file at slash-separated path p.
func (f *fixture) write(p string) {
	f.t.Helper()
	f.writes++
	file := filepath.Join(f.dir, filepath.FromSlash(p))
	require.NoError(f.t, os.MkdirAll(filepath.Dir(file), 0o750))
	require.NoError(f.t, os.WriteFile(file, []byte(strconv.Itoa(f.writes)), 0o600))
}

func (f *fixture) remove(p string) {
	f.t.Helper()
	require.NoError(f.t, os.Remove(filepath.Join(f.dir, filepath.FromSlash(p))))
}

// commit commits all changes at the given RFC 3339 time and returns the commit.
func (f *fixture) commit(at, subject string) collection.Commit {
	f.t.Helper()
	env := []string{"GIT_AUTHOR_DATE=" + at, "GIT_COMMITTER_DATE=" + at}
	f.git(env, "add", "--all")
	f.git(env, "commit", "--quiet", "--message", subject)
	when, err := time.Parse(time.RFC3339, at)
	require.NoError(f.t, err)
	return collection.Commit{Hash: f.git(nil, "rev-parse", "HEAD"), Author: "Jane Doe", Time: when, Subject: subject}
}

func (f *fixture) git(env []string, args ...string) string {
	f.t.Helper()
	cmd := exec.CommandContext(f.t.Context(), "git", args...)
	cmd.Dir = f.dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	require.NoError(f.t, err, string(out))
	return strings.TrimSpace(string(out))
}

func TestCollect(t *testing.T) {
	f := newFixture(t)
	f.write("payments/api/openapi.yaml")
	f.write("payments/service.go")
	f.write("customer/profile.go")
	f.commit("2026-10-01T10:00:00Z", "initial structure")

	f.write("payments/api/openapi.yaml")
	f.write("customer/refunds.go")
	refunds := f.commit("2026-10-20T10:00:00Z", "PAY-1421 expose refunds")

	f.write("customer/profile.go")
	typo := f.commit("2026-10-21T10:00:00Z", "fix typo")

	f.write("payments/fraud/rules.go")
	f.remove("payments/service.go")
	fraud := f.commit("2026-10-22T10:00:00Z", "[PAY-1421] add fraud rules")

	f.remove("payments/fraud/rules.go")
	removal := f.commit("2026-10-23T10:00:00Z", "remove fraud rules")

	repo, err := collection.Collect(context.Background(), f.dir, time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	require.Equal(t, collection.Repository{
		Name: "commerce",
		ChangeSets: []collection.ChangeSet{
			{
				ID:          removal.Hash[:12],
				Commits:     []collection.Commit{removal},
				Files:       []collection.Change{{Commit: removal.Hash, Path: "payments/fraud/rules.go", Status: collection.Deleted}},
				Directories: []collection.Change{{Commit: removal.Hash, Path: "payments/fraud", Status: collection.Deleted}},
			},
			{
				ID:      "PAY-1421",
				Commits: []collection.Commit{fraud, refunds},
				Files: []collection.Change{
					{Commit: fraud.Hash, Path: "payments/fraud/rules.go", Status: collection.Added},
					{Commit: fraud.Hash, Path: "payments/service.go", Status: collection.Deleted},
					{Commit: refunds.Hash, Path: "customer/refunds.go", Status: collection.Added},
					{Commit: refunds.Hash, Path: "payments/api/openapi.yaml", Status: collection.Modified},
				},
				Directories: []collection.Change{{Commit: fraud.Hash, Path: "payments/fraud", Status: collection.Added}},
			},
			{
				ID:      typo.Hash[:12],
				Commits: []collection.Commit{typo},
				Files:   []collection.Change{{Commit: typo.Hash, Path: "customer/profile.go", Status: collection.Modified}},
			},
		},
	}, repo)
}

func TestCollectWorkItemKeys(t *testing.T) {
	f := newFixture(t)
	subjects := []string{
		"PAY-7 first",
		"[PAY-7] second",
		"PAY-7: third",
		"Fix UTF-8 handling",
		"PAY-7x is not a key",
	}
	var commits []collection.Commit
	for i, subject := range subjects {
		f.write("file.txt")
		commits = append(commits, f.commit(time.Date(2026, 10, 20, i, 0, 0, 0, time.UTC).Format(time.RFC3339), subject))
	}

	repo, err := collection.Collect(context.Background(), f.dir, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	var ids []string
	for _, cs := range repo.ChangeSets {
		ids = append(ids, cs.ID)
	}
	require.Equal(t, []string{commits[4].Hash[:12], commits[3].Hash[:12], "PAY-7"}, ids)
	require.Len(t, repo.ChangeSets[2].Commits, 3)
}

func TestCollectFailsOutsideRepository(t *testing.T) {
	_, err := collection.Collect(context.Background(), t.TempDir(), time.Now())
	require.ErrorContains(t, err, "git rev-parse --show-toplevel")
}
