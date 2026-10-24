package architecture

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// ArtifactKind classifies a path as an architecturally significant artifact.
type ArtifactKind string

// Artifact kinds accepted in the artifacts section of an architecture file.
const (
	PublicContract    ArtifactKind = "public-contract"
	PersistenceSchema ArtifactKind = "persistence-schema"
	Deployment        ArtifactKind = "deployment"
	Security          ArtifactKind = "security"
)

// artifactKinds lists every known kind. Each must be declared in an
// architecture file, so Classify reports kinds in this order.
var artifactKinds = []ArtifactKind{PublicContract, PersistenceSchema, Deployment, Security}

// Artifact records that a path is an architectural artifact of Kind because it
// matches the declared Pattern.
type Artifact struct {
	Kind    ArtifactKind
	Pattern string
}

// Component is a declared component of a bounded context. It owns the
// directory Path and everything below it.
type Component struct {
	Name string
	Path string
}

// Scope is the declared architectural location of a path. Component is the
// zero value when the path is outside every component of its context.
type Scope struct {
	System    string
	Context   string
	Component Component
}

// String returns the scope as system/context or system/context/component.
func (s Scope) String() string {
	if s.Component.Name == "" {
		return s.System + "/" + s.Context
	}
	return s.System + "/" + s.Context + "/" + s.Component.Name
}

// Architecture is a validated architecture declaration.
type Architecture struct {
	// contexts are sorted by system name, then context name.
	contexts []boundedContext
	// artifacts are sorted by kind in artifactKinds order, then declaration order.
	artifacts []Artifact
}

type boundedContext struct {
	system string
	name   string
	paths  []string
	// components are sorted by name and never nest.
	components []Component
}

// Load reads and validates the architecture file at filename.
func Load(filename string) (*Architecture, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read architecture: %w", err)
	}
	a, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("architecture %s: %w", filename, err)
	}
	return a, nil
}

// document is the YAML layout of an architecture file.
type document struct {
	Systems   map[string]systemDocument `yaml:"systems"`
	Artifacts map[ArtifactKind][]string `yaml:"artifacts"`
}

type systemDocument struct {
	Contexts map[string]contextDocument `yaml:"contexts"`
}

type contextDocument struct {
	Paths      []string                     `yaml:"paths"`
	Components map[string]componentDocument `yaml:"components"`
}

type componentDocument struct {
	Path string `yaml:"path"`
}

// Parse decodes and validates an architecture declaration.
//
// It rejects unknown fields, systems without contexts, contexts without paths,
// invalid patterns, component paths that are not clean
// relative directories or that nest, unknown artifact kinds, and missing
// artifact kinds. Every artifact kind must be declared explicitly; an empty
// list declares that no path is an artifact of that kind.
func Parse(data []byte) (*Architecture, error) {
	var doc document
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}
	contexts, err := parseContexts(doc.Systems)
	if err != nil {
		return nil, err
	}
	artifacts, err := parseArtifacts(doc.Artifacts)
	if err != nil {
		return nil, err
	}
	return &Architecture{contexts: contexts, artifacts: artifacts}, nil
}

func parseContexts(systems map[string]systemDocument) ([]boundedContext, error) {
	if len(systems) == 0 {
		return nil, errors.New("no systems declared")
	}
	var contexts []boundedContext
	for _, system := range slices.Sorted(maps.Keys(systems)) {
		declared := systems[system].Contexts
		if len(declared) == 0 {
			return nil, fmt.Errorf("system %s: no contexts declared", system)
		}
		for _, name := range slices.Sorted(maps.Keys(declared)) {
			bc, err := parseContext(system, name, declared[name])
			if err != nil {
				return nil, fmt.Errorf("context %s/%s: %w", system, name, err)
			}
			contexts = append(contexts, bc)
		}
	}
	return contexts, nil
}

func parseContext(system, name string, doc contextDocument) (boundedContext, error) {
	if len(doc.Paths) == 0 {
		return boundedContext{}, errors.New("no paths declared")
	}
	if err := validatePatterns(doc.Paths); err != nil {
		return boundedContext{}, err
	}
	bc := boundedContext{system: system, name: name, paths: doc.Paths}
	for _, cname := range slices.Sorted(maps.Keys(doc.Components)) {
		p := doc.Components[cname].Path
		if !cleanRelativeDir(p) {
			return boundedContext{}, fmt.Errorf("component %s: path %q is not a clean relative directory", cname, p)
		}
		for _, other := range bc.components {
			if within(p, other.Path) || within(other.Path, p) {
				return boundedContext{}, fmt.Errorf("component %s: path %q overlaps component %s", cname, p, other.Name)
			}
		}
		bc.components = append(bc.components, Component{Name: cname, Path: p})
	}
	return bc, nil
}

func parseArtifacts(declared map[ArtifactKind][]string) ([]Artifact, error) {
	for _, kind := range slices.Sorted(maps.Keys(declared)) {
		if !slices.Contains(artifactKinds, kind) {
			return nil, fmt.Errorf("artifacts: unknown kind %q", kind)
		}
	}
	var artifacts []Artifact
	for _, kind := range artifactKinds {
		patterns, ok := declared[kind]
		if !ok {
			return nil, fmt.Errorf("artifacts: kind %s not declared, declare an empty list to detect none", kind)
		}
		if err := validatePatterns(patterns); err != nil {
			return nil, fmt.Errorf("artifacts %s: %w", kind, err)
		}
		for _, p := range patterns {
			artifacts = append(artifacts, Artifact{Kind: kind, Pattern: p})
		}
	}
	return artifacts, nil
}

func cleanRelativeDir(p string) bool {
	return p != "" && p != "." && p != ".." && path.Clean(p) == p && !path.IsAbs(p) && !strings.HasPrefix(p, "../")
}

// within reports whether p is dir or a path below dir.
func within(p, dir string) bool {
	return p == dir || strings.HasPrefix(p, dir+"/")
}

// Resolve maps a slash-separated path, relative to the repository root, to its
// declared scope.
//
// A path belongs to the context whose paths match it. ok is false when the
// path is outside the declared architecture. A path that belongs to more than
// one context is an ambiguous declaration and returns an error.
func (a *Architecture) Resolve(p string) (scope Scope, ok bool, err error) {
	bc, ok, err := a.context(p)
	if err != nil || !ok {
		return Scope{}, false, err
	}
	scope = Scope{System: bc.system, Context: bc.name}
	for _, c := range bc.components {
		if within(p, c.Path) {
			scope.Component = c
			break
		}
	}
	return scope, true, nil
}

func (a *Architecture) context(p string) (boundedContext, bool, error) {
	var matches []boundedContext
	for _, bc := range a.contexts {
		if slices.ContainsFunc(bc.paths, func(pattern string) bool { return match(pattern, p) }) {
			matches = append(matches, bc)
		}
	}
	switch len(matches) {
	case 0:
		return boundedContext{}, false, nil
	case 1:
		return matches[0], true, nil
	default:
		names := make([]string, len(matches))
		for i, bc := range matches {
			names[i] = bc.system + "/" + bc.name
		}
		return boundedContext{}, false, fmt.Errorf("%s maps to several contexts: %s", p, strings.Join(names, ", "))
	}
}

// Classify returns the artifacts a slash-separated path belongs to, at most
// one per kind. Each Artifact names the first declared pattern of its kind
// that matches the path.
func (a *Architecture) Classify(p string) []Artifact {
	var found []Artifact
	for _, art := range a.artifacts {
		if match(art.Pattern, p) && !slices.ContainsFunc(found, func(f Artifact) bool { return f.Kind == art.Kind }) {
			found = append(found, art)
		}
	}
	return found
}
