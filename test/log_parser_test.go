package test

import (
	"os"
	"strings"
	"testing"

	"github.com/idursun/jjui/internal/parser"
	"github.com/stretchr/testify/assert"
)

func TestParser_Parse(t *testing.T) {
	file, _ := os.Open("testdata/output.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 11)
}

func TestParser_Parse_WorkingCopyCommit(t *testing.T) {
	file, _ := os.Open("testdata/working-copy-commit.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 9)
	row := rows[5]
	assert.True(t, row.Commit.IsWorkingCopy)
}

func TestParser_Parse_NoCommitId(t *testing.T) {
	file, _ := os.Open("testdata/no-commit-id.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 1)
}

func TestParser_Parse_ShortId(t *testing.T) {
	file, _ := os.Open("testdata/short-id.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 2)
	assert.Equal(t, "X", rows[0].Commit.ChangeId)
	assert.Equal(t, "E", rows[0].Commit.CommitId)
	assert.Equal(t, "T", rows[1].Commit.ChangeId)
	assert.Equal(t, "79", rows[1].Commit.CommitId)
}

func TestParser_Parse_SingleLineWithDescription(t *testing.T) {
	file, _ := os.Open("testdata/single-line-with-description.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 1)
	assert.Equal(t, "x", rows[0].Commit.ChangeId)
	assert.Equal(t, "4", rows[0].Commit.CommitId)
}

func TestParser_Parse_CommitIdOnASeparateLine(t *testing.T) {
	file, _ := os.Open("testdata/commit-id.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 1)
	assert.Equal(t, "o", rows[0].Commit.ChangeId)
	assert.Equal(t, "5", rows[0].Commit.CommitId)
}

func TestParser_Parse_ConflictedLongIds(t *testing.T) {
	file, _ := os.Open("testdata/conflicted-change-id.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 3)
	assert.Equal(t, "p??", rows[0].Commit.ChangeId)
	assert.Equal(t, "qusvoztl??", rows[1].Commit.ChangeId)
	assert.Equal(t, "tyoqvzlm??", rows[2].Commit.ChangeId)
}

func TestParser_Parse_Disconnected(t *testing.T) {
	var lb LogBuilder
	lb.Write("*   id=abcde author=some@author id=xyrq")
	lb.Write("│   some documentation")
	lb.Write("~\n")
	lb.Write("*   id=abcde author=some@author id=xyrq")
	lb.Write("│   another commit")
	lb.Write("~\n")
	rows := parser.ParseRows(strings.NewReader(lb.String()))
	assert.Len(t, rows, 2)
}

func TestParser_Parse_Extend(t *testing.T) {
	var lb LogBuilder
	lb.Write("*   id=abcde author=some@author id=xyrq")
	lb.Write("│   some documentation")

	rows := parser.ParseRows(strings.NewReader(lb.String()))
	assert.Len(t, rows, 1)
	row := rows[0]

	extended := row.Extend()
	assert.Len(t, extended.Segments, 3)
}

func TestParser_Parse_WorkingCopy(t *testing.T) {
	var lb LogBuilder
	lb.Write("*   id=abcde author=some@author id=xyrq")
	lb.Write("│   some documentation")
	lb.Write("@   id=kdys author=some@author id=12cd")
	lb.Write("│   some documentation")

	rows := parser.ParseRows(strings.NewReader(lb.String()))
	assert.Len(t, rows, 2)
	row := rows[1]

	assert.True(t, row.Commit.IsWorkingCopy)
}

func TestParser_ChangeIdLikeDescription(t *testing.T) {
	file, _ := os.Open("testdata/change-id-like-description.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 1)
}
