package list

import "io"

type IList interface {
	Len() int
	GetItemRenderer(index int) IItemRenderer
}

type IItemRenderer interface {
	Render(w io.Writer, width int)
	Height() int
}
