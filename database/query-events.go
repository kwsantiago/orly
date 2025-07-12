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

// QueryEvents retrieves events based on the provided filter.
// If the filter contains Ids, it fetches events by those Ids directly,
// overriding other filter criteria. Otherwise, it queries by other filter
// criteria and fetches matching events. Results are returned in reverse
// chronological order of their creation timestamps.
func (d *D) QueryEvents(c context.T, f *filter.F) (evs event.S, err error) {
	// if there is Ids in the query, this overrides anything else
	if f.Ids != nil && f.Ids.Len() > 0 {
		var idxs []Range
		if idxs, err = GetIndexesFromFilter(f); chk.E(err) {
			return
		}
		for _, idx := range idxs {
			// we know there is only Ids in this, so run the ID query and fetch.
			var ser *types.Uint40
			if ser, err = d.GetSerialById(idx.Start); chk.E(err) {
				continue
			}
			// fetch the events
			var ev *event.E
			if ev, err = d.FetchEventBySerial(ser); chk.E(err) {
				continue
			}
			evs = append(evs, ev)
		}
		// sort the events by timestamp
		sort.Slice(
			evs, func(i, j int) bool {
				return evs[i].CreatedAt.I64() > evs[j].CreatedAt.I64()
			},
		)
	} else {
		var idPkTs []store.IdPkTs
		if idPkTs, err = d.QueryForIds(c, f); chk.E(err) {
			return
		}
		// fetch the events
		for _, idpk := range idPkTs {
			var ev *event.E
			ser := new(types.Uint40)
			if err = ser.Set(idpk.Ser); chk.E(err) {
				continue
			}
			if ev, err = d.FetchEventBySerial(ser); err != nil {
				continue
			}
			if ev.Kind.IsReplaceable() {
				for _, e := range evs {
					if bytes.Equal(
						ev.Pubkey, e.Pubkey,
					) && ev.Kind.K == e.Kind.K {

					}
				}
				// } else if ev.Kind.IsParameterizedReplaceable(){

			} else {

			}
			evs = append(evs, ev)
		}
	}
	return
}
