package database

import (
	"not.realy.lol/context"
	"not.realy.lol/database/indexes/types"
	"not.realy.lol/filter"
)

// QueryForSerials takes a filter and returns the events that match, sorted in
// reverse chronological order, of their database serial numbers, which can then
// be retrieved using the indexes.Event table.
func (d *D) QueryForSerials(c context.T, f *filter.T) (
	sers types.Uint40s, err error,
) {
	return
}
