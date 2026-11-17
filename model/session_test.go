package model

import "testing"

func TestRemoveSet(t *testing.T) {
	s := Session{Exercises: []ExerciseLog{{
		Planned: WorkoutExercise{Exercise: "Row", Sets: 3},
		Sets:    []SetLog{{Reps: 8, Load: 40}, {Reps: 9, Load: 45}, {Reps: 10, Load: 50}},
	}}}
	if err := s.RemoveSet("Row", 1); err != nil {
		t.Fatal(err)
	}
	got := s.Exercises[0].Sets
	if len(got) != 2 || got[0].Load != 40 || got[1].Load != 50 {
		t.Errorf("sets = %+v", got)
	}
	for _, idx := range []int{-1, 2} {
		if err := s.RemoveSet("Row", idx); err == nil {
			t.Errorf("RemoveSet(%d) succeeded", idx)
		}
	}
	if err := s.RemoveSet("Curl", 0); err == nil {
		t.Error("removing from an unknown exercise succeeded")
	}
}
