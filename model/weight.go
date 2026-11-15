package model

import (
	"slices"
	"strconv"
	"strings"
)

type WeightTypes int

const (
	Onekg = iota
	ThreeKg
	FiveKg
	TenKg
	FifteenKg
	TwentyKg
	TwentyFiveKg
	FortyFiveKg
)

var weightTypesValues = map[WeightTypes]uint16{
	Onekg:        1,
	ThreeKg:      3,
	FiveKg:       5,
	TenKg:        10,
	FifteenKg:    15,
	TwentyKg:     20,
	TwentyFiveKg: 25,
	FortyFiveKg:  45,
}

var orderedWeightTypes = []WeightTypes{
	FortyFiveKg,
	TwentyFiveKg,
	TwentyKg,
	FifteenKg,
	TenKg,
	FiveKg,
	ThreeKg,
	Onekg,
}

type WeightCombination []WeightTypes

func (c WeightCombination) Total() uint16 {
	var total uint16
	for _, w := range c {
		total += weightTypesValues[w]
	}
	return total
}

type Weight struct {
	Value        uint16
	Combinations []WeightCombination
}

func (w WeightTypes) Value() uint16 {
	return weightTypesValues[w]
}

func NewWeight(value uint16) Weight {
	return Weight{
		Value:        value,
		Combinations: CalculateCombinations(value),
	}
}

func CalculateCombinations(target uint16) []WeightCombination {
	if target == 0 {
		return nil
	}

	var (
		result  []WeightCombination
		current WeightCombination
		search  func(remaining uint16, start int)
	)

	search = func(remaining uint16, start int) {
		if remaining == 0 {
			combo := make(WeightCombination, len(current))
			copy(combo, current)
			result = append(result, combo)
			return
		}

		for i := start; i < len(orderedWeightTypes); i++ {
			plate := orderedWeightTypes[i]
			value := weightTypesValues[plate]
			if value > remaining {
				continue
			}
			current = append(current, plate)
			search(remaining-value, i)
			current = current[:len(current)-1]
		}
	}

	search(target, 0)
	return result
}

func (c WeightCombination) String() string {
	parts := make([]string, len(c))
	for i, w := range c {
		parts[i] = strconv.Itoa(int(weightTypesValues[w]))
	}
	return strings.Join(parts, " + ")
}

// GreedyCombination returns the first combination found by taking the
// largest plates first, without enumerating every alternative.
func GreedyCombination(target uint16) WeightCombination {
	if target == 0 {
		return nil
	}
	var (
		found   WeightCombination
		current WeightCombination
		search  func(remaining uint16, start int) bool
	)
	search = func(remaining uint16, start int) bool {
		if remaining == 0 {
			found = slices.Clone(current)
			return true
		}
		for i := start; i < len(orderedWeightTypes); i++ {
			value := weightTypesValues[orderedWeightTypes[i]]
			if value > remaining {
				continue
			}
			current = append(current, orderedWeightTypes[i])
			if search(remaining-value, i) {
				return true
			}
			current = current[:len(current)-1]
		}
		return false
	}
	search(target, 0)
	return found
}
