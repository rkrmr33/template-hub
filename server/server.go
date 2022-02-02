package server

import (
	"context"
	"crypto/tls"
	"embed"
	"fmt"
	gologger "log"
	"net"
	"net/http"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	grpcgw "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/rkrmr33/pkg/log"
	registrypkg "github.com/rkrmr33/template-hub/pkg/api/registry"
	versionpkg "github.com/rkrmr33/template-hub/pkg/api/version"
	"github.com/rkrmr33/template-hub/server/registry"
	"github.com/rkrmr33/template-hub/server/version"
	"github.com/rkrmr33/template-hub/util"
	serverutil "github.com/rkrmr33/template-hub/util/server"
)

type (
	Server interface {
		Start(context.Context) error
	}

	server struct {
		log            *log.Logger
		lisAddr        string
		tlsConf        *tls.Config
		tlsCert        []byte
		rootPath       string
		staticAssetsFs embed.FS
		backoffConfig  wait.Backoff

		envMock []byte

		stopped bool
	}
)

func (s *server) Start(ctx context.Context) error {
	var (
		httpS  *http.Server
		httpsS *http.Server

		lis    net.Listener
		grpcL  net.Listener
		httpL  net.Listener
		httpsL net.Listener
		tlsm   cmux.CMux

		err error
	)

	grpcS := s.newGrpcServer()

	if s.tlsConf == nil {
		httpS = s.newHTTPServer(ctx)
	} else {
		httpS = s.newRedirectServer(ctx)
		httpsS = s.newHTTPServer(ctx)
	}

	// setup root path
	if s.rootPath != "" {
		s.withRootPath(ctx, httpS)

		if httpsS != nil {
			s.withRootPath(ctx, httpsS)
		}
	}

	_ = wait.ExponentialBackoffWithContext(ctx, s.backoffConfig, func() (bool, error) {
		lis, err = net.Listen("tcp", s.lisAddr)
		if err != nil {
			s.log.Errorw("in backoff - failed to bind tcp listener", "err", err)
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return fmt.Errorf("failed to bind tcp listener: %w", err)
	}

	tcpm := cmux.New(lis)

	if s.tlsConf == nil {
		httpL = tcpm.Match(cmux.HTTP1Fast())
		grpcL = tcpm.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	} else {
		// first match http 1.1
		httpL = tcpm.Match(cmux.HTTP1Fast())

		tlsConfig := tls.Config{
			Certificates: s.tlsConf.Certificates,
		}

		// all the rest are tls
		tlsl := tls.NewListener(tcpm.Match(cmux.Any()), &tlsConfig)

		tlsm = cmux.New(tlsl)
		httpsL = tlsm.Match(cmux.HTTP1Fast())
		grpcL = tlsm.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	}

	s.log.Infow("frontiers-server started",
		"address", s.lisAddr,
		"tls", s.tlsConf != nil,
		"rootpath", s.rootPath,
	)

	go func() { s.handleListen("grpc-server", grpcS.Serve(grpcL)) }()
	go func() { s.handleListen("http-server", httpS.Serve(httpL)) }()

	if s.tlsConf != nil {
		go func() { s.handleListen("https-server", httpsS.Serve(httpsL)) }()
		go func() { s.handleListen("tlsm", tlsm.Serve()) }()
	}

	go func() { s.handleListen("tcp-mux", tcpm.Serve()) }()

	<-ctx.Done()

	s.stopped = true

	return lis.Close()
}

// newGrpcServer builds and returns an initialized grpc server
func (s *server) newGrpcServer() *grpc.Server {
	sOpts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(MaxGRPCMessageSize),
		grpc.MaxSendMsgSize(MaxGRPCMessageSize),
		grpc.ConnectionTimeout(ConnectionTimeout),
	}

	// unary middlewares
	sOpts = append(sOpts, grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
		grpc_zap.UnaryServerInterceptor(s.log.Desugar()),
		grpc_prometheus.UnaryServerInterceptor,
	)))

	// stream middlewares
	sOpts = append(sOpts, grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
		grpc_zap.StreamServerInterceptor(s.log.Desugar()),
		grpc_prometheus.StreamServerInterceptor,
	)))

	grpcS := grpc.NewServer(sOpts...)

	// create grpc services
	versionServer := version.NewServer()
	registryServer := registry.NewServer(s.envMock)

	// register grpc services
	versionpkg.RegisterVersionServiceServer(grpcS, versionServer)
	registrypkg.RegisterRegistryServiceServer(grpcS, registryServer)

	reflection.Register(grpcS)
	grpc_prometheus.Register(grpcS)

	return grpcS
}

func (s *server) newHTTPServer(ctx context.Context) *http.Server {
	mux := http.NewServeMux()

	dOpts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxGRPCMessageSize)),
	}

	if s.tlsConf != nil {
		tlsConfig := s.tlsConf.Clone()
		tlsConfig.InsecureSkipVerify = true
		dCreds := credentials.NewTLS(tlsConfig)
		dOpts = append(dOpts, grpc.WithTransportCredentials(dCreds))
	} else {
		dOpts = append(dOpts, grpc.WithInsecure())
	}

	gwMux := grpcgw.NewServeMux(
		grpcgw.WithMarshalerOption(grpcgw.MIMEWildcard, &grpcgw.JSONBuiltin{}),
		grpcgw.WithProtoErrorHandler(grpcgw.DefaultHTTPProtoErrorHandler),
		grpcgw.WithIncomingHeaderMatcher(func(key string) (string, bool) { return key, true }),
	)

	// register all services here
	util.Must(versionpkg.RegisterVersionServiceHandlerFromEndpoint(ctx, gwMux, s.lisAddr, dOpts))
	util.Must(registrypkg.RegisterRegistryServiceHandlerFromEndpoint(ctx, gwMux, s.lisAddr, dOpts))

	// server static content
	mux.Handle("/", s.log.NewMiddleware(serverutil.NewStaticAssetsHandler(s.staticAssetsFs)))

	// serve openapi spec
	mux.Handle("/api", s.log.NewMiddleware(serverutil.NewOpenAPIHandler(s.staticAssetsFs, "/api")))

	// server api endpoints
	mux.Handle("/api/", gwMux)

	// serve readiness endpoints
	mux.HandleFunc("/healthz", s.buildHealthz(ctx))
	mux.HandleFunc("/readyz", s.buildHealthz(ctx))

	return &http.Server{
		Addr:      s.lisAddr,
		Handler:   mux,
		TLSConfig: s.tlsConf,
		ErrorLog:  gologger.New(s.log, "", 0),
	}
}

// newRedirectServer returns an HTTP server which does a 307 redirect to the HTTPS server
func (s *server) newRedirectServer(ctx context.Context) *http.Server {
	addr := fmt.Sprintf("%s/%s", s.lisAddr, s.rootPath)

	mux := http.NewServeMux()

	// serve readiness endpoints on regular path
	mux.HandleFunc("/readyz", s.buildHealthz(ctx))
	mux.HandleFunc("/healthz", s.buildHealthz(ctx))

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		target := "https://" + req.Host

		if s.rootPath != "" {
			target += "/" + s.rootPath
		}

		target += req.URL.Path

		if len(req.URL.RawQuery) > 0 {
			target += "?" + req.URL.RawQuery
		}

		http.Redirect(w, req, target, http.StatusMovedPermanently)
	})

	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}

func (s *server) withRootPath(ctx context.Context, server *http.Server) {
	mux := http.NewServeMux()

	// serve readiness endpoints on regular path
	mux.HandleFunc("/healthz", s.buildHealthz(ctx))
	mux.HandleFunc("/readyz", s.buildHealthz(ctx))

	// move the base handler under the new root path
	mux.Handle("/"+s.rootPath+"/", http.StripPrefix("/"+s.rootPath, server.Handler))

	server.Handler = mux
}

func (s *server) handleListen(name string, err error) {
	if err != nil {
		if s.stopped {
			s.log.Infof("gracefully shutting down: %s: %v", name, err)
		} else {
			s.log.Fatalf("failed listening: %s: %v", name, err)
		}
	} else {
		s.log.Infof("gracefully shutting down: %s: %v", name, err)
	}
}

func (s *server) buildHealthz(ctx context.Context) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	}
}
