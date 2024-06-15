// Code generated by "gocqlx/cmd/schemagen"; DO NOT EDIT.

package cqlschema

import (
	"github.com/scylladb/gocqlx/v2/table"
)

// Table models.
var (
	Short = table.New(table.Metadata{
		Name: "short",
		Columns: []string{
			"long",
			"owner",
			"short",
		},
		PartKey: []string{
			"short",
		},
		SortKey: []string{},
	})

	Users = table.New(table.Metadata{
		Name: "users",
		Columns: []string{
			"email",
			"name",
			"uid",
		},
		PartKey: []string{
			"uid",
		},
		SortKey: []string{},
	})
)

type ShortStruct struct {
	Long  string
	Owner int64
	Short string
}
type UsersStruct struct {
	Email string
	Name  string
	Uid   int64
}