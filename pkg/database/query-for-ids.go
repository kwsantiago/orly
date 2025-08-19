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
	var results []*store.IdPkTs
	var founds []*types.Uint40
	for _, idx := range idxs {
		if founds, err = d.GetSerialsByRange(idx); chk.E(err) {
			return
		}
		var tmp []*store.IdPkTs
		if tmp, err = d.GetFullIdPubkeyBySerials(founds); chk.E(err) {
			return
		}
		results = append(results, tmp...)
	}
	// deduplicate in case this somehow happened (such as two or more
	// from one tag matched, only need it once)
	seen := make(map[uint64]struct{})
	for _, idpk := range results {
		if _, ok := seen[idpk.Ser]; !ok {
			seen[idpk.Ser] = struct{}{}
			idPkTs = append(idPkTs, idpk)
		}
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
