package model

type TargetMuscle int

const (
	UpperBody TargetMuscle = iota
	TotalBody
	ChestShoulderTriceps
	Push
	BackBicepsAbdomen
	Pull
	LowerBody
	Legs
)

var targetMuscleName = map[TargetMuscle]string{
	UpperBody:            "Upper body",
	TotalBody:            "Total body",
	ChestShoulderTriceps: "Chest, shoulder & triceps",
	Push:                 "Push",
	BackBicepsAbdomen:    "Back, biceps & abdomen",
	Pull:                 "Pull",
	LowerBody:            "Lower body",
	Legs:                 "Legs",
}

func (t TargetMuscle) String() string {
	return targetMuscleName[t]
}

func AllTargetMuscles() []TargetMuscle {
	targets := make([]TargetMuscle, 0, len(targetMuscleName))
	for t := TargetMuscle(0); t < TargetMuscle(len(targetMuscleName)); t++ {
		targets = append(targets, t)
	}
	return targets
}

func (t TargetMuscle) Valid() bool {
	_, ok := targetMuscleName[t]
	return ok
}
