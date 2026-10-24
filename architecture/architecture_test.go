package architecture_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/miroslav-matejovsky/signals/architecture"
)

const declaration = `
systems:
  commerce:
    contexts:
      payments:
        paths: ["payments/**"]
        components:
          payments-api:
            path: payments/api
      customer:
        paths: ["customer/**"]
  platform:
    contexts:
      identity:
        paths: ["identity/**"]
artifacts:
  public-contract: ["**/openapi.yaml", "**/*.proto"]
  persistence-schema: ["**/migrations/**"]
  deployment: ["**/helm/**", "**/*.tf"]
  security: ["**/rbac.yaml", "**/helm/**/rbac/**"]
`

func mustParse(t *testing.T, yaml string) *architecture.Architecture {
	t.Helper()
	a, err := architecture.Parse([]byte(yaml))
	require.NoError(t, err)
	return a
}

func TestResolve(t *testing.T) {
	a := mustParse(t, declaration)
	payments := architecture.Scope{System: "commerce", Context: "payments"}
	paymentsAPI := payments
	paymentsAPI.Component = architecture.Component{Name: "payments-api", Path: "payments/api"}

	tests := []struct {
		name   string
		path   string
		want   architecture.Scope
		wantOK bool
	}{
		{name: "path in context", path: "payments/service.go", want: payments, wantOK: true},
		{name: "context directory itself", path: "payments", want: payments, wantOK: true},
		{name: "component file", path: "payments/api/openapi.yaml", want: paymentsAPI, wantOK: true},
		{name: "component directory", path: "payments/api", want: paymentsAPI, wantOK: true},
		{name: "component name prefix is not the component", path: "payments/apiary/x.go", want: payments, wantOK: true},
		{name: "context of another system", path: "identity/login.go", want: architecture.Scope{System: "platform", Context: "identity"}, wantOK: true},
		{name: "undeclared path", path: "docs/readme.md"},
		{name: "glob does not match prefix of segment", path: "paymentsx/a.go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := a.Resolve(tt.path)
			require.NoError(t, err)
			require.Equal(t, tt.wantOK, ok)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestResolveRejectsAmbiguousDeclaration(t *testing.T) {
	a := mustParse(t, `
systems:
  commerce:
    contexts:
      payments:
        paths: ["payments/**"]
      refunds:
        paths: ["payments/refunds/**"]
artifacts: {public-contract: [], persistence-schema: [], deployment: [], security: []}
`)
	_, _, err := a.Resolve("payments/refunds/r.go")
	require.ErrorContains(t, err, "maps to several contexts: commerce/payments, commerce/refunds")

	_, ok, err := a.Resolve("payments/p.go")
	require.NoError(t, err)
	require.True(t, ok)
}

func TestScopeString(t *testing.T) {
	s := architecture.Scope{System: "commerce", Context: "payments"}
	require.Equal(t, "commerce/payments", s.String())
	s.Component = architecture.Component{Name: "payments-api", Path: "payments/api"}
	require.Equal(t, "commerce/payments/payments-api", s.String())
}

func TestClassify(t *testing.T) {
	a := mustParse(t, declaration)
	tests := []struct {
		path string
		want []architecture.Artifact
	}{
		{path: "payments/api/openapi.yaml", want: []architecture.Artifact{{Kind: architecture.PublicContract, Pattern: "**/openapi.yaml"}}},
		{path: "openapi.yaml", want: []architecture.Artifact{{Kind: architecture.PublicContract, Pattern: "**/openapi.yaml"}}},
		{path: "proto/payments/v1/refund.proto", want: []architecture.Artifact{{Kind: architecture.PublicContract, Pattern: "**/*.proto"}}},
		{path: "payments/migrations/001_init.sql", want: []architecture.Artifact{{Kind: architecture.PersistenceSchema, Pattern: "**/migrations/**"}}},
		{path: "infra/main.tf", want: []architecture.Artifact{{Kind: architecture.Deployment, Pattern: "**/*.tf"}}},
		{
			// One artifact per kind, first matching pattern of each kind.
			path: "deploy/helm/rbac/rbac.yaml",
			want: []architecture.Artifact{
				{Kind: architecture.Deployment, Pattern: "**/helm/**"},
				{Kind: architecture.Security, Pattern: "**/rbac.yaml"},
			},
		},
		{path: "payments/service.go"},
		{path: "payments/openapi.yaml.bak"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			require.Equal(t, tt.want, a.Classify(tt.path))
		})
	}
}

func TestParseRejectsInvalidDeclaration(t *testing.T) {
	const artifacts = "artifacts: {public-contract: [], persistence-schema: [], deployment: [], security: []}\n"
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "unknown field",
			yaml:    "systems: {s: {contexts: {c: {repositories: [r]}}}}\n" + artifacts,
			wantErr: "field repositories not found",
		},
		{
			name:    "no systems",
			yaml:    artifacts,
			wantErr: "no systems declared",
		},
		{
			name:    "system without contexts",
			yaml:    "systems: {s: {}}\n" + artifacts,
			wantErr: "system s: no contexts declared",
		},
		{
			name:    "context without paths",
			yaml:    "systems: {s: {contexts: {c: {}}}}\n" + artifacts,
			wantErr: "context s/c: no paths declared",
		},
		{
			name:    "malformed path pattern",
			yaml:    "systems: {s: {contexts: {c: {paths: ['a/[b']}}}}\n" + artifacts,
			wantErr: `context s/c: pattern "a/[b": syntax error in pattern`,
		},
		{
			name:    "component path not clean",
			yaml:    "systems: {s: {contexts: {c: {paths: ['a/**'], components: {x: {path: a/x/}}}}}}\n" + artifacts,
			wantErr: `component x: path "a/x/" is not a clean relative directory`,
		},
		{
			name:    "component path outside repository",
			yaml:    "systems: {s: {contexts: {c: {paths: ['a/**'], components: {x: {path: ../x}}}}}}\n" + artifacts,
			wantErr: `component x: path "../x" is not a clean relative directory`,
		},
		{
			name:    "nested components",
			yaml:    "systems: {s: {contexts: {c: {paths: ['a/**'], components: {x: {path: a/x}, y: {path: a/x/y}}}}}}\n" + artifacts,
			wantErr: `component y: path "a/x/y" overlaps component x`,
		},
		{
			name:    "unknown artifact kind",
			yaml:    "systems: {s: {contexts: {c: {paths: ['a/**']}}}}\nartifacts: {public-contract: [], persistence-schema: [], deployment: [], security: [], docs: []}\n",
			wantErr: `artifacts: unknown kind "docs"`,
		},
		{
			name:    "missing artifact kind",
			yaml:    "systems: {s: {contexts: {c: {paths: ['a/**']}}}}\nartifacts: {public-contract: [], persistence-schema: [], deployment: []}\n",
			wantErr: "artifacts: kind security not declared",
		},
		{
			name:    "malformed artifact pattern",
			yaml:    "systems: {s: {contexts: {c: {paths: ['a/**']}}}}\nartifacts: {public-contract: ['**/[x'], persistence-schema: [], deployment: [], security: []}\n",
			wantErr: `artifacts public-contract: pattern "**/[x": syntax error in pattern`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := architecture.Parse([]byte(tt.yaml))
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestLoad(t *testing.T) {
	file := filepath.Join(t.TempDir(), "architecture.yaml")
	require.NoError(t, os.WriteFile(file, []byte(declaration), 0o600))

	a, err := architecture.Load(file)
	require.NoError(t, err)
	_, ok, err := a.Resolve("customer/profile.go")
	require.NoError(t, err)
	require.True(t, ok)

	_, err = architecture.Load(filepath.Join(t.TempDir(), "missing.yaml"))
	require.ErrorContains(t, err, "read architecture")
}
