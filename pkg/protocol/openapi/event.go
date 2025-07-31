package openapi

import (
	"bytes"
	"fmt"
	"github.com/danielgtaylor/huma/v2"
	"net/http"
	"orly.dev/pkg/app/relay/helpers"
	"orly.dev/pkg/crypto/sha256"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/eventid"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/encoders/ints"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"strings"
)

var EventBody = &huma.RequestBody{
	Description: "a signed nostr event",
	Content: map[string]*huma.MediaType{
		"application/json": {
			Schema: &huma.Schema{
				Type: huma.TypeObject,
				Properties: map[string]*huma.Schema{
					"id": {
						Type:        huma.TypeString,
						Description: "SHA256 hash of event in canonical (JSON array with fixed ordering) form, 64 characters hex",
					},
					"pubkey": {
						Type:        huma.TypeString,
						Description: "BIP-340 Schnorr public key, 64 characters hex",
					},
					"created_at": {
						Type:        huma.TypeInteger,
						Description: "unix timestamp of time of event creation",
					},
					"kind": {
						Type:        huma.TypeInteger,
						Description: "kind number of event",
					},
					"tags": {
						Type:        huma.TypeArray,
						Description: "array of arrays of strings",
						Items: &huma.Schema{
							Type: huma.TypeArray,
							Items: &huma.Schema{
								Type: huma.TypeString,
							},
						},
					},
					"content": {
						Type:        huma.TypeString,
						Description: "content of event, escaped using NIP-01 standard escapes and UTF-8 encoding",
					},
					"sig": {
						Type:        huma.TypeString,
						Description: "BIP-340 Schnorr signature, 128 characters hex",
					},
				},
			},
		},
	},
}

// EventInput is the parameters for the Event HTTP API method.
type EventInput struct {
	Auth   string   `header:"Authorization" doc:"nostr nip-98 (and expiring variant)" required:"false"`
	Accept string   `header:"Accept" default:"application/nostr+json"`
	Body   *event.J `doc:"event JSON"`
}

// EventOutput is the return parameters for the HTTP API Event method.
type EventOutput struct{}

// RegisterEvent is the implementation of the HTTP API Event method.
func (x *Operations) RegisterEvent(api huma.API) {
	name := "Event"
	description := "Submit an event"
	path := x.path + "/event"
	scopes := []string{"user", "write"}
	method := http.MethodPost
	huma.Register(
		api, huma.Operation{
			OperationID: name,
			Summary:     name,
			Path:        path,
			Method:      method,
			Tags:        []string{"events"},
			RequestBody: EventBody,
			Description: helpers.GenerateDescription(description, scopes),
			Security:    []map[string][]string{{"auth": scopes}},
		}, func(ctx context.T, input *EventInput) (
			output *EventOutput, err error,
		) {
			r := ctx.Value("http-request").(*http.Request)
			remote := helpers.GetRemoteFromReq(r)

			var authed, super bool

			var pubkey []byte
			if x.I.AuthRequired() {
				authed, pubkey, super = x.UserAuth(r, remote)
				if !authed {
					err = huma.Error401Unauthorized("Not Authorized")
					return
				}
			}
			// get the other pubkeys from the header that will be sent forward
			// by the other replicas to avoid repeatedly sending the message
			// back to replicas that already received and forwarded it.
			pubkeyHeader := r.Header.Get("X-Pubkeys")
			pubkeys := [][]byte{pubkey}
			if strings.Contains(pubkeyHeader, ":") {
				split := strings.Split(pubkeyHeader, ":")
				for _, pk := range split {
					var pkb []byte
					if pkb, err = hex.Dec(pk); chk.E(err) {
						continue
					}
					pubkeys = append(pubkeys, pkb)
				}
			}
			var ev *event.E
			if ev, err = input.Body.ToEvent(); chk.E(err) {
				err = huma.Error422UnprocessableEntity(
					"Failed to convert event", err,
				)
				return
			}
			// these aliases make it so most of the following code can be copied
			// verbatim from its counterpart in socketapi.HandleEvent, with the
			// aid of a different implementation of the openapi.OK type.
			a := x
			env := ev
			c := x.Context()
			calculatedId := ev.GetIDBytes()
			if !bytes.Equal(calculatedId, ev.ID) {
				err = huma.Error422UnprocessableEntity(
					Ok.Invalid(
						a, env, "event id is computed incorrectly, "+
							"event has ID %0x, but when computed it is %0x",
						ev.ID, calculatedId,
					).Error(),
				)
				return
			}
			var ok bool
			if ok, err = env.Verify(); chk.T(err) {
				if err = Ok.Error(
					a, env, fmt.Sprintf(
						"failed to verify signature: %s",
						err.Error(),
					),
				); chk.E(err) {
					return
				}
			} else if !ok {
				if err = Ok.Invalid(
					a, env,
					"signature is invalid",
				); chk.E(err) {
					return
				}
				return
			}
			// check that relay policy allows this event
			accept, notice, _ := x.I.AcceptEvent(c, env, r, pubkey, remote)
			if !accept && !super {
				if err = Ok.Blocked(
					a, env, notice,
				); chk.E(err) {
					return
				}
				return
			}
			// check for protected tag (NIP-70)
			protectedTag := ev.Tags.GetFirst(tag.New("-"))
			// if the super flag was set protected is ignored because the relay
			// cluster replicas must replicate this event (and all events).
			if protectedTag != nil && a.AuthRequired() && !super {
				// check that the pubkey of the event matches the authed pubkey
				if !bytes.Equal(pubkey, ev.Pubkey) {
					if err = Ok.Blocked(
						a, env,
						"protected tag may only be published by user authed to the same pubkey",
					); chk.E(err) {
						return
					}
					return
				}
			}
			// check and process delete
			sto := x.I.Storage()
			if ev.Kind.K == kind.Deletion.K {
				log.I.F("delete event\n%s", ev.Serialize())
				for _, t := range ev.Tags.ToSliceOfTags() {
					var res []*event.E
					if t.Len() >= 2 {
						switch {
						case bytes.Equal(t.Key(), []byte("e")):
							// Process 'e' tag (event reference)
							eventId := make([]byte, sha256.Size)
							if _, err = hex.DecBytes(
								eventId, t.Value(),
							); chk.E(err) {
								return
							}

							// Create a filter to find the referenced event
							f := filter.New()
							f.Ids = f.Ids.Append(eventId)

							// Query for the referenced event
							var referencedEvents []*event.E
							referencedEvents, err = sto.QueryEvents(c, f)
							if chk.E(err) {
								if err = Ok.Error(
									a, env,
									"failed to query for referenced event",
								); chk.E(err) {
									return
								}
								return
							}

							// If we found the referenced event, check if the
							// author matches
							if len(referencedEvents) > 0 {
								referencedEvent := referencedEvents[0]

								// Check if the author of the deletion event
								// matches the author of the referenced event
								if !bytes.Equal(
									referencedEvent.Pubkey, env.Pubkey,
								) {
									if err = Ok.Blocked(
										a, env,
										"blocked: cannot delete events from other authors",
									); chk.E(err) {
										return
									}
									return
								}

								// Create eventid.T from the event ID bytes
								var eid *eventid.T
								if eid, err = eventid.NewFromBytes(eventId); chk.E(err) {
									if err = Ok.Error(
										a, env, "failed to create event ID",
									); chk.E(err) {
										return
									}
									return
								}

								// Use DeleteEvent to actually delete the
								// referenced event
								if err = sto.DeleteEvent(c, eid); chk.E(err) {
									if err = Ok.Error(
										a, env,
										"failed to delete referenced event",
									); chk.E(err) {
										return
									}
									return
								}

								log.I.F(
									"successfully deleted event %x", eventId,
								)
							}
						case bytes.Equal(t.Key(), []byte("a")):
							split := bytes.Split(t.Value(), []byte{':'})
							if len(split) != 3 {
								continue
							}
							var pk []byte
							if pk, err = hex.DecAppend(
								nil, split[1],
							); chk.E(err) {
								if err = Ok.Invalid(
									a, env,
									"delete event a tag pubkey value invalid: %s",
									t.Value(),
								); chk.E(err) {
									return
								}
								return
							}
							kin := ints.New(uint16(0))
							if _, err = kin.Unmarshal(split[0]); chk.E(err) {
								if err = Ok.Invalid(
									a, env, "delete event a tag kind value "+
										"invalid: %s",
									t.Value(),
								); chk.E(err) {
									return
								}
								return
							}
							kk := kind.New(kin.Uint16())
							if kk.Equal(kind.Deletion) {
								if err = Ok.Blocked(
									a, env, "delete event kind may not be "+
										"deleted",
								); chk.E(err) {
									return
								}
								return
							}
							if !kk.IsParameterizedReplaceable() {
								if err = Ok.Error(
									a, env,
									"delete tags with a tags containing "+
										"non-parameterized-replaceable events can't be processed",
								); chk.E(err) {
									return
								}
								return
							}
							if !bytes.Equal(pk, ev.Pubkey) {
								if err = Ok.Blocked(
									a, env,
									"can't delete other users' events (delete by a tag)",
								); chk.E(err) {
									return
								}
								return
							}
							f := filter.New()
							f.Kinds.K = []*kind.T{kk}
							f.Authors.Append(pk)
							f.Tags.AppendTags(
								tag.New(
									[]byte{'#', 'd'}, split[2],
								),
							)
							res, err = sto.QueryEvents(c, f)
							if chk.E(err) {
								if err = Ok.Error(
									a, env, "failed to query for target event",
								); chk.E(err) {
									return
								}
								return
							}
						}
					}
					if len(res) < 1 {
						continue
					}
					var resTmp []*event.E
					for _, v := range res {
						if ev.CreatedAt.U64() >= v.CreatedAt.U64() {
							resTmp = append(resTmp, v)
						}
					}
					res = resTmp
					for _, target := range res {
						if target.Kind.K == kind.Deletion.K {
							if err = Ok.Error(
								a, env, "cannot delete delete event %s", ev.ID,
							); chk.E(err) {
								return
							}
						}
						if target.CreatedAt.Int() > ev.CreatedAt.Int() {
							log.I.F(
								"not deleting\n%d%\nbecause delete event is older\n%d",
								target.CreatedAt.Int(), ev.CreatedAt.Int(),
							)
							continue
						}
						if !bytes.Equal(target.Pubkey, env.Pubkey) {
							if err = Ok.Error(
								a, env, "only author can delete event",
							); chk.E(err) {
								return
							}
							return
						}

						// Create eventid.T from the target event ID bytes
						var eid *eventid.T
						eid, err = eventid.NewFromBytes(target.EventId().Bytes())
						if chk.E(err) {
							if err = Ok.Error(
								a, env, "failed to create event ID",
							); chk.E(err) {
								return
							}
							return
						}

						// Use DeleteEvent to actually delete the target event
						if err = sto.DeleteEvent(c, eid); chk.E(err) {
							if err = Ok.Error(
								a, env, "failed to delete target event",
							); chk.E(err) {
								return
							}
							return
						}

						log.I.F(
							"successfully deleted event %x",
							target.EventId().Bytes(),
						)
					}
					res = nil
				}
				// Send a success response after processing all deletions
				if err = Ok.Ok(a, env, ""); chk.E(err) {
					return
				}
				// Check if this event has been deleted before
				if ev.Kind.K != kind.Deletion.K {
					// Create a filter to check for deletion events that
					// reference this event ID
					f := filter.New()
					f.Kinds.K = []*kind.T{kind.Deletion}
					f.Tags.AppendTags(tag.New([]byte{'e'}, ev.ID))

					// Query for deletion events
					var deletionEvents []*event.E
					deletionEvents, err = sto.QueryEvents(c, f)
					if err == nil && len(deletionEvents) > 0 {
						// Found deletion events for this ID, don't save it
						if err = Ok.Blocked(
							a, env,
							"the event was deleted, not storing it again",
						); chk.E(err) {
							return
						}
						return
					}
				}
			}
			var reason []byte
			ok, reason = x.I.AddEvent(
				c, x.Relay(), ev, r, remote, pubkeys,
			)
			log.I.F("http API event %0x added %v %s", ev.ID, ok, reason)
			if !ok && err != nil {
				if err = Ok.Error(
					a, env, err.Error(),
				); chk.E(err) {
					return
				}
			}
			return
		},
	)
}
