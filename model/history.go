package model

import (
	"slices"
	"time"
)

type History []Session

type ExerciseEntry struct {
	Date time.Time
	Log  ExerciseLog
}

func (h History) Sorted() History {
	sorted := slices.Clone(h)
	slices.SortStableFunc(sorted, func(a, b Session) int {
		return a.StartedAt.Compare(b.StartedAt)
	})
	return sorted
}

func (h History) ForExercise(ex Exercise) []ExerciseEntry {
	var entries []ExerciseEntry
	for _, s := range h.Sorted() {
		for _, l := range s.Exercises {
			if l.Planned.Exercise == ex && len(l.CompletedSets()) > 0 {
				entries = append(entries, ExerciseEntry{Date: s.StartedAt, Log: l})
			}
		}
	}
	return entries
}

func (h History) LastLoad(programID string, day Day, ex Exercise) (uint16, bool) {
	sorted := h.Sorted()
	sameSlot := func(s Session) bool { return s.ProgramID == programID && s.Day == day }
	anywhere := func(Session) bool { return true }

	for _, match := range []func(Session) bool{sameSlot, anywhere} {
		for i := len(sorted) - 1; i >= 0; i-- {
			if !match(sorted[i]) {
				continue
			}
			for _, l := range sorted[i].Exercises {
				if l.Planned.Exercise != ex {
					continue
				}
				done := l.CompletedSets()
				if len(done) > 0 {
					return done[len(done)-1].Load, true
				}
			}
		}
	}
	return 0, false
}

func (h History) Week(t time.Time) History {
	start := time.Date(t.Year(), t.Month(), t.Day()-int(t.Weekday()), 0, 0, 0, 0, t.Location())
	end := start.AddDate(0, 0, 7)
	var week History
	for _, s := range h.Sorted() {
		if !s.StartedAt.Before(start) && s.StartedAt.Before(end) {
			week = append(week, s)
		}
	}
	return week
}

func (h History) WeekProgress(p *Program, t time.Time) map[Day]bool {
	progress := make(map[Day]bool)
	for _, d := range p.TrainingDays() {
		progress[d] = false
	}
	for _, s := range h.Week(t) {
		if s.ProgramID != p.ID || !s.Finished() {
			continue
		}
		if _, ok := progress[s.Day]; ok {
			progress[s.Day] = true
		}
	}
	return progress
}
