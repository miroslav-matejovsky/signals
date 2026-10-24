package detection_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/miroslav-matejovsky/signals/architecture"
	"github.com/miroslav-matejovsky/signals/collection"
	"github.com/miroslav-matejovsky/signals/detection"
)

const declaration = `
systems:
  commerce:
    contexts:
      payments:
        paths: ["payments/**"]
        components:
          fraud-detection:
            path: payments/fraud
      customer:
        paths: ["customer/**"]
artifacts:
  public-contract: ["**/openapi.yaml"]
  persistence-schema: ["**/migrations/**"]
  deployment: ["**/helm/**", "**/*.tf"]
  security: ["**/rbac.yaml"]
`

func detect(t *testing.T, sets ...collection.ChangeSet) []detection.Signal {
	t.Helper()
	arch, err := architecture.Parse([]byte(declaration))
	require.NoError(t, err)
	signals, err := detection.Detect(collection.Repository{Name: "shop", ChangeSets: sets}, arch)
	require.NoError(t, err)
	return signals
}

func files(changes ...collection.Change) collection.ChangeSet {
	return collection.ChangeSet{ID: "PAY-1", Files: changes}
}

func modified(p string) collection.Change {
	return collection.Change{Commit: "c1", Path: p, Status: collection.Modified}
}

func TestDetectRules(t *testing.T) {
	tests := []struct {
		name string
		cs   collection.ChangeSet
		want []detection.Signal
	}{
		{
			name: "cross context change",
			cs:   files(modified("payments/service.go"), modified("docs/readme.md"), modified("customer/profile.go")),
			want: []detection.Signal{{
				Type:      detection.CrossContextChange,
				ChangeSet: "PAY-1",
				Summary:   "bounded contexts changed together: commerce/customer, commerce/payments",
				Evidence: []detection.Evidence{
					{Commit: "c1", Path: "payments/service.go", Status: collection.Modified, Reason: "in commerce/payments"},
					{Commit: "c1", Path: "customer/profile.go", Status: collection.Modified, Reason: "in commerce/customer"},
				},
			}},
		},
		{
			name: "public contract changed",
			cs:   files(modified("payments/api/openapi.yaml")),
			want: []detection.Signal{{
				Type:      detection.PublicContractChanged,
				ChangeSet: "PAY-1",
				Summary:   "public-contract files changed: 1",
				Evidence:  []detection.Evidence{{Commit: "c1", Path: "payments/api/openapi.yaml", Status: collection.Modified, Reason: `matches public-contract pattern "**/openapi.yaml"`}},
			}},
		},
		{
			name: "persistence schema changed",
			cs: files(
				collection.Change{Commit: "c1", Path: "payments/migrations/002.sql", Status: collection.Added},
				collection.Change{Commit: "c2", Path: "payments/migrations/002.sql", Status: collection.Modified},
			),
			want: []detection.Signal{{
				Type:      detection.PersistenceSchemaChanged,
				ChangeSet: "PAY-1",
				Summary:   "persistence-schema files changed: 1",
				Evidence: []detection.Evidence{
					{Commit: "c1", Path: "payments/migrations/002.sql", Status: collection.Added, Reason: `matches persistence-schema pattern "**/migrations/**"`},
					{Commit: "c2", Path: "payments/migrations/002.sql", Status: collection.Modified, Reason: `matches persistence-schema pattern "**/migrations/**"`},
				},
			}},
		},
		{
			name: "deployment changed",
			cs:   files(collection.Change{Commit: "c1", Path: "infra/main.tf", Status: collection.Deleted}),
			want: []detection.Signal{{
				Type:      detection.DeploymentChanged,
				ChangeSet: "PAY-1",
				Summary:   "deployment files changed: 1",
				Evidence:  []detection.Evidence{{Commit: "c1", Path: "infra/main.tf", Status: collection.Deleted, Reason: `matches deployment pattern "**/*.tf"`}},
			}},
		},
		{
			name: "security changed",
			cs:   files(modified("config/rbac.yaml")),
			want: []detection.Signal{{
				Type:      detection.SecurityChanged,
				ChangeSet: "PAY-1",
				Summary:   "security files changed: 1",
				Evidence:  []detection.Evidence{{Commit: "c1", Path: "config/rbac.yaml", Status: collection.Modified, Reason: `matches security pattern "**/rbac.yaml"`}},
			}},
		},
		{
			name: "component created and removed",
			cs: collection.ChangeSet{ID: "PAY-1", Directories: []collection.Change{
				{Commit: "c1", Path: "payments/fraud", Status: collection.Added},
				{Commit: "c1", Path: "payments/fraud/rules", Status: collection.Added},
				{Commit: "c2", Path: "payments/fraud", Status: collection.Deleted},
			}},
			want: []detection.Signal{{
				Type:      detection.ComponentCreatedOrRemoved,
				ChangeSet: "PAY-1",
				Summary:   "component commerce/payments/fraud-detection created; component commerce/payments/fraud-detection removed",
				Evidence: []detection.Evidence{
					{Commit: "c1", Path: "payments/fraud", Status: collection.Added, Reason: "declared path of component commerce/payments/fraud-detection"},
					{Commit: "c2", Path: "payments/fraud", Status: collection.Deleted, Reason: "declared path of component commerce/payments/fraud-detection"},
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, detect(t, tt.cs))
		})
	}
}

func TestDetectIgnoresImplementationOnlyChanges(t *testing.T) {
	tests := []struct {
		name string
		cs   collection.ChangeSet
	}{
		{name: "files in one context", cs: files(modified("payments/service.go"), modified("payments/fraud/rules.go"))},
		{name: "undeclared files", cs: files(modified("docs/readme.md"), modified("tools/lint.go"))},
		{name: "one context and undeclared files", cs: files(modified("customer/profile.go"), modified("README.md"))},
		{
			name: "directory inside an existing component",
			cs:   collection.ChangeSet{ID: "PAY-1", Directories: []collection.Change{{Commit: "c1", Path: "payments/fraud/rules", Status: collection.Added}}},
		},
		{
			name: "directory that is not a component",
			cs:   collection.ChangeSet{ID: "PAY-1", Directories: []collection.Change{{Commit: "c1", Path: "payments/refunds", Status: collection.Added}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Empty(t, detect(t, tt.cs))
		})
	}
}

func TestDetectCombinedSignalsAcrossContexts(t *testing.T) {
	signals := detect(t,
		collection.ChangeSet{
			ID: "PAY-1421",
			Files: []collection.Change{
				{Commit: "c1", Path: "payments/api/openapi.yaml", Status: collection.Modified},
				{Commit: "c1", Path: "customer/profile.go", Status: collection.Modified},
				{Commit: "c2", Path: "payments/migrations/002_refunds.sql", Status: collection.Added},
				{Commit: "c2", Path: "payments/fraud/rules.go", Status: collection.Added},
			},
			Directories: []collection.Change{{Commit: "c2", Path: "payments/fraud", Status: collection.Added}},
		},
		files(modified("payments/service.go")),
	)

	var types []detection.Type
	for _, s := range signals {
		require.Equal(t, "PAY-1421", s.ChangeSet)
		require.NotEmpty(t, s.Evidence)
		types = append(types, s.Type)
	}
	require.Equal(t, []detection.Type{
		detection.CrossContextChange,
		detection.PublicContractChanged,
		detection.PersistenceSchemaChanged,
		detection.ComponentCreatedOrRemoved,
	}, types)
	require.Len(t, signals[0].Evidence, 4)
}

func TestDetectFailsOnAmbiguousArchitecture(t *testing.T) {
	arch, err := architecture.Parse([]byte(`
systems:
  commerce:
    contexts:
      payments: {paths: ["payments/**"]}
      refunds: {paths: ["payments/refunds/**"]}
artifacts: {public-contract: [], persistence-schema: [], deployment: [], security: []}
`))
	require.NoError(t, err)

	_, err = detection.Detect(collection.Repository{Name: "shop", ChangeSets: []collection.ChangeSet{files(modified("payments/refunds/r.go"))}}, arch)
	require.ErrorContains(t, err, "change set PAY-1: payments/refunds/r.go maps to several contexts")
}
