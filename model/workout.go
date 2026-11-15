package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type WorkoutExercise struct {
	Exercise Exercise
	Sets     Sets
	Reps     Reps
	RM       RM
	Load     uint16
	Rest     time.Duration
	Notes    string
}

func (e WorkoutExercise) Weight() Weight {
	return NewWeight(e.Load)
}

func (e WorkoutExercise) Validate() error {
	if strings.TrimSpace(string(e.Exercise)) == "" {
		return errors.New("exercise name is empty")
	}
	if e.Sets == 0 {
		return fmt.Errorf("%s: sets must be greater than 0", e.Exercise)
	}
	if e.Reps == 0 {
		return fmt.Errorf("%s: reps must be greater than 0", e.Exercise)
	}
	if err := e.RM.Validate(); err != nil {
		return fmt.Errorf("%s: %w", e.Exercise, err)
	}
	return nil
}

func (e WorkoutExercise) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %dx%d (%sRM)", e.Exercise, e.Sets, e.Reps, e.RM)
	if e.Load > 0 {
		fmt.Fprintf(&b, " @ %dkg", e.Load)
	}
	if e.Rest > 0 {
		fmt.Fprintf(&b, ", rest %s", e.Rest)
	}
	if e.Notes != "" {
		fmt.Fprintf(&b, " - %s", e.Notes)
	}
	return b.String()
}

type Workout struct {
	Target    TargetMuscle
	Exercises []WorkoutExercise
}

func (w *Workout) AddExercise(e WorkoutExercise) *Workout {
	w.Exercises = append(w.Exercises, e)
	return w
}

func (w Workout) TotalSets() int {
	total := 0
	for _, e := range w.Exercises {
		total += int(e.Sets)
	}
	return total
}

func (w Workout) Validate() error {
	if len(w.Exercises) == 0 {
		return fmt.Errorf("%s workout has no exercises", w.Target)
	}
	for i, e := range w.Exercises {
		if err := e.Validate(); err != nil {
			return fmt.Errorf("exercise %d: %w", i+1, err)
		}
	}
	return nil
}

func (w Workout) String() string {
	var b strings.Builder
	b.WriteString(w.Target.String())
	for i, e := range w.Exercises {
		fmt.Fprintf(&b, "\n  %d. %s", i+1, e)
	}
	return b.String()
}
