package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var now = time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

// newRepository creates a Git repository named "commerce" whose history
// exercises the example architecture.yaml, isolated from the user's and the
// system's Git configuration. It returns the repository directory.
func newRepository(t *testing.T) string {
	t.Helper()
	config := filepath.Join(t.TempDir(), "gitconfig")
	require.NoError(t, os.WriteFile(config, nil, 0o600))
	t.Setenv("GIT_CONFIG_GLOBAL", config)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Jane Doe")
	t.Setenv("GIT_AUTHOR_EMAIL", "jane@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Jane Doe")
	t.Setenv("GIT_COMMITTER_EMAIL", "jane@example.com")

	dir := filepath.Join(t.TempDir(), "commerce")
	require.NoError(t, os.Mkdir(dir, 0o750))
	git := func(env []string, args ...string) {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), env...)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	writes := 0
	commit := func(at, subject string, paths ...string) {
		t.Helper()
		for _, p := range paths {
			writes++
			file := filepath.Join(dir, filepath.FromSlash(p))
			require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o750))
			require.NoError(t, os.WriteFile(file, []byte(strconv.Itoa(writes)), 0o600))
		}
		env := []string{"GIT_AUTHOR_DATE=" + at, "GIT_COMMITTER_DATE=" + at}
		git(env, "add", "--all")
		git(env, "commit", "--quiet", "--message", subject)
	}

	git(nil, "init", "--quiet")
	commit("2026-10-01T10:00:00Z", "initial structure",
		"payments/api/openapi.yaml", "payments/service.go", "customer/profile/profile.go", "deploy/helm/values.yaml")
	commit("2026-10-20T10:00:00Z", "PAY-1421 expose refund status in customer profile",
		"payments/api/openapi.yaml", "customer/profile/profile.go")
	commit("2026-10-21T10:00:00Z", "PAY-1421 add refunds table",
		"payments/migrations/002_refunds.sql")
	commit("2026-10-21T11:00:00Z", "CUST-7 tidy profile formatting",
		"customer/profile/profile.go")
	commit("2026-10-22T10:00:00Z", "add fraud detection rules",
		"payments/fraud/rules.go")
	return dir
}

func scan(t *testing.T, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	require.NoError(t, run(context.Background(), args, now, &out))
	return out.String()
}

func TestScanText(t *testing.T) {
	repo := newRepository(t)

	out := scan(t, "scan", "--repo", repo, "--architecture", "architecture.yaml", "--since", "7d")

	high := strings.Index(out, "HIGH  PAY-1421\n")
	medium := strings.Index(out, "MEDIUM  ")
	require.GreaterOrEqual(t, high, 0, out)
	require.Greater(t, medium, high, out)
	for _, want := range []string{
		"PAY-1421 expose refund status in customer profile\n",
		"PAY-1421 add refunds table\n",
		"  CROSS_CONTEXT_CHANGE          bounded contexts changed together: commerce/customer, commerce/payments\n",
		"  PUBLIC_CONTRACT_CHANGED       public-contract files changed: 1\n",
		"  PERSISTENCE_SCHEMA_CHANGED    persistence-schema files changed: 1\n",
		" payments/api/openapi.yaml modified (matches public-contract pattern \"**/openapi.yaml\")\n",
		" customer/profile/profile.go modified (in commerce/customer/customer-profile)\n",
		" payments/migrations/002_refunds.sql added (matches persistence-schema pattern \"**/migrations/**\")\n",
		"  COMPONENT_CREATED_OR_REMOVED  component commerce/payments/fraud-detection created\n",
		" payments/fraud added (declared path of component commerce/payments/fraud-detection)\n",
		"Repository commerce: 3 change sets since 2026-10-16T12:00:00Z.\n",
		"2 attention candidates. 1 change sets without architectural signal: impact unknown, not assessed as safe.\n",
	} {
		require.Contains(t, out, want)
	}
	require.NotContains(t, out, "CUST-7", "change set without signal is suppressed")
	require.NotContains(t, out, "initial structure", "commit before the scan period")
}

func TestScanJSON(t *testing.T) {
	repo := newRepository(t)

	out := scan(t, "scan", "--repo", repo, "--architecture", "architecture.yaml", "--since", "7d", "--format", "json")

	var got struct {
		Repository    string
		ChangeSets    int
		WithoutSignal int
		Candidates    []struct {
			Level     string
			ChangeSet string
			Commits   []struct{ Hash, Subject string }
			Signals   []struct {
				Type      string
				ChangeSet string
				Evidence  []struct{ Commit, Path, Status, Reason string }
			}
		}
	}
	require.NoError(t, json.Unmarshal([]byte(out), &got), out)
	require.Equal(t, "commerce", got.Repository)
	require.Equal(t, 3, got.ChangeSets)
	require.Equal(t, 1, got.WithoutSignal)
	require.Len(t, got.Candidates, 2)

	high := got.Candidates[0]
	require.Equal(t, "HIGH", high.Level)
	require.Equal(t, "PAY-1421", high.ChangeSet)
	require.Len(t, high.Commits, 2)
	require.Len(t, high.Signals, 3)
	contract := high.Signals[1]
	require.Equal(t, "PUBLIC_CONTRACT_CHANGED", contract.Type)
	require.Equal(t, "PAY-1421", contract.ChangeSet)
	require.Len(t, contract.Evidence, 1)
	require.Equal(t, high.Commits[1].Hash, contract.Evidence[0].Commit, "evidence refers to the commit that changed the contract")
	require.Equal(t, "payments/api/openapi.yaml", contract.Evidence[0].Path)
	require.Equal(t, "modified", contract.Evidence[0].Status)

	require.Equal(t, "MEDIUM", got.Candidates[1].Level)
}

func TestScanWithoutCandidates(t *testing.T) {
	repo := newRepository(t)

	out := scan(t, "scan", "--repo", repo, "--architecture", "architecture.yaml", "--since", "12h")
	require.Equal(t,
		"Repository commerce: 0 change sets since 2026-10-23T00:00:00Z.\n"+
			"0 attention candidates. 0 change sets without architectural signal: impact unknown, not assessed as safe.\n",
		out)

	out = scan(t, "scan", "--repo", repo, "--architecture", "architecture.yaml", "--since", "12h", "--format", "json")
	require.Contains(t, out, `"candidates": []`)
}

func TestRunRejectsInvalidArguments(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "no command", args: nil, wantErr: usage},
		{name: "unknown command", args: []string{"report"}, wantErr: usage},
		{name: "missing flags", args: []string{"scan", "--repo", "."}, wantErr: "--repo, --architecture, and --since are required"},
		{name: "unknown flag", args: []string{"scan", "--verbose"}, wantErr: "flag provided but not defined: -verbose"},
		{name: "positional argument", args: []string{"scan", "--repo", ".", "extra"}, wantErr: `unexpected arguments ["extra"]`},
		{name: "unknown format", args: []string{"scan", "--repo", ".", "--architecture", "a.yaml", "--since", "7d", "--format", "xml"}, wantErr: `unknown format "xml"`},
		{name: "invalid since", args: []string{"scan", "--repo", ".", "--architecture", "a.yaml", "--since", "week"}, wantErr: `invalid --since "week"`},
		{name: "missing architecture", args: []string{"scan", "--repo", ".", "--architecture", "missing.yaml", "--since", "7d"}, wantErr: "read architecture"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(context.Background(), tt.args, now, &bytes.Buffer{})
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestParseSince(t *testing.T) {
	tests := []struct {
		in      string
		want    time.Duration
		wantErr string
	}{
		{in: "7d", want: 7 * 24 * time.Hour},
		{in: "36h", want: 36 * time.Hour},
		{in: "90m", want: 90 * time.Minute},
		{in: "1.5d", wantErr: "invalid syntax"},
		{in: "0d", wantErr: "must be positive"},
		{in: "-2h", wantErr: "must be positive"},
		{in: "7", wantErr: "missing unit"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := parseSince(tt.in)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
