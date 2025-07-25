package openapi

import (
	"errors"
	"github.com/danielgtaylor/huma/v2"
	"github.com/dgraph-io/badger/v4"
	"math"
	"net/http"
	"orly.dev/pkg/app/relay/helpers"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/filters"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/protocol/auth"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"orly.dev/pkg/utils/pointers"
)

type Filter struct {
	Ids     []string `json:"ids,omitempty"`
	Kinds   []int    `json:"kinds,omitempty"`
	Authors []string `json:"authors,omitempty"`
	Tag_a   []string `json:"#a,omitempty"`
	Tag_b   []string `json:"#b,omitempty"`
	Tag_c   []string `json:"#c,omitempty"`
	Tag_d   []string `json:"#d,omitempty"`
	Tag_e   []string `json:"#e,omitempty"`
	Tag_f   []string `json:"#f,omitempty"`
	Tag_g   []string `json:"#g,omitempty"`
	Tag_h   []string `json:"#h,omitempty"`
	Tag_i   []string `json:"#i,omitempty"`
	Tag_j   []string `json:"#j,omitempty"`
	Tag_k   []string `json:"#k,omitempty"`
	Tag_l   []string `json:"#l,omitempty"`
	Tag_m   []string `json:"#m,omitempty"`
	Tag_n   []string `json:"#n,omitempty"`
	Tag_o   []string `json:"#o,omitempty"`
	Tag_p   []string `json:"#p,omitempty"`
	Tag_q   []string `json:"#q,omitempty"`
	Tag_r   []string `json:"#r,omitempty"`
	Tag_s   []string `json:"#s,omitempty"`
	Tag_t   []string `json:"#t,omitempty"`
	Tag_u   []string `json:"#u,omitempty"`
	Tag_v   []string `json:"#v,omitempty"`
	Tag_w   []string `json:"#w,omitempty"`
	Tag_x   []string `json:"#x,omitempty"`
	Tag_y   []string `json:"#y,omitempty"`
	Tag_z   []string `json:"#z,omitempty"`
	Tag_A   []string `json:"#A,omitempty"`
	Tag_B   []string `json:"#B,omitempty"`
	Tag_C   []string `json:"#C,omitempty"`
	Tag_D   []string `json:"#D,omitempty"`
	Tag_E   []string `json:"#E,omitempty"`
	Tag_F   []string `json:"#F,omitempty"`
	Tag_G   []string `json:"#G,omitempty"`
	Tag_H   []string `json:"#H,omitempty"`
	Tag_I   []string `json:"#I,omitempty"`
	Tag_J   []string `json:"#J,omitempty"`
	Tag_K   []string `json:"#K,omitempty"`
	Tag_L   []string `json:"#L,omitempty"`
	Tag_M   []string `json:"#M,omitempty"`
	Tag_N   []string `json:"#N,omitempty"`
	Tag_O   []string `json:"#O,omitempty"`
	Tag_P   []string `json:"#P,omitempty"`
	Tag_Q   []string `json:"#Q,omitempty"`
	Tag_R   []string `json:"#R,omitempty"`
	Tag_S   []string `json:"#S,omitempty"`
	Tag_T   []string `json:"#T,omitempty"`
	Tag_U   []string `json:"#U,omitempty"`
	Tag_V   []string `json:"#V,omitempty"`
	Tag_W   []string `json:"#W,omitempty"`
	Tag_X   []string `json:"#X,omitempty"`
	Tag_Y   []string `json:"#Y,omitempty"`
	Tag_Z   []string `json:"#Z,omitempty"`
	Since   *int64   `json:"since,omitempty"`
	Until   *int64   `json:"until,omitempty"`
	Search  *string  `json:"search,omitempty"`
	Limit   *int     `json:"limit,omitempty"`
}

func (f *Filter) ToFilter() (ff *filter.F) {
	ff = filter.New()

	// Convert Ids
	if f.Ids != nil && len(f.Ids) > 0 {
		for _, id := range f.Ids {
			ff.Ids.Append([]byte(id))
		}
	}
	if f.Kinds != nil && len(f.Kinds) > 0 {
		for _, k := range f.Kinds {
			ff.Kinds.K = append(ff.Kinds.K, kind.New(uint16(k)))
		}
	}
	if f.Authors != nil && len(f.Authors) > 0 {
		for _, author := range f.Authors {
			ff.Authors.Append([]byte(author))
		}
	}
	if f.Since != nil {
		ts := timestamp.New(*f.Since)
		ff.Since = ts
	}
	if f.Until != nil {
		ts := timestamp.New(*f.Until)
		ff.Until = ts
	}
	if f.Search != nil {
		ff.Search = []byte(*f.Search)
	}
	if f.Limit != nil {
		u := uint(*f.Limit)
		ff.Limit = &u
	}
	if f.Tag_a != nil && len(f.Tag_a) > 0 {
		t := tag.New("#a")
		for _, v := range f.Tag_a {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_b != nil && len(f.Tag_b) > 0 {
		t := tag.New("#b")
		for _, v := range f.Tag_b {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_c != nil && len(f.Tag_c) > 0 {
		t := tag.New("#c")
		for _, v := range f.Tag_c {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_d != nil && len(f.Tag_d) > 0 {
		t := tag.New("#d")
		for _, v := range f.Tag_d {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_e != nil && len(f.Tag_e) > 0 {
		t := tag.New("#e")
		for _, v := range f.Tag_e {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_f != nil && len(f.Tag_f) > 0 {
		t := tag.New("#f")
		for _, v := range f.Tag_f {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_g != nil && len(f.Tag_g) > 0 {
		t := tag.New("#g")
		for _, v := range f.Tag_g {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_h != nil && len(f.Tag_h) > 0 {
		t := tag.New("#h")
		for _, v := range f.Tag_h {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_i != nil && len(f.Tag_i) > 0 {
		t := tag.New("#i")
		for _, v := range f.Tag_i {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_j != nil && len(f.Tag_j) > 0 {
		t := tag.New("#j")
		for _, v := range f.Tag_j {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_k != nil && len(f.Tag_k) > 0 {
		t := tag.New("#k")
		for _, v := range f.Tag_k {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_l != nil && len(f.Tag_l) > 0 {
		t := tag.New("#l")
		for _, v := range f.Tag_l {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_m != nil && len(f.Tag_m) > 0 {
		t := tag.New("#m")
		for _, v := range f.Tag_m {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_n != nil && len(f.Tag_n) > 0 {
		t := tag.New("#n")
		for _, v := range f.Tag_n {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_o != nil && len(f.Tag_o) > 0 {
		t := tag.New("#o")
		for _, v := range f.Tag_o {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_p != nil && len(f.Tag_p) > 0 {
		t := tag.New("#p")
		for _, v := range f.Tag_p {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_q != nil && len(f.Tag_q) > 0 {
		t := tag.New("#q")
		for _, v := range f.Tag_q {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_r != nil && len(f.Tag_r) > 0 {
		t := tag.New("#r")
		for _, v := range f.Tag_r {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_s != nil && len(f.Tag_s) > 0 {
		t := tag.New("#s")
		for _, v := range f.Tag_s {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_t != nil && len(f.Tag_t) > 0 {
		t := tag.New("#t")
		for _, v := range f.Tag_t {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_u != nil && len(f.Tag_u) > 0 {
		t := tag.New("#u")
		for _, v := range f.Tag_u {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_v != nil && len(f.Tag_v) > 0 {
		t := tag.New("#v")
		for _, v := range f.Tag_v {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_w != nil && len(f.Tag_w) > 0 {
		t := tag.New("#w")
		for _, v := range f.Tag_w {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_x != nil && len(f.Tag_x) > 0 {
		t := tag.New("#x")
		for _, v := range f.Tag_x {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_y != nil && len(f.Tag_y) > 0 {
		t := tag.New("#y")
		for _, v := range f.Tag_y {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_z != nil && len(f.Tag_z) > 0 {
		t := tag.New("#z")
		for _, v := range f.Tag_z {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_A != nil && len(f.Tag_A) > 0 {
		t := tag.New("#A")
		for _, v := range f.Tag_A {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_B != nil && len(f.Tag_B) > 0 {
		t := tag.New("#B")
		for _, v := range f.Tag_B {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_C != nil && len(f.Tag_C) > 0 {
		t := tag.New("#C")
		for _, v := range f.Tag_C {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_D != nil && len(f.Tag_D) > 0 {
		t := tag.New("#D")
		for _, v := range f.Tag_D {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_E != nil && len(f.Tag_E) > 0 {
		t := tag.New("#E")
		for _, v := range f.Tag_E {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_F != nil && len(f.Tag_F) > 0 {
		t := tag.New("#F")
		for _, v := range f.Tag_F {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_G != nil && len(f.Tag_G) > 0 {
		t := tag.New("#G")
		for _, v := range f.Tag_G {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_H != nil && len(f.Tag_H) > 0 {
		t := tag.New("#H")
		for _, v := range f.Tag_H {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_I != nil && len(f.Tag_I) > 0 {
		t := tag.New("#I")
		for _, v := range f.Tag_I {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_J != nil && len(f.Tag_J) > 0 {
		t := tag.New("#J")
		for _, v := range f.Tag_J {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_K != nil && len(f.Tag_K) > 0 {
		t := tag.New("#K")
		for _, v := range f.Tag_K {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_L != nil && len(f.Tag_L) > 0 {
		t := tag.New("#L")
		for _, v := range f.Tag_L {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_M != nil && len(f.Tag_M) > 0 {
		t := tag.New("#M")
		for _, v := range f.Tag_M {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_N != nil && len(f.Tag_N) > 0 {
		t := tag.New("#N")
		for _, v := range f.Tag_N {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_O != nil && len(f.Tag_O) > 0 {
		t := tag.New("#O")
		for _, v := range f.Tag_O {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_P != nil && len(f.Tag_P) > 0 {
		t := tag.New("#P")
		for _, v := range f.Tag_P {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_Q != nil && len(f.Tag_Q) > 0 {
		t := tag.New("#Q")
		for _, v := range f.Tag_Q {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_R != nil && len(f.Tag_R) > 0 {
		t := tag.New("#R")
		for _, v := range f.Tag_R {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_S != nil && len(f.Tag_S) > 0 {
		t := tag.New("#S")
		for _, v := range f.Tag_S {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_T != nil && len(f.Tag_T) > 0 {
		t := tag.New("#T")
		for _, v := range f.Tag_T {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_U != nil && len(f.Tag_U) > 0 {
		t := tag.New("#U")
		for _, v := range f.Tag_U {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_V != nil && len(f.Tag_V) > 0 {
		t := tag.New("#V")
		for _, v := range f.Tag_V {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_W != nil && len(f.Tag_W) > 0 {
		t := tag.New("#W")
		for _, v := range f.Tag_W {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_X != nil && len(f.Tag_X) > 0 {
		t := tag.New("#X")
		for _, v := range f.Tag_X {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_Y != nil && len(f.Tag_Y) > 0 {
		t := tag.New("#Y")
		for _, v := range f.Tag_Y {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	if f.Tag_Z != nil && len(f.Tag_Z) > 0 {
		t := tag.New("#Z")
		for _, v := range f.Tag_Z {
			t.Append([]byte(v))
		}
		ff.Tags.AppendTags(t)
	}
	return
}

var exampleSince int64 = 1753432853
var exampleUntil int64 = 1753462853
var exampleLimit int = 20
var created_atMinimum float64 = 0
var created_atMaximum float64 = float64(math.MaxInt64)
var limitMinimum float64 = 0
var limitMaximum float64 = float64(math.MaxUint64)

var EventsBody = &huma.RequestBody{
	Description: "array of nostr events",
	Content: map[string]*huma.MediaType{
		"application/json": {
			Schema: &huma.Schema{
				Type: huma.TypeObject,
				Examples: []any{
					Filter{
						Kinds: []int{0, 1},
						Authors: []string{
							"deadbeefcafe8008deadbeefcafe8008deadbeefcafe8008deadbeefcafe8008",
							"deadbeefcafe8008deadbeefcafe8008deadbeefcafe8008deadbeefcafe8008",
						},
						Tag_e: []string{
							"deadbeefcafe8008deadbeefcafe8008deadbeefcafe8008deadbeefcafe8008",
							"deadbeefcafe8008deadbeefcafe8008deadbeefcafe8008deadbeefcafe8008",
							"deadbeefcafe8008deadbeefcafe8008deadbeefcafe8008deadbeefcafe8008",
							"deadbeefcafe8008deadbeefcafe8008deadbeefcafe8008deadbeefcafe8008",
						},
						Since: &exampleSince,
						Until: &exampleUntil,
						Limit: &exampleLimit,
					},
					Filter{
						Ids: []string{
							"deadbeefcafe8008deadbeefcafe8008deadbeefcafe8008deadbeefcafe8008",
							"deadbeefcafe8008deadbeefcafe8008deadbeefcafe8008deadbeefcafe8008",
						},
					},
				},
				Properties: map[string]*huma.Schema{
					"ids": {
						Type:        huma.TypeArray,
						Description: "list of event IDs to search for (if present, all other fields are excluded)",
						Items: &huma.Schema{
							Type: huma.TypeString,
						},
					},
					"kinds": {
						Type:        huma.TypeArray,
						Description: "list of event kinds to search for",
						Items: &huma.Schema{
							Type: huma.TypeInteger,
						},
					},
					"authors": {
						Type:        huma.TypeArray,
						Description: "list of pubkeys to search for",
						Items: &huma.Schema{
							Type: huma.TypeString,
						},
					},
					"^#[a-zA-Z]$": {
						Type:        huma.TypeArray,
						Description: "list of tag values to search for",
						Items: &huma.Schema{
							Type: huma.TypeString,
						},
					},
					"since": {
						Type:        huma.TypeInteger,
						Description: "earliest (smallest, inclusive) created_at value for events",
						Minimum:     &created_atMinimum,
						Maximum:     &created_atMaximum,
					},
					"until": {
						Type:        huma.TypeInteger,
						Description: "latest (largest, inclusive) created_at value for events",
						Minimum:     &created_atMinimum,
						Maximum:     &created_atMaximum,
					},
					"limit": {
						Type:        huma.TypeInteger,
						Description: "maximum number of events to return (newest first, reverse chronological order)",
						Minimum:     &limitMinimum,
						Maximum:     &limitMaximum,
					},
				},
			},
		},
	},
}

type EventsInput struct {
	Auth   string  `header:"Authorization" doc:"nostr nip-98 (and expiring variant)" required:"false"`
	Accept string  `header:"Accept" default:"application/nostr+json"`
	Body   *Filter `doc:"filter JSON (standard NIP-01 filter syntax)"`
}

type EventsOutput struct {
	Body []event.J
}

// RegisterEvents is the implementation of the HTTP API Events method.
//
// This method returns the results of a single filter query, filtered by
// privilege.
func (x *Operations) RegisterEvents(api huma.API) {
	name := "Events"
	description := "query for events, returns raw binary data containing the events in JSON line-structured format (only allows one filter)"
	path := x.path + "/events"
	scopes := []string{"user", "read"}
	method := http.MethodPost
	huma.Register(
		api, huma.Operation{
			OperationID: name,
			Summary:     name,
			Path:        path,
			Method:      method,
			Tags:        []string{"events"},
			RequestBody: EventsBody,
			Description: helpers.GenerateDescription(description, scopes),
			Security:    []map[string][]string{{"auth": scopes}},
		}, func(ctx context.T, input *EventsInput) (
			output *EventsOutput, err error,
		) {
			r := ctx.Value("http-request").(*http.Request)
			remote := helpers.GetRemoteFromReq(r)
			var authed bool
			var pubkey []byte
			// if auth is required and not public readable, the request is not
			// authorized.
			if x.I.AuthRequired() && !x.I.PublicReadable() {
				authed, pubkey = x.UserAuth(r, remote)
				if !authed {
					err = huma.Error401Unauthorized("Not Authorized")
					return
				}
			}
			f := filter.New()
			var rem []byte
			log.I.S(input)
			if len(rem) > 0 {
				log.I.F("extra '%s'", rem)
			}
			var accept bool
			allowed, accept, _ := x.AcceptReq(
				x.Context(), r, filters.New(f), pubkey, remote,
			)
			if !accept {
				err = huma.Error401Unauthorized("Not Authorized for query")
				return
			}
			var events event.S
			for _, ff := range allowed.F {
				// var i uint
				if pointers.Present(ff.Limit) {
					if *ff.Limit == 0 {
						continue
					}
				}
				if events, err = x.Storage().QueryEvents(
					x.Context(), ff,
				); err != nil {
					if errors.Is(err, badger.ErrDBClosed) {
						return
					}
					continue
				}
				// filter events the authed pubkey is not privileged to fetch.
				if x.AuthRequired() && len(pubkey) > 0 {
					var tmp event.S
					for _, ev := range events {
						if !auth.CheckPrivilege(pubkey, ev) {
							log.W.F(
								"not privileged: client pubkey '%0x' event pubkey '%0x' kind %s privileged: %v",
								pubkey, ev.Pubkey,
								ev.Kind.Name(),
								ev.Kind.IsPrivileged(),
							)
							continue
						}
						tmp = append(tmp, ev)
					}
					events = tmp
				}
			}
			for _, ev := range events {
				_ = ev
			}
			return
		},
	)
}
