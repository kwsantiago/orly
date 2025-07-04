package database

import (
	"bytes"
	"github.com/dgraph-io/badger/v4"
	"not.realy.lol/chk"
	"not.realy.lol/codecbuf"
	"not.realy.lol/context"
	"not.realy.lol/database/indexes"
	"not.realy.lol/database/indexes/types/idhash"
	"not.realy.lol/filter"
	"not.realy.lol/interfaces/store"
)

func (d *D) QueryForIds(c context.T, f *filter.T) (
	evs []store.IdTsPk, err error,
) {
	// Use a read-only transaction to query the database
	err = d.View(
		func(txn *badger.Txn) (err error) {
			// Get a buffer from the pool
			buf := codecbuf.Get()
			defer codecbuf.Put(buf)

			// If the filter has IDs, use the Id index to find matching events
			if f.IDs != nil && f.IDs.Len() > 0 {
				for i := 0; i < f.IDs.Len(); i++ {
					id := f.IDs.B(i)

					// Create an idhash from the ID
					idHash := idhash.New()
					idHash.FromId(id)

					// Create a search key for the Id index
					idSearch := indexes.IdSearch(idHash)
					buf.Reset()
					if err = idSearch.MarshalWrite(buf); chk.E(err) {
						return
					}

					// Create an iterator to scan the Id index
					opts := badger.DefaultIteratorOptions
					opts.PrefetchValues = false
					it := txn.NewIterator(opts)
					defer it.Close()

					// Seek to the first key that matches the search prefix
					prefix := buf.Bytes()
					for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
						item := it.Item()
						key := item.Key()

						// Decode the key to get the serial number
						idDec, ser := indexes.IdVars()
						idDecT := indexes.IdDec(idDec, ser)
						if err = idDecT.UnmarshalRead(bytes.NewReader(key)); chk.E(err) {
							return
						}

						// Use the serial number to find the corresponding
						// IdPubkeyCreatedAt index
						ipcBuf := codecbuf.Get()
						defer codecbuf.Put(ipcBuf)
						ipcSearch := indexes.IdPubkeyCreatedAtSearch(ser)
						if err = ipcSearch.MarshalWrite(ipcBuf); chk.E(err) {
							return
						}

						// Create an iterator to scan the IdPubkeyCreatedAt index
						ipcIt := txn.NewIterator(opts)
						defer ipcIt.Close()

						// Seek to the first key that matches the search prefix
						ipcPrefix := ipcBuf.Bytes()
						for ipcIt.Seek(ipcPrefix); ipcIt.ValidForPrefix(ipcPrefix); ipcIt.Next() {
							ipcItem := ipcIt.Item()
							ipcKey := ipcItem.Key()

							// Decode the key to get the ID, pubkey, and timestamp
							_, fullID, pubHash, createdAt := indexes.IdPubkeyCreatedAtVars()
							ipcDecT := indexes.IdPubkeyCreatedAtDec(
								ser, fullID, pubHash, createdAt,
							)
							if err = ipcDecT.UnmarshalRead(bytes.NewReader(ipcKey)); chk.E(err) {
								return
							}

							// Create an IdTsPk object and add it to the result
							idTsPk := store.IdTsPk{
								Ts:  int64(createdAt.Get()),
								Id:  fullID.Bytes(),
								Pub: pubHash.Bytes(),
							}
							evs = append(evs, idTsPk)

							// We only need one match per ID
							break
						}
					}
				}
			}

			return nil
		},
	)

	return
}
