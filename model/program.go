package model

import (
	"fmt"
	"strings"
)

type Program struct {
	ID       string
	Name     string
	Workouts map[Day]Workout
}

func NewProgram(name string) *Program {
	return &Program{
		Name:     name,
		Workouts: make(map[Day]Workout),
	}
}

func (p *Program) SetWorkout(day Day, w Workout) error {
	if !day.Valid() {
		return fmt.Errorf("invalid day %d", day)
	}
	if err := w.Validate(); err != nil {
		return fmt.Errorf("%s: %w", day, err)
	}
	if p.Workouts == nil {
		p.Workouts = make(map[Day]Workout)
	}
	p.Workouts[day] = w
	return nil
}

func (p *Program) RemoveWorkout(day Day) {
	delete(p.Workouts, day)
}

func (p *Program) Workout(day Day) (Workout, bool) {
	w, ok := p.Workouts[day]
	return w, ok
}

func (p *Program) TrainingDays() []Day {
	var days []Day
	for d := Sunday; d <= Saturday; d++ {
		if _, ok := p.Workouts[d]; ok {
			days = append(days, d)
		}
	}
	return days
}

func (p *Program) RestDays() []Day {
	var days []Day
	for d := Sunday; d <= Saturday; d++ {
		if _, ok := p.Workouts[d]; !ok {
			days = append(days, d)
		}
	}
	return days
}

func (p *Program) TotalSets() int {
	total := 0
	for _, w := range p.Workouts {
		total += w.TotalSets()
	}
	return total
}

func (p *Program) Validate() error {
	if len(p.Workouts) == 0 {
		return fmt.Errorf("program %q has no workouts", p.Name)
	}
	for day, w := range p.Workouts {
		if !day.Valid() {
			return fmt.Errorf("invalid day %d", day)
		}
		if err := w.Validate(); err != nil {
			return fmt.Errorf("%s: %w", day, err)
		}
	}
	return nil
}

func (p *Program) String() string {
	var b strings.Builder
	b.WriteString(p.Name)
	for _, d := range p.TrainingDays() {
		fmt.Fprintf(&b, "\n%s - %s", d, p.Workouts[d])
	}
	return b.String()
}
