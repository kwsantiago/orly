package database

import (
	"bytes"
	"orly.dev/crypto/sha256"
	"orly.dev/database/indexes/types"
	"orly.dev/encoders/event"
	"orly.dev/encoders/filter"
	"orly.dev/encoders/hex"
	"orly.dev/encoders/kind"
	"orly.dev/encoders/tag"
	"orly.dev/interfaces/store"
	"orly.dev/utils/chk"
	"orly.dev/utils/context"
	"sort"
	"strconv"
)

// QueryEvents retrieves events based on the provided filter. If the filter
// contains Ids, it fetches events by those Ids directly, overriding other
// filter criteria. Otherwise, it queries by other filter criteria and fetches
// matching events. Results are returned in reverse chronological order of their
// creation timestamps.
func (d *D) QueryEvents(c context.T, f *filter.F) (evs event.S, err error) {
	// if there is Ids in the query, this overrides anything else
	if f.Ids != nil && f.Ids.Len() > 0 {
		for _, idx := range f.Ids.ToSliceOfBytes() {
			// we know there is only Ids in this, so run the ID query and fetch.
			var ser *types.Uint40
			if ser, err = d.GetSerialById(idx); chk.E(err) {
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

		// First pass: collect all deletion events
		for _, idpk := range idPkTs {
			var ev *event.E
			ser := new(types.Uint40)
			if err = ser.Set(idpk.Ser); chk.E(err) {
				continue
			}
			if ev, err = d.FetchEventBySerial(ser); err != nil {
				continue
			}

			// Process deletion events to build our deletion maps
			if ev.Kind.Equal(kind.Deletion) {
				// Check for 'e' tags that directly reference event IDs
				eTags := ev.Tags.GetAll(tag.New([]byte{'e'}))
				for _, eTag := range eTags.ToSliceOfTags() {
					if eTag.Len() < 2 {
						continue
					}
					// We don't need to do anything with direct event ID
					// references as we'll filter those out in the second pass
				}

				// Check for 'a' tags that reference parameterized replaceable
				// events
				aTags := ev.Tags.GetAll(tag.New([]byte{'a'}))
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

					// Create the key for the deletion map
					key := string(pk) + ":" + strconv.Itoa(int(kk.K))

					// Initialize the inner map if it doesn't exist
					if _, exists := deletionsByKindPubkeyDTag[key]; !exists {
						deletionsByKindPubkeyDTag[key] = make(map[string]bool)
					}

					// Mark this d-tag as deleted
					dValue := string(split[2])
					deletionsByKindPubkeyDTag[key][dValue] = true
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

					// If the event is replaceable, mark it as deleted
					if targetEv.Kind.IsReplaceable() {
						key := string(targetEv.Pubkey) + ":" + strconv.Itoa(int(targetEv.Kind.K))
						deletionsByKindPubkey[key] = true
					} else if targetEv.Kind.IsParameterizedReplaceable() {
						// For parameterized replaceable events, we need to consider the 'd' tag
						key := string(targetEv.Pubkey) + ":" + strconv.Itoa(int(targetEv.Kind.K))

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
					if bytes.Equal(ev.Id, f.Ids.B(i)) {
						isIdInFilter = true
						break
					}
				}
			}

			if ev.Kind.IsReplaceable() {
				// For replaceable events, we only keep the latest version for
				// each pubkey and kind, and only if it hasn't been deleted
				key := string(ev.Pubkey) + ":" + strconv.Itoa(int(ev.Kind.K))

				// Skip this event if it has been deleted and its ID is not in
				// the filter
				if deletionsByKindPubkey[key] && !isIdInFilter {
					continue
				}

				existing, exists := replaceableEvents[key]
				if !exists || ev.CreatedAt.I64() > existing.CreatedAt.I64() {
					replaceableEvents[key] = ev
				}
			} else if ev.Kind.IsParameterizedReplaceable() {
				// For parameterized replaceable events, we need to consider the
				// 'd' tag
				key := string(ev.Pubkey) + ":" + strconv.Itoa(int(ev.Kind.K))

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
					// If the d-tag value is in the deletion map and this event is not
					// specifically requested by ID, skip it
					if deletionMap[dValue] && !isIdInFilter {
						continue
					}
				}

				// Initialize the inner map if it doesn't exist
				if _, exists := paramReplaceableEvents[key]; !exists {
					paramReplaceableEvents[key] = make(map[string]*event.E)
				}

				// Check if we already have an event with this 'd' tag value
				existing, exists := paramReplaceableEvents[key][dValue]
				if !exists || ev.CreatedAt.I64() > existing.CreatedAt.I64() {
					paramReplaceableEvents[key][dValue] = ev
				}
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
	}
	return
}
