package common

type ViewRange struct {
	*Sizeable
	Start         int
	End           int
	FirstRowIndex int
	LastRowIndex  int
}
