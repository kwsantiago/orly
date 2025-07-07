package database

import (
	"bytes"
	"github.com/dgraph-io/badger/v4"
	"orly.dev/chk"
	"orly.dev/codecbuf"
	"orly.dev/context"
	"orly.dev/database/indexes"
	"orly.dev/database/indexes/types"
	"orly.dev/filter"
	"orly.dev/interfaces/store"
	"sort"
)

func (d *D) QueryForIds(c context.T, f *filter.F) (
	evs []store.IdTsPk, err error,
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
			if err = d.View(
				func(txn *badger.Txn) (err error) {
					it := txn.NewIterator(badger.DefaultIteratorOptions)
					var key []byte
					defer it.Close()
					it.Seek(idx.Start)
					if it.Valid() {
						item := it.Item()
						key = item.KeyCopy(nil)
						ser = new(types.Uint40)
						buf := bytes.NewBuffer(key[len(key)-5:])
						if err = ser.UnmarshalRead(buf); chk.E(err) {
							return
						}
					} else {
						// just don't return what we don't have? others may be
						// found tho.
					}
					return
				},
			); chk.E(err) {
				return
			}
			// scan for the IdTsPk
			if err = d.View(
				func(txn *badger.Txn) (err error) {
					buf := codecbuf.Get()
					defer codecbuf.Put(buf)
					if err = indexes.IdPubkeyEnc(
						ser, nil, nil, nil,
					).MarshalWrite(buf); chk.E(err) {
						return
					}
					prf := buf.Bytes()
					it := txn.NewIterator(
						badger.IteratorOptions{
							Prefix: prf,
						},
					)
					defer it.Close()
					it.Seek(prf)
					if it.Valid() {
						item := it.Item()
						key := item.KeyCopy(nil)
						ser, fid, p, ca := indexes.IdPubkeyVars()
						buf2 := bytes.NewBuffer(key)
						if err = indexes.IdPubkeyDec(
							ser, fid, p, ca,
						).UnmarshalRead(buf2); chk.E(err) {
							return
						}
						idtspk := store.IdTsPk{
							Id:  fid.Bytes(),
							Pub: p.Bytes(),
							Ts:  int64(ca.Get()),
						}
						evs = append(evs, idtspk)
					}
					return
				},
			); chk.E(err) {
				return
			}
		} else {
			prf := idx.End[:len(idx.End)-types.TimestampLen]
			var founds types.Uint40s
			if err = d.View(
				func(txn *badger.Txn) (err error) {
					it := txn.NewIterator(
						badger.IteratorOptions{
							Reverse: true,
						},
					)
					defer it.Close()
					var count int
					for it.Rewind(); it.Valid(); it.Next() {
						count++
						item := it.Item()
						var key, val []byte
						_ = val
						key = item.KeyCopy(nil)
						if !bytes.HasPrefix(key, prf) {
							continue
						}
						ser := new(types.Uint40)
						buf := bytes.NewBuffer(key[len(key)-5:])
						if err = ser.UnmarshalRead(buf); chk.E(err) {
							return
						}
						founds = append(founds, ser)
					}
					return
				},
			); chk.E(err) {
				return
			}
			// fetch the events
			for _, ser := range founds {
				// scan for the IdTsPk
				if err = d.View(
					func(txn *badger.Txn) (err error) {
						buf := codecbuf.Get()
						defer codecbuf.Put(buf)
						if err = indexes.IdPubkeyEnc(
							ser, nil, nil, nil,
						).MarshalWrite(buf); chk.E(err) {
							return
						}
						prf := buf.Bytes()
						it := txn.NewIterator(
							badger.IteratorOptions{
								Prefix: prf,
							},
						)
						defer it.Close()
						it.Seek(prf)
						if it.Valid() {
							item := it.Item()
							key := item.KeyCopy(nil)
							ser, fid, p, ca := indexes.IdPubkeyVars()
							buf2 := bytes.NewBuffer(key)
							if err = indexes.IdPubkeyDec(
								ser, fid, p, ca,
							).UnmarshalRead(buf2); chk.E(err) {
								return
							}
							idtspk := store.IdTsPk{
								Id:  fid.Bytes(),
								Pub: p.Bytes(),
								Ts:  int64(ca.Get()),
							}
							evs = append(evs, idtspk)
						}
						return
					},
				); chk.E(err) {
					return
				}
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
