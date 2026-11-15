package model

type Day int

const (
	Sunday Day = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)

var dayName = map[Day]string{
	Sunday:    "Sunday",
	Monday:    "Monday",
	Tuesday:   "Tuesday",
	Wednesday: "Wednesday",
	Thursday:  "Thursday",
	Friday:    "Friday",
	Saturday:  "Saturday",
}

func (d Day) String() string {
	return dayName[d]
}

func (d Day) Valid() bool {
	return d >= Sunday && d <= Saturday
}

func AllDays() []Day {
	days := make([]Day, 0, 7)
	for d := Sunday; d <= Saturday; d++ {
		days = append(days, d)
	}
	return days
}

func (d Day) Short() string {
	return dayName[d][:3]
}
