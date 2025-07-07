package database

import (
	"bytes"
	"orly.dev/chk"
	"orly.dev/context"
	"orly.dev/database/indexes/types"
	"orly.dev/filter"
	"orly.dev/interfaces/store"
	"sort"
)

func (d *D) QueryForIds(c context.T, f *filter.F) (
	evs []store.IdPkTs, err error,
) {
	var idxs []Range
	if idxs, err = GetIndexesFromFilter(f); chk.E(err) {
		return
	}

	for _, idx := range idxs {
		// Id searches are a special case as they don't require iteration
		if bytes.Equal(idx.Start, idx.End) {
			// this is an Id search
			var ser *types.Uint40
			if ser, err = d.GetSerialById(idx.Start); chk.E(err) {
				return
			}
			// scan for the IdPkTs
			var fidpk *store.IdPkTs
			if fidpk, err = d.GetFullIdPubkeyBySerial(ser); chk.E(err) {
				return
			}
			evs = append(evs, *fidpk)
		} else {
			var founds types.Uint40s
			if founds, err = d.GetSerialsByRange(idx); chk.E(err) {
				return
			}
			// fetch the events
			for _, ser := range founds {
				// scan for the IdPkTs
				var fidpk *store.IdPkTs
				if fidpk, err = d.GetFullIdPubkeyBySerial(ser); chk.E(err) {
					return
				}
				evs = append(evs, *fidpk)
			}
			// sort results by timestamp in reverse chronological order
			sort.Slice(
				evs, func(i, j int) bool {
					return evs[i].Ts > evs[j].Ts
				},
			)
		}
	}
	return

}
