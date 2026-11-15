package model

import (
	"errors"
	"fmt"
	"time"
)

func DayOf(t time.Time) Day {
	return Day(t.Weekday())
}

type SetLog struct {
	Reps Reps
	Load uint16
	Done bool
}

type ExerciseLog struct {
	Planned WorkoutExercise
	Sets    []SetLog
	Notes   string
}

func (l ExerciseLog) Exercise() Exercise {
	return l.Planned.Exercise
}

func (l ExerciseLog) CompletedSets() []SetLog {
	var done []SetLog
	for _, s := range l.Sets {
		if s.Done {
			done = append(done, s)
		}
	}
	return done
}

func (l ExerciseLog) TopLoad() uint16 {
	var top uint16
	for _, s := range l.CompletedSets() {
		if s.Load > top {
			top = s.Load
		}
	}
	return top
}

func (l ExerciseLog) Volume() int {
	total := 0
	for _, s := range l.CompletedSets() {
		total += int(s.Reps) * int(s.Load)
	}
	return total
}

func (l ExerciseLog) HitTopOfRange() bool {
	planned := int(l.Planned.Sets)
	if planned == 0 || len(l.Sets) < planned {
		return false
	}
	for _, s := range l.Sets[:planned] {
		if !s.Done || uint8(s.Reps) < l.Planned.RM.UpperBound {
			return false
		}
	}
	return true
}

type Session struct {
	ID         string
	ProgramID  string
	Day        Day
	Target     TargetMuscle
	StartedAt  time.Time
	FinishedAt time.Time
	Exercises  []ExerciseLog
	Notes      string
}

func NewSession(p *Program, day Day, startedAt time.Time, history History) (Session, error) {
	w, ok := p.Workout(day)
	if !ok {
		return Session{}, fmt.Errorf("%s is a rest day in %q", day, p.Name)
	}

	s := Session{
		ProgramID: p.ID,
		Day:       day,
		Target:    w.Target,
		StartedAt: startedAt,
		Exercises: make([]ExerciseLog, 0, len(w.Exercises)),
	}
	for _, planned := range w.Exercises {
		load := planned.Load
		if last, ok := history.LastLoad(p.ID, day, planned.Exercise); ok {
			load = last
		}
		sets := make([]SetLog, planned.Sets)
		for i := range sets {
			sets[i] = SetLog{Reps: planned.Reps, Load: load}
		}
		s.Exercises = append(s.Exercises, ExerciseLog{Planned: planned, Sets: sets})
	}
	return s, nil
}

func (s *Session) Exercise(ex Exercise) *ExerciseLog {
	for i := range s.Exercises {
		if s.Exercises[i].Planned.Exercise == ex {
			return &s.Exercises[i]
		}
	}
	return nil
}

func (s *Session) LogSet(ex Exercise, setIndex int, reps Reps, load uint16) error {
	if s.Finished() {
		return errors.New("session is already finished")
	}
	l := s.Exercise(ex)
	if l == nil {
		return fmt.Errorf("%s is not part of this session", ex)
	}
	switch {
	case setIndex < 0 || setIndex > len(l.Sets):
		return fmt.Errorf("%s: set %d does not exist", ex, setIndex+1)
	case setIndex == len(l.Sets):
		l.Sets = append(l.Sets, SetLog{})
	}
	l.Sets[setIndex] = SetLog{Reps: reps, Load: load, Done: true}
	return nil
}

func (s *Session) SetLoad(ex Exercise, load uint16) error {
	l := s.Exercise(ex)
	if l == nil {
		return fmt.Errorf("%s is not part of this session", ex)
	}
	for i := range l.Sets {
		if !l.Sets[i].Done {
			l.Sets[i].Load = load
		}
	}
	return nil
}

func (s *Session) Finish(at time.Time) {
	s.FinishedAt = at
}

func (s Session) Finished() bool {
	return !s.FinishedAt.IsZero()
}

func (s Session) Duration() time.Duration {
	if !s.Finished() {
		return 0
	}
	return s.FinishedAt.Sub(s.StartedAt)
}

func (s Session) Volume() int {
	total := 0
	for _, l := range s.Exercises {
		total += l.Volume()
	}
	return total
}
