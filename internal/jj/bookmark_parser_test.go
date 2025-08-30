package jj

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBookmarkListOutput_WithNonLocalBookmarks(t *testing.T) {
	output := `alpha;origin;false;false;false;2
main;.;false;false;false;b
main;git;true;false;false;b
main;origin;true;false;false;b
zeta;origin;false;false;false;c`
	bookmarks := ParseBookmarkListOutput(output)
	assert.Len(t, bookmarks, 3)

	alpha := bookmarks[slices.IndexFunc(bookmarks, func(b Bookmark) bool { return b.Name == "alpha" })]
	assert.Nil(t, alpha.Local, "alpha should not have a local bookmark")
	assert.Len(t, alpha.Remotes, 1)
	main := bookmarks[slices.IndexFunc(bookmarks, func(b Bookmark) bool { return b.Name == "main" })]
	assert.NotNil(t, main.Local, "main should have a local bookmark")
	assert.Len(t, main.Remotes, 1)
	zeta := bookmarks[slices.IndexFunc(bookmarks, func(b Bookmark) bool { return b.Name == "zeta" })]
	assert.Nil(t, zeta.Local, "zeta should not have a local bookmark")
	assert.Len(t, zeta.Remotes, 1)
}

func TestParseBookmarkListOutput(t *testing.T) {
	type args struct {
		output string
	}
	tests := []struct {
		name string
		args args
		want []Bookmark
	}{
		{
			name: "empty",
			args: args{
				output: "",
			},
			want: nil,
		},
		{
			name: "single",
			args: args{
				output: "feat-1;.;false;false;false;9",
			},
			want: []Bookmark{
				{
					Name:    "feat-1",
					Remotes: nil,
					Local: &BookmarkRemote{
						Remote:   ".",
						CommitId: "9",
						Tracked:  false,
					},
					Conflict:  false,
					Backwards: false,
					CommitId:  "9",
				},
			},
		},
		{
			name: "remote",
			args: args{
				output: `feature;.;false;false;false;b
feature;origin;true;false;false;b`,
			},
			want: []Bookmark{
				{
					Name: "feature",
					Remotes: []BookmarkRemote{
						{"origin", "b", true},
					},
					Local: &BookmarkRemote{
						Remote:   ".",
						CommitId: "b",
						Tracked:  false,
					},
					Conflict:  false,
					Backwards: false,
					CommitId:  "b",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, ParseBookmarkListOutput(tt.args.output), "ParseBookmarkListOutput(%v)", tt.args.output)
		})
	}
}
