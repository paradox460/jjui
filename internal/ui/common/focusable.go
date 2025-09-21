package common

type Focusable interface {
	IsFocused() bool
}

type Editable interface {
	IsEditing() bool
}
