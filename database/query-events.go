package database

import (
	"bytes"
	"orly.dev/chk"
	"orly.dev/context"
	"orly.dev/database/indexes/types"
	"orly.dev/event"
	"orly.dev/filter"
	"orly.dev/interfaces/store"
	"sort"
)

func (d *D) QueryEvents(c context.T, f *filter.F) (evs event.S, err error) {
	var idxs []Range
	if idxs, err = GetIndexesFromFilter(f); chk.E(err) {
		return
	}
	var idOnly bool
	var idPkTs []store.IdPkTs
	for _, idx := range idxs {
		// Id searches are a special case as they don't require iteration
		if bytes.Equal(idx.Start, idx.End) {
			idOnly = true
			// this is an Id search
			var ser *types.Uint40
			if ser, err = d.GetSerialById(idx.Start); chk.E(err) {
				return
			}
			// fetch the events (we return them in the order requested)
			var ev *event.E
			if ev, err = d.FetchEventBySerial(ser); chk.E(err) {
				return
			}
			evs = append(evs, ev)
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
				idPkTs = append(idPkTs, *fidpk)
			}
		}
	}
	if idOnly {
		return
	}
	// sort results by timestamp in reverse chronological order
	sort.Slice(
		idPkTs, func(i, j int) bool {
			return idPkTs[i].Ts > idPkTs[j].Ts
		},
	)
	// fetch the events
	for _, idpk := range idPkTs {
		var ev *event.E
		ser := new(types.Uint40)
		if err = ser.Set(idpk.Ser); chk.E(err) {
			continue
		}
		if ev, err = d.FetchEventBySerial(ser); chk.E(err) {
			continue
		}
		evs = append(evs, ev)
	}
	return
}
