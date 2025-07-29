package app

import (
	"golang.org/x/sync/errgroup"
	"net"
	"net/http"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"time"
)

type RunArgs struct {
	Addr  string        `arg:"-l,--listen" default:":https" help:"address to listen at"`
	Conf  string        `arg:"-m,--map" default:"mapping.txt" help:"file with host/backend mapping"`
	Cache string        `arg:"-c,--cachedir" default:"/var/cache/letsencrypt" help:"path to directory to cache key and certificates"`
	HSTS  bool          `arg:"-h,--hsts" help:"add Strict-Transport-Security header"`
	Email string        `arg:"-e,--email" help:"contact email address presented to letsencrypt CA"`
	HTTP  string        `arg:"--http" default:":http" help:"optional address to serve http-to-https redirects and ACME http-01 challenge responses"`
	RTO   time.Duration `arg:"-r,--rto" default:"1m" help:"maximum duration before timing out read of the request"`
	WTO   time.Duration `arg:"-w,--wto" default:"5m" help:"maximum duration before timing out write of the response"`
	Idle  time.Duration `arg:"-i,--idle" help:"how long idle connection is kept before closing (set rto, wto to 0 to use this)"`
	Certs []string      `arg:"--cert,separate" help:"certificates and the domain they match: eg: orly.dev:/path/to/cert - this will indicate to load two, one with extension .key and one with .crt, each expected to be PEM encoded TLS private and public keys, respectively"`
	// Rewrites string        `arg:"-r,--rewrites" default:"rewrites.txt"`
}

func Run(c context.T, args RunArgs) (err error) {
	if args.Cache == "" {
		err = log.E.Err("no cache specified")
		return
	}
	var srv *http.Server
	var httpHandler http.Handler
	if srv, httpHandler, err = SetupServer(args); chk.E(err) {
		return
	}
	srv.ReadHeaderTimeout = 5 * time.Second
	if args.RTO > 0 {
		srv.ReadTimeout = args.RTO
	}
	if args.WTO > 0 {
		srv.WriteTimeout = args.WTO
	}
	group, ctx := errgroup.WithContext(c)
	if args.HTTP != "" {
		httpServer := http.Server{
			Addr:         args.HTTP,
			Handler:      httpHandler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
		group.Go(
			func() (err error) {
				chk.E(httpServer.ListenAndServe())
				return
			},
		)
		group.Go(
			func() error {
				<-ctx.Done()
				ctx, cancel := context.Timeout(
					context.Bg(),
					time.Second,
				)
				defer cancel()
				return httpServer.Shutdown(ctx)
			},
		)
	}
	if srv.ReadTimeout != 0 || srv.WriteTimeout != 0 || args.Idle == 0 {
		group.Go(
			func() (err error) {
				chk.E(srv.ListenAndServeTLS("", ""))
				return
			},
		)
	} else {
		group.Go(
			func() (err error) {
				var ln net.Listener
				if ln, err = net.Listen("tcp", srv.Addr); chk.E(err) {
					return
				}
				defer ln.Close()
				ln = Listener{
					Duration:    args.Idle,
					TCPListener: ln.(*net.TCPListener),
				}
				err = srv.ServeTLS(ln, "", "")
				chk.E(err)
				return
			},
		)
	}
	group.Go(
		func() error {
			<-ctx.Done()
			ctx, cancel := context.Timeout(context.Bg(), time.Second)
			defer cancel()
			return srv.Shutdown(ctx)
		},
	)
	return group.Wait()
}
