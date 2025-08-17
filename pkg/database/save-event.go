package database

import (
	"bytes"
	"github.com/dgraph-io/badger/v4"
	"orly.dev/pkg/database/indexes"
	"orly.dev/pkg/database/indexes/types"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/kinds"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/encoders/tag/atag"
	"orly.dev/pkg/encoders/tags"
	"orly.dev/pkg/interfaces/store"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/errorf"
	"sort"
)

// SaveEvent saves an event to the database, generating all the necessary indexes.
func (d *D) SaveEvent(
	c context.T, ev *event.E, noVerify bool, owners [][]byte,
) (kc, vc int, err error) {
	if !noVerify {
		// check if the event already exists
		var ser *types.Uint40
		if ser, err = d.GetSerialById(ev.ID); err == nil && ser != nil {
			err = errorf.E("event already exists: %0x", ev.ID)
			return
		}
	}

	// check if an existing delete event references this event submission
	if ev.Kind.IsParameterizedReplaceable() {
		var idxs []Range
		// construct a tag
		t := ev.Tags.GetFirst(tag.New("d"))
		a := atag.T{
			Kind:   ev.Kind,
			PubKey: ev.Pubkey,
			DTag:   t.Value(),
		}
		at := a.Marshal(nil)
		if idxs, err = GetIndexesFromFilter(
			&filter.F{
				Authors: tag.New(ev.Pubkey),
				Kinds:   kinds.New(kind.Deletion),
				Tags:    tags.New(tag.New([]byte("#a"), at)),
			},
		); chk.E(err) {
			return
		}
		var sers types.Uint40s
		for _, idx := range idxs {
			var s types.Uint40s
			if s, err = d.GetSerialsByRange(idx); chk.E(err) {
				return
			}
			sers = append(sers, s...)
		}
		if len(sers) > 0 {
			// there can be multiple of these because the author/kind/tag is a
			// stable value but refers to any event from the author, of the
			// kind, with the identifier. so we need to fetch the full ID index
			// to get the timestamp and ensure that the event post-dates it.
			// otherwise, it should be rejected.
			var idPkTss []*store.IdPkTs
			var tmp []*store.IdPkTs
			if tmp, err = d.GetFullIdPubkeyBySerials(sers); chk.E(err) {
				return
			}
			idPkTss = append(idPkTss, tmp...)
			// for _, ser := range sers {
			// 	var fidpk *store.IdPkTs
			// 	if fidpk, err = d.GetFullIdPubkeyBySerial(ser); chk.E(err) {
			// 		return
			// 	}
			// 	if fidpk == nil {
			// 		continue
			// 	}
			// 	idPkTss = append(idPkTss, fidpk)
			// }
			// sort by timestamp, so the first is the newest
			sort.Slice(
				idPkTss, func(i, j int) bool {
					return idPkTss[i].Ts > idPkTss[j].Ts
				},
			)
			if ev.CreatedAt.I64() < idPkTss[0].Ts {
				err = errorf.E(
					"blocked: %0x was deleted by address %s because it is older than the delete: event: %d delete: %d",
					ev.ID, at, ev.CreatedAt.I64(), idPkTss[0].Ts,
				)
				return
			}
			return
		}
	} else {
		var idxs []Range
		keys := [][]byte{ev.Pubkey}
		for _, owner := range owners {
			keys = append(keys, owner)
		}
		if idxs, err = GetIndexesFromFilter(
			&filter.F{
				Authors: tag.New(keys...),
				Kinds:   kinds.New(kind.Deletion),
				Tags:    tags.New(tag.New([]byte("#e"), ev.ID)),
			},
		); chk.E(err) {
			return
		}
		var sers types.Uint40s
		for _, idx := range idxs {
			var s types.Uint40s
			if s, err = d.GetSerialsByRange(idx); chk.E(err) {
				return
			}
			sers = append(sers, s...)
		}
		if len(sers) > 0 {
			// really there can only be one of these; the chances of an idhash
			// collision are basically zero in practice, at least, one in a
			// billion or more anyway, more than a human is going to create.
			err = errorf.E("blocked: event %0x deleted by event ID", ev.ID)
			return
		}
	}
	// Get the next sequence number for the event
	var serial uint64
	if serial, err = d.seq.Next(); chk.E(err) {
		return
	}
	// Generate all indexes for the event
	var idxs [][]byte
	if idxs, err = GetIndexesForEvent(ev, serial); chk.E(err) {
		return
	}
	// log.I.S(idxs)
	for _, k := range idxs {
		kc += len(k)
	}
	// Start a transaction to save the event and all its indexes
	err = d.Update(
		func(txn *badger.Txn) (err error) {
			// Save each index
			for _, key := range idxs {
				if err = func() (err error) {
					// Save the index to the database
					if err = txn.Set(key, nil); chk.E(err) {
						return err
					}
					return
				}(); chk.E(err) {
					return
				}
			}
			// write the event
			k := new(bytes.Buffer)
			ser := new(types.Uint40)
			if err = ser.Set(serial); chk.E(err) {
				return
			}
			if err = indexes.EventEnc(ser).MarshalWrite(k); chk.E(err) {
				return
			}
			v := new(bytes.Buffer)
			ev.MarshalBinary(v)
			kb, vb := k.Bytes(), v.Bytes()
			kc += len(kb)
			vc += len(vb)
			// log.I.S(kb, vb)
			if err = txn.Set(kb, vb); chk.E(err) {
				return
			}
			return
		},
	)
	// log.T.F("total data written: %d bytes keys %d bytes values", kc, vc)
	return
}
