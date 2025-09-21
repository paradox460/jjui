package list

import "io"

type IList interface {
	Len() int
	GetItemRenderer(index int) IItemRenderer
}

type IListCursor interface {
	Cursor() int
	SetCursor(index int)
}

type IItemRenderer interface {
	Render(w io.Writer, width int)
	Height() int
}
