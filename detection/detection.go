package detection

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/miroslav-matejovsky/signals/architecture"
	"github.com/miroslav-matejovsky/signals/collection"
)

// Type identifies the rule that produced a Signal.
type Type string

// Signal types, one per detection rule.
const (
	// CrossContextChange: the ChangeSet changes files in two or more declared
	// bounded contexts.
	CrossContextChange Type = "CROSS_CONTEXT_CHANGE"
	// PublicContractChanged: the ChangeSet changes a public-contract artifact.
	PublicContractChanged Type = "PUBLIC_CONTRACT_CHANGED"
	// PersistenceSchemaChanged: the ChangeSet changes a persistence-schema artifact.
	PersistenceSchemaChanged Type = "PERSISTENCE_SCHEMA_CHANGED"
	// DeploymentChanged: the ChangeSet changes a deployment artifact.
	DeploymentChanged Type = "DEPLOYMENT_CHANGED"
	// SecurityChanged: the ChangeSet changes a security artifact.
	SecurityChanged Type = "SECURITY_CHANGED"
	// ComponentCreatedOrRemoved: the ChangeSet creates or removes the
	// directory declared as the path of a component.
	ComponentCreatedOrRemoved Type = "COMPONENT_CREATED_OR_REMOVED"
)

// Signal is a factual observation that a ChangeSet touched the declared
// architecture. It does not judge whether the change is correct.
type Signal struct {
	Type Type `json:"type"`
	// ChangeSet is the ID of the ChangeSet the signal was detected in.
	ChangeSet string `json:"changeSet"`
	// Summary states the observed fact in one line.
	Summary string `json:"summary"`
	// Evidence lists every change that satisfied the rule, in ChangeSet order.
	Evidence []Evidence `json:"evidence"`
}

// Evidence is one observed change that caused a Signal.
type Evidence struct {
	Commit string            `json:"commit"`
	Path   string            `json:"path"`
	Status collection.Status `json:"status"`
	// Reason names the declaration the change matched.
	Reason string `json:"reason"`
}

func newEvidence(c collection.Change, reason string) Evidence {
	return Evidence{Commit: c.Commit, Path: c.Path, Status: c.Status, Reason: reason}
}

// rule detects one signal type in a ChangeSet of the named repository. ok is
// false when the ChangeSet shows no evidence for the signal.
type rule func(repository string, cs *collection.ChangeSet, arch *architecture.Architecture) (s Signal, ok bool, err error)

// rules run in this order on every ChangeSet; signals keep the order.
var rules = []rule{
	crossContextChange,
	artifactChanged(architecture.PublicContract, PublicContractChanged),
	artifactChanged(architecture.PersistenceSchema, PersistenceSchemaChanged),
	artifactChanged(architecture.Deployment, DeploymentChanged),
	artifactChanged(architecture.Security, SecurityChanged),
	componentCreatedOrRemoved,
}

// Detect applies every rule to every ChangeSet of repo and returns the
// signals found, at most one per type and ChangeSet. It fails when a changed
// path maps ambiguously to the declared architecture.
func Detect(repo collection.Repository, arch *architecture.Architecture) ([]Signal, error) {
	var signals []Signal
	for i := range repo.ChangeSets {
		cs := &repo.ChangeSets[i]
		for _, r := range rules {
			s, ok, err := r(repo.Name, cs, arch)
			if err != nil {
				return nil, fmt.Errorf("change set %s: %w", cs.ID, err)
			}
			if ok {
				signals = append(signals, s)
			}
		}
	}
	return signals, nil
}

func crossContextChange(repository string, cs *collection.ChangeSet, arch *architecture.Architecture) (Signal, bool, error) {
	var evidence []Evidence
	contexts := map[string]bool{}
	for _, f := range cs.Files {
		scope, ok, err := arch.Resolve(repository, f.Path)
		if err != nil {
			return Signal{}, false, err
		}
		if !ok {
			continue
		}
		contexts[scope.System+"/"+scope.Context] = true
		evidence = append(evidence, newEvidence(f, "in "+scope.String()))
	}
	if len(contexts) < 2 {
		return Signal{}, false, nil
	}
	return Signal{
		Type:      CrossContextChange,
		ChangeSet: cs.ID,
		Summary:   "bounded contexts changed together: " + strings.Join(slices.Sorted(maps.Keys(contexts)), ", "),
		Evidence:  evidence,
	}, true, nil
}

func artifactChanged(kind architecture.ArtifactKind, signal Type) rule {
	return func(_ string, cs *collection.ChangeSet, arch *architecture.Architecture) (Signal, bool, error) {
		var evidence []Evidence
		paths := map[string]bool{}
		for _, f := range cs.Files {
			for _, a := range arch.Classify(f.Path) {
				if a.Kind == kind {
					paths[f.Path] = true
					evidence = append(evidence, newEvidence(f, fmt.Sprintf("matches %s pattern %q", kind, a.Pattern)))
				}
			}
		}
		if len(evidence) == 0 {
			return Signal{}, false, nil
		}
		return Signal{
			Type:      signal,
			ChangeSet: cs.ID,
			Summary:   fmt.Sprintf("%s files changed: %d", kind, len(paths)),
			Evidence:  evidence,
		}, true, nil
	}
}

func componentCreatedOrRemoved(repository string, cs *collection.ChangeSet, arch *architecture.Architecture) (Signal, bool, error) {
	var evidence []Evidence
	var facts []string
	for _, d := range cs.Directories {
		scope, ok, err := arch.Resolve(repository, d.Path)
		if err != nil {
			return Signal{}, false, err
		}
		if !ok || scope.Component.Path != d.Path {
			continue
		}
		action := "created"
		if d.Status == collection.Deleted {
			action = "removed"
		}
		evidence = append(evidence, newEvidence(d, "declared path of component "+scope.String()))
		facts = append(facts, "component "+scope.String()+" "+action)
	}
	if len(evidence) == 0 {
		return Signal{}, false, nil
	}
	return Signal{
		Type:      ComponentCreatedOrRemoved,
		ChangeSet: cs.ID,
		Summary:   strings.Join(facts, "; "),
		Evidence:  evidence,
	}, true, nil
}
