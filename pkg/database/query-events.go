package database

import (
	"bytes"
	"fmt"
	"orly.dev/pkg/crypto/sha256"
	"orly.dev/pkg/database/indexes/types"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/encoders/ints"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/kinds"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/interfaces/store"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"sort"
	"strconv"
	"time"
)

func CheckExpiration(ev *event.E) (expired bool) {
	var err error
	expTag := ev.Tags.GetFirst(tag.New("expiration"))
	if expTag != nil {
		expTS := ints.New(0)
		if _, err = expTS.Unmarshal(expTag.Value()); !chk.E(err) {
			if int64(expTS.N) < time.Now().Unix() {
				return true
			}
		}
	}
	return
}

func (d *D) QueryEvents(c context.T, f *filter.F) (evs event.S, err error) {
	// if there is Ids in the query, this overrides anything else
	var expDeletes types.Uint40s
	var expEvs event.S
	if f.Ids != nil && f.Ids.Len() > 0 {
		for _, idx := range f.Ids.ToSliceOfBytes() {
			// we know there is only Ids in this, so run the ID query and fetch.
			var ser *types.Uint40
			if ser, err = d.GetSerialById(idx); chk.E(err) {
				continue
			}
			// fetch the events
			var ev *event.E
			if ev, err = d.FetchEventBySerial(ser); err != nil {
				continue
			}
			// check for an expiration tag and delete after returning the result
			if CheckExpiration(ev) {
				expDeletes = append(expDeletes, ser)
				expEvs = append(expEvs, ev)
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
		// Create a map to store the latest version of replaceable events
		replaceableEvents := make(map[string]*event.E)
		// Create a map to store the latest version of parameterized replaceable
		// events
		paramReplaceableEvents := make(map[string]map[string]*event.E)
		// Regular events that are not replaceable
		var regularEvents event.S
		// Map to track deletion events by kind and pubkey (for replaceable
		// events)
		deletionsByKindPubkey := make(map[string]bool)
		// Map to track deletion events by kind, pubkey, and d-tag (for
		// parameterized replaceable events)
		deletionsByKindPubkeyDTag := make(map[string]map[string]bool)
		// Map to track specific event IDs that have been deleted
		deletedEventIds := make(map[string]bool)
		// Query for deletion events separately if we have authors in the filter
		if f.Authors != nil && f.Authors.Len() > 0 {
			// Create a filter for deletion events with the same authors
			deletionFilter := &filter.F{
				Kinds:   kinds.New(kind.New(5)), // Kind 5 is deletion
				Authors: f.Authors,
			}

			var deletionIdPkTs []store.IdPkTs
			if deletionIdPkTs, err = d.QueryForIds(
				c, deletionFilter,
			); chk.E(err) {
				return
			}

			// Add deletion events to the list of events to process
			idPkTs = append(idPkTs, deletionIdPkTs...)
		}
		// First pass: collect all deletion events
		log.T.C(
			func() string {
				return fmt.Sprintf(
					"Debug: Starting first pass - processing %d events\n",
					len(idPkTs),
				)
			},
		)
		for _, idpk := range idPkTs {
			var ev *event.E
			ser := new(types.Uint40)
			if err = ser.Set(idpk.Ser); chk.E(err) {
				continue
			}
			if ev, err = d.FetchEventBySerial(ser); err != nil {
				continue
			}
			// check for an expiration tag and delete after returning the result
			if CheckExpiration(ev) {
				expDeletes = append(expDeletes, ser)
				expEvs = append(expEvs, ev)
				continue
			}
			// Process deletion events to build our deletion maps
			if ev.Kind.Equal(kind.Deletion) {
				log.T.C(
					func() string {
						return fmt.Sprintf(
							"found deletion event with ID: %s\n",
							hex.Enc(ev.ID),
						)
					},
				)
				// Check for 'e' tags that directly reference event IDs
				eTags := ev.Tags.GetAll(tag.New([]byte{'e'}))
				for _, eTag := range eTags.ToSliceOfTags() {
					if eTag.Len() < 2 {
						continue
					}
					// We don't need to do anything with direct event ID
					// references as we will filter those out in the second pass
				}
				// Check for 'a' tags that reference parameterized replaceable
				// events
				log.T.C(
					func() string {
						return fmt.Sprintf(
							"processing deletion event with ID: %s\n",
							hex.Enc(ev.ID),
						)
					},
				)
				aTags := ev.Tags.GetAll(tag.New([]byte{'a'}))
				log.D.C(
					func() string {
						return fmt.Sprintf(
							"Found %d a-tags\n", aTags.Len(),
						)
					},
				)
				for _, aTag := range aTags.ToSliceOfTags() {
					if aTag.Len() < 2 {
						continue
					}
					// Parse the 'a' tag value: kind:pubkey:d-tag
					split := bytes.Split(aTag.Value(), []byte{':'})
					if len(split) != 3 {
						continue
					}
					// Parse the kind
					kindStr := string(split[0])
					kindInt, err := strconv.Atoi(kindStr)
					if err != nil {
						continue
					}
					kk := kind.New(uint16(kindInt))
					// Only process parameterized replaceable events
					if !kk.IsParameterizedReplaceable() {
						continue
					}
					// Parse the pubkey
					var pk []byte
					if pk, err = hex.DecAppend(nil, split[1]); err != nil {
						continue
					}
					// Only allow users to delete their own events
					if !bytes.Equal(pk, ev.Pubkey) {
						continue
					}
					// Create the key for the deletion map using hex
					// representation of pubkey
					key := hex.Enc(pk) + ":" + strconv.Itoa(int(kk.K))
					// Initialize the inner map if it doesn't exist
					if _, exists := deletionsByKindPubkeyDTag[key]; !exists {
						deletionsByKindPubkeyDTag[key] = make(map[string]bool)
					}
					// Mark this d-tag as deleted
					dValue := string(split[2])
					deletionsByKindPubkeyDTag[key][dValue] = true
					// Debug logging
					log.D.C(
						func() string {
							return fmt.Sprintf(
								"processing a-tag: %s\n", string(aTag.Value()),
							)
						},
					)
					log.D.C(
						func() string {
							return fmt.Sprintf(
								"adding to deletion map - key: %s, d-tag: %s\n",
								key, dValue,
							)
						},
					)
				}
				// For replaceable events, we need to check if there are any
				// e-tags that reference events with the same kind and pubkey
				for _, eTag := range eTags.ToSliceOfTags() {
					if eTag.Len() < 2 {
						continue
					}
					// Get the event ID from the e-tag
					evId := make([]byte, sha256.Size)
					if _, err = hex.DecBytes(evId, eTag.Value()); err != nil {
						continue
					}
					// Query for the event
					var targetEvs event.S
					targetEvs, err = d.QueryEvents(
						c, &filter.F{Ids: tag.New(evId)},
					)
					if err != nil || len(targetEvs) == 0 {
						continue
					}
					targetEv := targetEvs[0]
					// Only allow users to delete their own events
					if !bytes.Equal(targetEv.Pubkey, ev.Pubkey) {
						continue
					}
					// Mark the specific event ID as deleted
					deletedEventIds[hex.Enc(targetEv.ID)] = true
					// If the event is replaceable, mark it as deleted, but only
					// for events older than this one
					if targetEv.Kind.IsReplaceable() {
						key := hex.Enc(targetEv.Pubkey) + ":" + strconv.Itoa(int(targetEv.Kind.K))
						// We will still use deletionsByKindPubkey, but we'll
						// check timestamps in the second pass
						deletionsByKindPubkey[key] = true
					} else if targetEv.Kind.IsParameterizedReplaceable() {
						// For parameterized replaceable events, we need to
						// consider the 'd' tag
						key := hex.Enc(targetEv.Pubkey) + ":" + strconv.Itoa(int(targetEv.Kind.K))

						// Get the 'd' tag value
						dTag := targetEv.Tags.GetFirst(tag.New([]byte{'d'}))
						var dValue string
						if dTag != nil && dTag.Len() > 1 {
							dValue = string(dTag.Value())
						} else {
							// If no 'd' tag, use empty string
							dValue = ""
						}
						// Initialize the inner map if it doesn't exist
						if _, exists := deletionsByKindPubkeyDTag[key]; !exists {
							deletionsByKindPubkeyDTag[key] = make(map[string]bool)
						}
						// Mark this d-tag as deleted
						deletionsByKindPubkeyDTag[key][dValue] = true
					}
				}
			}
		}

		// Second pass: process all events, filtering out deleted ones
		for _, idpk := range idPkTs {
			var ev *event.E
			ser := new(types.Uint40)
			if err = ser.Set(idpk.Ser); chk.E(err) {
				continue
			}
			if ev, err = d.FetchEventBySerial(ser); err != nil {
				continue
			}
			// Skip events with kind 5 (Deletion)
			if ev.Kind.Equal(kind.Deletion) {
				continue
			}
			// Check if this event's ID is in the filter
			isIdInFilter := false
			if f.Ids != nil && f.Ids.Len() > 0 {
				for i := 0; i < f.Ids.Len(); i++ {
					if bytes.Equal(ev.ID, f.Ids.B(i)) {
						isIdInFilter = true
						break
					}
				}
			}
			// Check if this specific event has been deleted
			eventIdHex := hex.Enc(ev.ID)
			if deletedEventIds[eventIdHex] && !isIdInFilter {
				// Skip this event if it has been specifically deleted and is
				// not in the filter
				continue
			}
			if ev.Kind.IsReplaceable() {
				// For replaceable events, we only keep the latest version for
				// each pubkey and kind, and only if it hasn't been deleted
				key := hex.Enc(ev.Pubkey) + ":" + strconv.Itoa(int(ev.Kind.K))
				// For replaceable events, we need to be more careful with
				// deletion Only skip this event if it has been deleted by
				// kind/pubkey and is not in the filter AND there isn't a newer
				// event with the same kind/pubkey
				if deletionsByKindPubkey[key] && !isIdInFilter {
					// Check if there's a newer event with the same kind/pubkey
					// that hasn't been specifically deleted
					existing, exists := replaceableEvents[key]
					if !exists || ev.CreatedAt.I64() > existing.CreatedAt.I64() {
						// This is the newest event so far, keep it
						replaceableEvents[key] = ev
					} else {
						// There's a newer event, skip this one
						continue
					}
				} else {
					// Normal replaceable event handling
					existing, exists := replaceableEvents[key]
					if !exists || ev.CreatedAt.I64() > existing.CreatedAt.I64() {
						replaceableEvents[key] = ev
					}
				}
			} else if ev.Kind.IsParameterizedReplaceable() {
				// For parameterized replaceable events, we need to consider the
				// 'd' tag
				key := hex.Enc(ev.Pubkey) + ":" + strconv.Itoa(int(ev.Kind.K))

				// Get the 'd' tag value
				dTag := ev.Tags.GetFirst(tag.New([]byte{'d'}))
				var dValue string
				if dTag != nil && dTag.Len() > 1 {
					dValue = string(dTag.Value())
				} else {
					// If no 'd' tag, use empty string
					dValue = ""
				}

				// Check if this event has been deleted via an a-tag
				if deletionMap, exists := deletionsByKindPubkeyDTag[key]; exists {
					// Debug logging
					log.T.C(
						func() string {
							return fmt.Sprintf(
								"Checking deletion map - key: %s, d-tag: %s",
								key, dValue,
							)
						},
					)
					log.T.C(
						func() string {
							return fmt.Sprintf(
								"Deletion map contains key: %v, d-tag in map: %v",
								exists, deletionMap[dValue],
							)
						},
					)
					// If the d-tag value is in the deletion map and this event
					// is not specifically requested by ID, skip it
					if deletionMap[dValue] && !isIdInFilter {
						log.T.F("Debug: Event deleted - skipping")
						continue
					}
				}

				// Initialize the inner map if it doesn't exist
				if _, exists := paramReplaceableEvents[key]; !exists {
					paramReplaceableEvents[key] = make(map[string]*event.E)
				}

				// Check if we already have an event with this 'd' tag value
				existing, exists := paramReplaceableEvents[key][dValue]
				// Only keep the newer event, regardless of processing order
				if !exists {
					// No existing event, add this one
					paramReplaceableEvents[key][dValue] = ev
				} else if ev.CreatedAt.I64() > existing.CreatedAt.I64() {
					// This event is newer than the existing one, replace it
					paramReplaceableEvents[key][dValue] = ev
				}
				// If this event is older than the existing one, ignore it
			} else {
				// Regular events
				regularEvents = append(regularEvents, ev)
			}
		}
		// Add all the latest replaceable events to the result
		for _, ev := range replaceableEvents {
			evs = append(evs, ev)
		}

		// Add all the latest parameterized replaceable events to the result
		for _, innerMap := range paramReplaceableEvents {
			for _, ev := range innerMap {
				evs = append(evs, ev)
			}
		}
		// Add all regular events to the result
		evs = append(evs, regularEvents...)
		// Sort all events by timestamp (newest first)
		sort.Slice(
			evs, func(i, j int) bool {
				return evs[i].CreatedAt.I64() > evs[j].CreatedAt.I64()
			},
		)
		// delete the expired events in a background thread
		go func() {
			for i, ser := range expDeletes {
				if err = d.DeleteEventBySerial(c, ser, expEvs[i]); chk.E(err) {
					continue
				}
			}
		}()
	}
	return
}
