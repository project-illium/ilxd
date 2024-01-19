// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"net/http"
)

// AuthenticationTokenKey is the key used in the context to authenticate clients.
// If this is set to anything other than "" in the config, then the server expects
// the client to set a key value in the context metadata to 'AuthenticationToken: cfg.AuthToken'
const AuthenticationTokenKey = "AuthenticationToken"

func newGrpcServer(cfgOpts repo.RPCOptions, rpcCfg *rpc.GrpcServerConfig) (*rpc.GrpcServer, error) {
	i := interceptor{authToken: cfgOpts.GrpcAuthToken}
	opts := []grpc.ServerOption{grpc.StreamInterceptor(i.interceptStreaming), grpc.UnaryInterceptor(i.interceptUnary)}
	creds, err := credentials.NewServerTLSFromFile(cfgOpts.RPCCert, cfgOpts.RPCKey)
	if err != nil {
		return nil, err
	}
	opts = append(opts, grpc.Creds(creds), grpc.MaxSendMsgSize(1000000))
	server := grpc.NewServer(opts...)

	allowAllOrigins := grpcweb.WithOriginFunc(func(origin string) bool {
		return true
	})
	wrappedGrpc := grpcweb.WrapServer(server, allowAllOrigins)

	rpcCfg.Server = server

	handler := func(resp http.ResponseWriter, req *http.Request) {
		if wrappedGrpc.IsGrpcWebRequest(req) || wrappedGrpc.IsAcceptableGrpcCorsRequest(req) {
			wrappedGrpc.ServeHTTP(resp, req)
		} else {
			server.ServeHTTP(resp, req)
		}
	}

	ma, err := multiaddr.NewMultiaddr(cfgOpts.GrpcListener)
	if err != nil {
		return nil, err
	}

	netAddr, err := manet.ToNetAddr(ma)
	if err != nil {
		return nil, err
	}

	httpServer := &http.Server{
		Addr:    netAddr.String(),
		Handler: http.HandlerFunc(handler),
	}

	rpcCfg.HTTPServer = httpServer

	gRPCServer := rpc.NewGrpcServer(rpcCfg)

	go func() {
		if err := httpServer.ListenAndServeTLS(cfgOpts.RPCCert, cfgOpts.RPCKey); err != nil {
			log.WithCaller(true).Error("Err serving gRPC", log.Args("error", err))
		}
	}()
	return gRPCServer, nil
}

type interceptor struct {
	authToken string
}

func (i *interceptor) interceptStreaming(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	p, ok := peer.FromContext(ss.Context())
	if ok {
		log.Trace("Streaming gRPC method invoked", log.ArgsFromMap(map[string]any{
			"method": info.FullMethod,
			"addr":   p.Addr.String(),
		}))
	}

	err := validateAuthenticationToken(ss.Context(), i.authToken)
	if err != nil {
		return err
	}

	err = handler(srv, ss)
	if err != nil && ok {
		st, ok := status.FromError(err)
		if ok {
			log.Error("Streaming gRPC error", log.ArgsFromMap(map[string]any{
				"method":     info.FullMethod,
				"addr":       p.Addr.String(),
				"error code": st.Code().String(),
				"desc":       st.Message(),
			}))
		} else {
			log.Error("Streaming gRPC error", log.ArgsFromMap(map[string]any{
				"method": info.FullMethod,
				"addr":   p.Addr.String(),
				"error":  err,
			}))
		}
	}
	return err
}

func (i *interceptor) interceptUnary(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	p, ok := peer.FromContext(ctx)
	if ok {
		log.Trace("Unary gRPC method invoked", log.ArgsFromMap(map[string]any{
			"method": info.FullMethod,
			"addr":   p.Addr.String(),
		}))
	}

	err = validateAuthenticationToken(ctx, i.authToken)
	if err != nil {
		return nil, err
	}

	resp, err = handler(ctx, req)
	if err != nil && ok {
		st, ok := status.FromError(err)
		if ok {
			log.Error("Unary gRPC error", log.ArgsFromMap(map[string]any{
				"method":     info.FullMethod,
				"addr":       p.Addr.String(),
				"error code": st.Code().String(),
				"desc":       st.Message(),
			}))
		} else {
			log.Error("Unary gRPC error", log.ArgsFromMap(map[string]any{
				"method": info.FullMethod,
				"addr":   p.Addr.String(),
				"error":  err,
			}))
		}
	}
	return resp, err
}

func validateAuthenticationToken(ctx context.Context, authToken string) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if authToken != "" && (!ok || len(md.Get(AuthenticationTokenKey)) == 0 || md.Get(AuthenticationTokenKey)[0] != authToken) {
		return errors.New("invalid authentication token")
	}
	return nil
}
