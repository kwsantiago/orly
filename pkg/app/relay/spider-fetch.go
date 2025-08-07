package relay

import (
	"runtime/debug"
	"time"

	"orly.dev/pkg/crypto/ec/schnorr"
	"orly.dev/pkg/database/indexes/types"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/encoders/kinds"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/protocol/ws"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/errorf"
	"orly.dev/pkg/utils/log"
	"orly.dev/pkg/utils/values"
)

// IdPkTs is a map of event IDs to their id, pubkey, kind, and timestamp
// This is used to reduce memory usage by storing only the essential information
// instead of the full events
type IdPkTs struct {
	Id        []byte
	Pubkey    []byte
	Kind      uint16
	Timestamp int64
}

func (s *Server) SpiderFetch(
	k *kinds.T, noFetch, noExtract bool, pubkeys ...[]byte,
) (pks [][]byte, err error) {
	// Map to store id, pubkey, kind, and timestamp for each event
	// Key is a combination of pubkey and kind for deduplication
	pkKindMap := make(map[string]*IdPkTs)
	// Map to collect pubkeys from p tags
	pkMap := make(map[string]struct{})

	// first search the local database
	pkList := tag.New(pubkeys...)
	f := &filter.F{
		Kinds:   k,
		Authors: pkList,
	}

	var kindsList string
	if k != nil {
		for i, kk := range k.K {
			if i > 0 {
				kindsList += ","
			}
			kindsList += kk.Name()
		}
	} else {
		kindsList = "*"
	}

	// Query local database
	var localEvents event.S
	if localEvents, err = s.Storage().QueryEvents(s.Ctx, f); chk.E(err) {
		// none were found, so we need to scan the spiders
		err = nil
	}

	// Process local events
	for _, ev := range localEvents {
		// Create a key based on pubkey and kind for deduplication
		pkKindKey := string(ev.Pubkey) + string(ev.Kind.Marshal(nil))

		// Check if we already have an event with this pubkey and kind
		existing, exists := pkKindMap[pkKindKey]

		// If it doesn't exist or the new event is newer, store it
		if !exists || ev.CreatedAtInt64() > existing.Timestamp {
			pkKindMap[pkKindKey] = &IdPkTs{
				Id:        ev.ID,
				Pubkey:    ev.Pubkey,
				Kind:      ev.Kind.ToU16(),
				Timestamp: ev.CreatedAtInt64(),
			}

			// Extract p tags if not in noExtract mode
			if !noExtract {
				t := ev.Tags.GetAll(tag.New("p"))
				for _, tt := range t.ToSliceOfTags() {
					pkh := tt.Value()
					if len(pkh) != 2*schnorr.PubKeyBytesLen {
						continue
					}
					pk := make([]byte, schnorr.PubKeyBytesLen)
					if _, err = hex.DecBytes(pk, pkh); err != nil {
						err = nil
						continue
					}
					pkMap[string(pk)] = struct{}{}
				}
			}
		}

		// Nil the event to free memory
		ev = nil
	}

	log.I.F("%d events found of type %s", len(pkKindMap), kindsList)

	if !noFetch && len(s.C.SpiderSeeds) > 0 {
		// we need to search the spider seeds.
		// Break up pubkeys into batches of 128
		for i := 0; i < len(pubkeys); i += 128 {
			end := i + 128
			if end > len(pubkeys) {
				end = len(pubkeys)
			}
			batchPubkeys := pubkeys[i:end]
			log.I.F(
				"processing batch %d to %d of %d for kind %s",
				i, end, len(pubkeys), kindsList,
			)
			batchPkList := tag.New(batchPubkeys...)
			lim := uint(batchPkList.Len())
			l := &lim
			var since *timestamp.T
			if k == nil {
				since = timestamp.FromTime(time.Now().Add(-1 * s.C.SpiderTime * 3 / 2))
			} else {
				l = values.ToUintPointer(512)
			}
			batchFilter := &filter.F{
				Kinds:   k,
				Authors: batchPkList,
				Since:   since,
				Limit:   l,
			}
			for _, seed := range s.C.SpiderSeeds {
				select {
				case <-s.Ctx.Done():
					return
				default:
				}
				var evss event.S
				var cli *ws.Client
				if cli, err = ws.RelayConnect(
					context.Bg(), seed,
				); chk.E(err) {
					err = nil
					continue
				}
				if evss, err = cli.QuerySync(
					context.Bg(), batchFilter,
				); chk.E(err) {
					err = nil
					return
				}
				// Process each event immediately
				for i, ev := range evss {
					// log.I.S(ev)
					// Create a key based on pubkey and kind for deduplication
					pkKindKey := string(ev.Pubkey) + string(ev.Kind.Marshal(nil))
					// Check if we already have an event with this pubkey and kind
					existing, exists := pkKindMap[pkKindKey]
					// If it doesn't exist or the new event is newer, store it and save to database
					if !exists || ev.CreatedAtInt64() > existing.Timestamp {
						var ser *types.Uint40
						if ser, err = s.Storage().GetSerialById(ev.ID); err == nil && ser != nil {
							err = errorf.E("event already exists: %0x", ev.ID)
							return
						} else {
							// verify the signature
							var valid bool
							if valid, err = ev.Verify(); chk.E(err) || !valid {
								continue
							}
						}
						// Save the event to the database
						if _, _, err = s.Storage().SaveEvent(
							s.Ctx, ev, true, nil,
						); chk.E(err) {
							err = nil
							continue
						}
						// Store the essential information
						pkKindMap[pkKindKey] = &IdPkTs{
							Id:        ev.ID,
							Pubkey:    ev.Pubkey,
							Kind:      ev.Kind.ToU16(),
							Timestamp: ev.CreatedAtInt64(),
						}
						// Extract p tags if not in noExtract mode
						if !noExtract {
							t := ev.Tags.GetAll(tag.New("p"))
							for _, tt := range t.ToSliceOfTags() {
								pkh := tt.Value()
								if len(pkh) != 2*schnorr.PubKeyBytesLen {
									continue
								}
								pk := make([]byte, schnorr.PubKeyBytesLen)
								if _, err = hex.DecBytes(pk, pkh); err != nil {
									err = nil
									continue
								}
								pkMap[string(pk)] = struct{}{}
							}
						}
					}
					// Nil the event in the slice to free memory
					evss[i] = nil
				}
			}
		}
	}
	chk.E(s.Storage().Sync())
	debug.FreeOSMemory()
	// If we're in noExtract mode, just return
	if noExtract {
		return
	}
	// Convert the collected pubkeys to the return format
	for pk := range pkMap {
		pks = append(pks, []byte(pk))
	}
	log.I.F("found %d pks", len(pks))
	return
}
