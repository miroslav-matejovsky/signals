package attention_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/miroslav-matejovsky/signals/attention"
	"github.com/miroslav-matejovsky/signals/collection"
	"github.com/miroslav-matejovsky/signals/detection"
)

func changeSet(id string, day int) collection.ChangeSet {
	return collection.ChangeSet{
		ID:      id,
		Commits: []collection.Commit{{Hash: id + "-hash", Author: "Jane Doe", Time: time.Date(2026, 10, day, 0, 0, 0, 0, time.UTC), Subject: id}},
	}
}

func signal(changeSet string, t detection.Type) detection.Signal {
	return detection.Signal{
		Type:      t,
		ChangeSet: changeSet,
		Summary:   string(t),
		Evidence:  []detection.Evidence{{Commit: changeSet + "-hash", Path: "payments/api/openapi.yaml", Status: collection.Modified, Reason: "test"}},
	}
}

func TestSelectSuppressesChangeSetsWithoutSignals(t *testing.T) {
	sets := []collection.ChangeSet{changeSet("PAY-1", 20), changeSet("PAY-2", 21)}

	candidates, err := attention.Select(sets, []detection.Signal{signal("PAY-2", detection.DeploymentChanged)})
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, "PAY-2", candidates[0].ChangeSet)

	candidates, err = attention.Select(sets, nil)
	require.NoError(t, err)
	require.Empty(t, candidates)
}

func TestSelectLevels(t *testing.T) {
	tests := []struct {
		name  string
		types []detection.Type
		want  attention.Level
	}{
		{name: "one signal", types: []detection.Type{detection.PublicContractChanged}, want: attention.Medium},
		{name: "two distinct signals", types: []detection.Type{detection.CrossContextChange, detection.PublicContractChanged}, want: attention.High},
		{name: "same signal twice", types: []detection.Type{detection.SecurityChanged, detection.SecurityChanged}, want: attention.Medium},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var signals []detection.Signal
			for _, st := range tt.types {
				signals = append(signals, signal("PAY-1", st))
			}
			candidates, err := attention.Select([]collection.ChangeSet{changeSet("PAY-1", 20)}, signals)
			require.NoError(t, err)
			require.Len(t, candidates, 1)
			require.Equal(t, tt.want, candidates[0].Level)
		})
	}
}

func TestSelectPreservesProvenance(t *testing.T) {
	cs := changeSet("PAY-1421", 20)
	signals := []detection.Signal{
		signal("PAY-1421", detection.CrossContextChange),
		signal("PAY-1421", detection.PublicContractChanged),
	}

	candidates, err := attention.Select([]collection.ChangeSet{cs}, signals)
	require.NoError(t, err)
	require.Equal(t, []attention.Candidate{{
		Level:     attention.High,
		ChangeSet: "PAY-1421",
		Commits:   cs.Commits,
		Signals:   signals,
	}}, candidates)
}

func TestSelectOrder(t *testing.T) {
	sets := []collection.ChangeSet{
		changeSet("OLD-MEDIUM", 18),
		changeSet("NEW-MEDIUM", 22),
		changeSet("OLD-HIGH", 19),
		changeSet("NEW-HIGH", 21),
		changeSet("B-SAME-DAY", 20),
		changeSet("A-SAME-DAY", 20),
	}
	signals := []detection.Signal{
		signal("OLD-MEDIUM", detection.DeploymentChanged),
		signal("NEW-MEDIUM", detection.DeploymentChanged),
		signal("OLD-HIGH", detection.DeploymentChanged),
		signal("OLD-HIGH", detection.SecurityChanged),
		signal("NEW-HIGH", detection.DeploymentChanged),
		signal("NEW-HIGH", detection.SecurityChanged),
		signal("B-SAME-DAY", detection.DeploymentChanged),
		signal("A-SAME-DAY", detection.DeploymentChanged),
	}

	candidates, err := attention.Select(sets, signals)
	require.NoError(t, err)
	var ids []string
	for _, c := range candidates {
		ids = append(ids, c.ChangeSet)
	}
	require.Equal(t, []string{"NEW-HIGH", "OLD-HIGH", "NEW-MEDIUM", "A-SAME-DAY", "B-SAME-DAY", "OLD-MEDIUM"}, ids)
}

func TestSelectRejectsSignalsOfUnknownChangeSets(t *testing.T) {
	_, err := attention.Select([]collection.ChangeSet{changeSet("PAY-1", 20)}, []detection.Signal{signal("PAY-9", detection.SecurityChanged)})
	require.ErrorContains(t, err, "signals refer to unknown change sets: PAY-9")
}
