package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pedro-git-projects/gymtracker/model"
)

// legacyData is a data file as written before the bodyweight flag existed.
const legacyData = `{
  "Users": {
    "pedro": {
      "Name": "Pedro",
      "Programs": [{
        "ID": "85bb09ef7433",
        "Name": "Upper Lower",
        "Workouts": {"1": {"Target": 0, "Exercises": [
          {"Exercise": "Chest Dip", "Sets": 3, "Reps": 8,
           "RM": {"UpperBound": 10, "LowerBound": 8},
           "Load": 0, "Rest": 90000000000, "Notes": "https://youtu.be/x"}
        ]}}
      }],
      "ActiveProgram": "85bb09ef7433",
      "Current": null,
      "History": [{
        "ID": "e9aa44026813", "ProgramID": "85bb09ef7433", "Day": 1, "Target": 6,
        "StartedAt": "2026-11-09T17:57:52-03:00", "FinishedAt": "2026-11-09T18:40:00-03:00",
        "Exercises": [{
          "Planned": {"Exercise": "Chest Dip", "Sets": 3, "Reps": 8,
            "RM": {"UpperBound": 10, "LowerBound": 8}, "Load": 0, "Rest": 90000000000, "Notes": ""},
          "Sets": [{"Reps": 8, "Load": 0, "Done": true}, {"Reps": 7, "Load": 0, "Done": false}],
          "Notes": "felt good"
        }],
        "Notes": ""
      }]
    }
  }
}`

func TestLegacyDataSurvivesRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	if err := os.WriteFile(path, []byte(legacyData), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	// Any write rewrites the whole file in the current format.
	err = st.Update(func(d *Data) error {
		d.Users["pedro"].SetBodyweight("Chest Dip", true)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	st.View(func(d *Data) {
		u := d.Users["pedro"]
		if u == nil || u.Name != "Pedro" || u.ActiveProgram != "85bb09ef7433" {
			t.Fatalf("user lost: %+v", u)
		}
		wk, ok := u.Active().Workout(model.Monday)
		if !ok || wk.Target != model.UpperBody || wk.Target.String() != "Upper body" {
			t.Errorf("workout = %+v, %v", wk, ok)
		}
		if got := wk.Exercises[0].Notes; got != "https://youtu.be/x" {
			t.Errorf("notes = %q", got)
		}
		if len(u.History) != 1 {
			t.Fatalf("history has %d sessions", len(u.History))
		}
		s := u.History[0]
		if s.Target != model.LowerBody || !s.Finished() || len(s.Exercises[0].Sets) != 2 ||
			len(s.Exercises[0].CompletedSets()) != 1 || s.Exercises[0].Notes != "felt good" {
			t.Errorf("session changed: %+v", s)
		}
		if !u.IsBodyweight("Chest Dip") || u.IsBodyweight("Bench") {
			t.Errorf("bodyweight flags = %v", u.Bodyweight)
		}
	})
}

func TestExportMatchesDataFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	if err := os.WriteFile(path, []byte(legacyData), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Update(func(*Data) error { return nil }); err != nil {
		t.Fatal(err)
	}
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	exported, err := st.Export()
	if err != nil {
		t.Fatal(err)
	}
	if string(exported) != string(onDisk) {
		t.Error("export differs from the data file")
	}
}
