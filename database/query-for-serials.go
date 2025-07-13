package database

import (
	"orly.dev/database/indexes/types"
	"orly.dev/encoders/filter"
	"orly.dev/interfaces/store"
	"orly.dev/utils/chk"
	"orly.dev/utils/context"
	"sort"
)

// QueryForSerials takes a filter and returns the serials of events that match,
// sorted in reverse chronological order.
func (d *D) QueryForSerials(c context.T, f *filter.F) (
	sers types.Uint40s, err error,
) {
	var founds types.Uint40s
	var idPkTs []store.IdPkTs
	if f.Ids != nil && f.Ids.Len() > 0 {
		var idxs []Range
		if idxs, err = GetIndexesFromFilter(f); chk.E(err) {
			return
		}
		for _, idx := range idxs {
			var ser *types.Uint40
			if ser, err = d.GetSerialById(idx.Start); chk.E(err) {
				return
			}
			founds = append(founds, ser)
		}
		// fetch the events full id indexes so we can sort them
		for _, ser := range founds {
			// scan for the IdPkTs
			var fidpk *store.IdPkTs
			if fidpk, err = d.GetFullIdPubkeyBySerial(ser); chk.E(err) {
				return
			}
			idPkTs = append(idPkTs, *fidpk)
			// sort by timestamp
			sort.Slice(
				idPkTs, func(i, j int) bool {
					return idPkTs[i].Ts > idPkTs[j].Ts
				},
			)
		}
	} else {
		if idPkTs, err = d.QueryForIds(c, f); chk.E(err) {
			return
		}
	}
	// extract the serials
	for _, idpk := range idPkTs {
		ser := new(types.Uint40)
		if err = ser.Set(idpk.Ser); chk.E(err) {
			continue
		}
		sers = append(sers, ser)
	}
	return
}
