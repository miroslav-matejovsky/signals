package attention

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/miroslav-matejovsky/signals/collection"
	"github.com/miroslav-matejovsky/signals/detection"
)

// Level is the deterministic attention priority of a Candidate.
type Level string

// Attention levels. There is no level for ChangeSets without signals: they
// are suppressed, and their architectural impact is unknown.
const (
	// High: the ChangeSet carries two or more distinct signal types.
	High Level = "HIGH"
	// Medium: the ChangeSet carries exactly one signal type.
	Medium Level = "MEDIUM"
)

// Candidate is a ChangeSet that deserves architect attention, together with
// the signals that caused it and, through them, their evidence.
type Candidate struct {
	Level     Level               `json:"level"`
	ChangeSet string              `json:"changeSet"`
	Commits   []collection.Commit `json:"commits"`
	Signals   []detection.Signal  `json:"signals"`
}

// Select aggregates signals per ChangeSet and returns one Candidate for every
// ChangeSet with at least one signal. ChangeSets without signals are
// suppressed. Candidates are ordered HIGH first, then by newest commit, newest
// first, then by ChangeSet ID. A signal referring to a ChangeSet not in
// changeSets is an error.
func Select(changeSets []collection.ChangeSet, signals []detection.Signal) ([]Candidate, error) {
	byChangeSet := map[string][]detection.Signal{}
	for _, s := range signals {
		byChangeSet[s.ChangeSet] = append(byChangeSet[s.ChangeSet], s)
	}
	var candidates []Candidate
	for i := range changeSets {
		cs := &changeSets[i]
		found, ok := byChangeSet[cs.ID]
		if !ok {
			continue
		}
		delete(byChangeSet, cs.ID)
		candidates = append(candidates, Candidate{Level: level(found), ChangeSet: cs.ID, Commits: cs.Commits, Signals: found})
	}
	if len(byChangeSet) > 0 {
		return nil, fmt.Errorf("signals refer to unknown change sets: %s", strings.Join(slices.Sorted(maps.Keys(byChangeSet)), ", "))
	}
	slices.SortStableFunc(candidates, func(a, b Candidate) int {
		if a.Level != b.Level {
			if a.Level == High {
				return -1
			}
			return 1
		}
		if c := newest(b.Commits).Compare(newest(a.Commits)); c != 0 {
			return c
		}
		return strings.Compare(a.ChangeSet, b.ChangeSet)
	})
	return candidates, nil
}

func level(signals []detection.Signal) Level {
	types := map[detection.Type]bool{}
	for _, s := range signals {
		types[s.Type] = true
	}
	if len(types) >= 2 {
		return High
	}
	return Medium
}

func newest(commits []collection.Commit) time.Time {
	var t time.Time
	for _, c := range commits {
		if c.Time.After(t) {
			t = c.Time
		}
	}
	return t
}
