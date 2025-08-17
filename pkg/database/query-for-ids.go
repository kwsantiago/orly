package database

import (
	"orly.dev/pkg/database/indexes/types"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/interfaces/store"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/errorf"
	"sort"
)

// QueryForIds retrieves a list of IdPkTs based on the provided filter.
// It supports filtering by ranges and tags but disallows filtering by Ids.
// Results are sorted by timestamp in reverse chronological order.
// Returns an error if the filter contains Ids or if any operation fails.
func (d *D) QueryForIds(c context.T, f *filter.F) (
	idPkTs []*store.IdPkTs, err error,
) {
	if f.Ids != nil && f.Ids.Len() > 0 {
		// if there is Ids in the query, this is an error for this query
		err = errorf.E("query for Ids is invalid for a filter with Ids")
		return
	}
	var idxs []Range
	if idxs, err = GetIndexesFromFilter(f); chk.E(err) {
		return
	}
	var idOnly bool
	var tagIdPkTs []*store.IdPkTs
	for _, idx := range idxs {
		if f.Tags != nil && f.Tags.Len() > 1 {
			if len(tagIdPkTs) == 0 {
				// first
				var founds types.Uint40s
				if founds, err = d.GetSerialsByRange(idx); chk.E(err) {
					return
				}
				// fetch the events full id indexes
				for _, ser := range founds {
					// scan for the IdPkTs
					var fidpk *store.IdPkTs
					if fidpk, err = d.GetFullIdPubkeyBySerial(ser); chk.E(err) {
						return
					}
					if fidpk == nil {
						continue
					}
					tagIdPkTs = append(tagIdPkTs, fidpk)
				}
			} else {
				// second and subsequent
				var founds types.Uint40s
				var temp []*store.IdPkTs
				if founds, err = d.GetSerialsByRange(idx); chk.E(err) {
					return
				}
				// fetch the events full id indexes
				for _, ser := range founds {
					// scan for the IdPkTs
					var fidpk *store.IdPkTs
					if fidpk, err = d.GetFullIdPubkeyBySerial(ser); chk.E(err) {
						return
					}
					if fidpk == nil {
						continue
					}
					temp = append(temp, fidpk)
				}
				var intersecting []*store.IdPkTs
				for _, idpk := range temp {
					for _, tagIdPk := range tagIdPkTs {
						if tagIdPk.Ser == idpk.Ser {
							intersecting = append(intersecting, idpk)
						}
					}
				}
				tagIdPkTs = intersecting
			}
			// deduplicate in case this somehow happened (such as two or more
			// from one tag matched, only need it once)
			seen := make(map[uint64]struct{})
			for _, idpk := range tagIdPkTs {
				if _, ok := seen[idpk.Ser]; !ok {
					seen[idpk.Ser] = struct{}{}
					idPkTs = append(idPkTs, idpk)
				}
			}
			idPkTs = tagIdPkTs
		} else {
			var founds types.Uint40s
			if founds, err = d.GetSerialsByRange(idx); chk.E(err) {
				return
			}
			// fetch the events full id indexes
			for _, ser := range founds {
				// scan for the IdPkTs
				var fidpk *store.IdPkTs
				if fidpk, err = d.GetFullIdPubkeyBySerial(ser); chk.E(err) {
					return
				}
				if fidpk == nil {
					continue
				}
				idPkTs = append(idPkTs, fidpk)
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
	if f.Limit != nil && len(idPkTs) > int(*f.Limit) {
		idPkTs = idPkTs[:*f.Limit]
	}
	return
}
