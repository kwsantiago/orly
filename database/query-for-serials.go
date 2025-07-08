package database

import (
	"orly.dev/chk"
	"orly.dev/context"
	"orly.dev/database/indexes/types"
	"orly.dev/filter"
	"orly.dev/interfaces/store"
)

// QueryForSerials takes a filter and returns the events that match, sorted in
// reverse chronological order, of their database serial numbers, which can then
// be retrieved using the indexes.Event table.
func (d *D) QueryForSerials(c context.T, f *filter.F) (
	sers types.Uint40s, err error,
) {
	var idPkTs []store.IdPkTs
	if idPkTs, err = d.QueryForIds(c, f); chk.E(err) {
		return
	}
	for _, idpk := range idPkTs {
		ser := new(types.Uint40)
		if err = ser.Set(idpk.Ser); chk.E(err) {
			return
		}
		sers = append(sers, ser)
	}
	return
}
