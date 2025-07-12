package ratel

import (
	"errors"
	"github.com/dgraph-io/badger/v4"
	"orly.dev/chk"
	"orly.dev/log"
	"sort"

	"orly.dev/context"
	"orly.dev/event"
	"orly.dev/eventid"
	"orly.dev/filter"
	"orly.dev/hex"
	"orly.dev/ratel/keys/createdat"
	"orly.dev/ratel/keys/serial"
	"orly.dev/ratel/prefixes"
)

func (r *T) QueryEvents(c context.T, f *filter.F) (evs event.S, err error) {
	log.T.F("QueryEvents %s\n", f.Serialize())
	evMap := make(map[string]*event.E)
	var queries []query
	var ext *filter.F
	var since uint64
	if queries, ext, since, err = PrepareQueries(f); chk.E(err) {
		return
	}
	// log.I.S(f, queries)
	limit := r.MaxLimit
	if f.Limit != nil {
		limit = int(*f.Limit)
	}
	// search for the keys generated from the filter
	var total int
	eventKeys := make(map[string]struct{})
	for _, q := range queries {
		select {
		case <-r.Ctx.Done():
			return
		case <-c.Done():
			return
		default:
		}
		err = r.View(
			func(txn *badger.Txn) (err error) {
				// iterate only through keys and in reverse order
				opts := badger.IteratorOptions{
					Reverse: true,
				}
				it := txn.NewIterator(opts)
				defer it.Close()
				for it.Seek(q.start); it.ValidForPrefix(q.searchPrefix); it.Next() {
					select {
					case <-r.Ctx.Done():
						return
					case <-c.Done():
						return
					default:
					}
					item := it.Item()
					k := item.KeyCopy(nil)
					if !q.skipTS {
						if len(k) < createdat.Len+serial.Len {
							continue
						}
						createdAt := createdat.FromKey(k)
						if createdAt.Val.U64() < since {
							break
						}
					}
					ser := serial.FromKey(k)
					idx := prefixes.Event.Key(ser)
					eventKeys[string(idx)] = struct{}{}
					total++
					// some queries just produce stupid amounts of matches, they are a resource
					// exhaustion attack vector and only spiders make them
					if total >= r.MaxLimit {
						return
					}
				}
				return
			},
		)
		if chk.E(err) {
			// this means shutdown, probably
			if errors.Is(err, badger.ErrDBClosed) {
				return
			}
		}
	}
	log.T.F(
		"found %d event indexes from %d queries", len(eventKeys), len(queries),
	)
	select {
	case <-r.Ctx.Done():
		return
	case <-c.Done():
		return
	default:
	}
	var delEvs [][]byte
	defer func() {
		for _, d := range delEvs {
			// if events were found that should be deleted, delete them
			chk.E(r.DeleteEvent(r.Ctx, eventid.NewWith(d)))
		}
	}()
	// accessed := make(map[string]struct{})
	for ek := range eventKeys {
		eventKey := []byte(ek)
		err = r.View(
			func(txn *badger.Txn) (err error) {
				select {
				case <-r.Ctx.Done():
					return
				case <-c.Done():
					return
				default:
				}
				opts := badger.IteratorOptions{Reverse: true}
				it := txn.NewIterator(opts)
				defer it.Close()
				for it.Seek(eventKey); it.ValidForPrefix(eventKey); it.Next() {
					item := it.Item()
					// if r.HasL2 && item.ValueSize() == sha256.Size {
					//	// todo: this isn't actually calling anything right now, it should be
					//	//  accumulating to propagate the query (this means response lag also)
					//	//
					//	// this is a stub entry that indicates an L2 needs to be accessed for it, so we
					//	// populate only the event.F.Id and return the result, the caller will expect
					//	// this as a signal to query the L2 event store.
					//	var eventValue []byte
					//	ev := &event.F{}
					//	if eventValue, err = item.ValueCopy(nil); chk.E(err) {
					//		continue
					//	}
					//	log.F.F("found event stub %0x must seek in L2", eventValue)
					//	ev.Id = eventValue
					//	select {
					//	case <-c.Done():
					//		return
					//	case <-r.Ctx.Done():
					//		log.F.Ln("backend context canceled")
					//		return
					//	default:
					//	}
					//	evMap[hex.Enc(ev.Id)] = ev
					//	return
					// }
					ev := &event.E{}
					if err = item.Value(
						func(eventValue []byte) (err error) {
							log.I.F("%s", eventValue)
							var rem []byte
							if rem, err = r.Unmarshal(
								ev, eventValue,
							); chk.E(err) {
								return
							}
							if len(rem) > 0 {
								log.T.S(rem)
							}
							// if et := ev.Tags.GetFirst(tag.New("expiration")); et != nil {
							//	var exp uint64
							//	if exp, err = strconv.ParseUint(string(et.Value()), 10,
							//		64); chk.E(err) {
							//		return
							//	}
							//	if int64(exp) > time.Now().Unix() {
							//		// this needs to be deleted
							//		delEvs = append(delEvs, ev.Id)
							//		ev = nil
							//		return
							//	}
							// }
							return
						},
					); chk.E(err) {
						continue
					}
					if ev == nil {
						continue
					}
					// if ext != nil {
					//	log.I.S(ext)
					//	log.I.S(ev)
					//	log.I.S(ext.Matches(ev))
					// }
					if ext == nil || ext.Matches(ev) {
						evMap[hex.Enc(ev.Id)] = ev
						// add event counter key to accessed
						// ser := serial.FromKey(eventKey)
						// accessed[string(ser.Val)] = struct{}{}
						// if pointers.Present(f.Limit) {
						//	*f.Limit--
						//	if *f.Limit <= 0 {
						//		log.I.F("found events: %d", len(evMap))
						//		return
						//	}
						// }
						// if there is no limit, cap it at the MaxLimit, assume this was the
						// intent or the client is erroneous, if any limit greater is
						// requested this will be used instead as the previous clause.
						if len(evMap) >= r.MaxLimit {
							// log.F.ToSliceOfBytes("found MaxLimit events: %d", len(evMap))
							return
						}
					}
				}
				return
			},
		)
		if err != nil {
			// this means shutdown, probably
			if errors.Is(err, badger.ErrDBClosed) {
				return
			}
		}
		select {
		case <-r.Ctx.Done():
			return
		case <-c.Done():
			return
		default:
		}
	}
	// log.I.S(evMap)
	if len(evMap) > 0 {
		for i := range evMap {
			if len(evMap[i].Pubkey) == 0 {
				log.I.S(evMap[i])
				continue
			}
			evs = append(evs, evMap[i])
		}
		log.I.S(len(evs))
		sort.Sort(event.Descending(evs))
		if len(evs) > limit {
			evs = evs[:limit]
		}
		seen := make(map[uint16]struct{})
		var tmp event.S
		for _, ev := range evs {
			log.I.F("%d", ev.CreatedAt.V)
			if ev.Kind.IsReplaceable() {
				// remove all but newest versions of replaceable
				if _, ok := seen[ev.Kind.K]; ok {
					// already seen this replaceable avent, skip
					continue
				}
				seen[ev.Kind.K] = struct{}{}
			}
			tmp = append(tmp, ev)
		}
		evs = tmp
		// log.I.S(evs)
		// log.F.C(func() string {
		// 	evIds := make([]string, len(evs))
		// 	for i, ev := range evs {
		// 		evIds[i] = hex.Enc(ev.Id)
		// 	}
		// 	heading := fmt.Sprintf("query complete,%d events found,%s", len(evs),
		// 		f.Serialize())
		// 	return fmt.Sprintf("%s\nevents,%v", heading, evIds)
		// })
		// bump the access times on all retrieved events. do this in a goroutine so the
		// user's events are delivered immediately
		// go func() {
		//	for ser := range accessed {
		//		seri := serial.New([]byte(ser))
		//		now := timestamp.Now()
		//		err = r.Update(func(txn *badger.Txn) (err error) {
		//			key := GetCounterKey(seri)
		//			it := txn.NewIterator(badger.IteratorOptions{})
		//			defer it.Close()
		//			if it.Seek(key); it.ValidForPrefix(key) {
		//				// update access record
		//				if err = txn.Set(key, now.Bytes()); chk.E(err) {
		//					return
		//				}
		//			}
		//			// log.F.Ln("last access for", seri.Uint64(), now.U64())
		//			return nil
		//		})
		//	}
		// }()
	} else {
		log.T.F("no events found,%s", f.Serialize())
	}
	// }
	return
}
