package model

type TargetMuscle int

const (
	UpperLower = iota
	TotalBody
	ChestShoulderTriceps
	Push
	BackBicepsAbdomen
	Pull
	LowerBody
	Legs
)

var targetMuscleName = map[TargetMuscle]string{
	UpperLower:           "Upper lower",
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
