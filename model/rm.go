package model

import "fmt"

type RM struct {
	UpperBound uint8
	LowerBound uint8
}

func (rm RM) String() string {
	return fmt.Sprint("%d-%d", rm.LowerBound, rm.UpperBound)
}

func (rm RM) Validate() error {
	if rm.LowerBound > rm.UpperBound {
		return fmt.Errorf("RM lower bound %d is above upper bound %d", rm.LowerBound, rm.UpperBound)
	}
	return nil
}
