package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/alexej-v/grpc_cli/certs"

	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

var (
	defaultCli Client
	once       sync.Once
)

const (
	rpcNameDelimiter = "."
)

type Client interface {
	Headers() Headers
	Invoke(ctx context.Context, fqrn string, req, resp interface{}) error
}

type client struct {
	conn    *grpc.ClientConn
	headers Headers

	client *grpcreflect.Client
}

type ClientCfg struct {
	Addr          string
	ServerName    string
	UseReflection bool
	WithTLS       bool
	Certs         certs.Certs
}

func NewClient(cfg *ClientCfg) (cli Client, err error) {
	var opts []grpc.DialOption
	var tlsCfg tls.Config
	var conn *grpc.ClientConn

	if !cfg.WithTLS {
		opts = append(opts, grpc.WithInsecure())
	} else {
		if cfg.Certs.HasCaCert() {
			tlsCfg.RootCAs = cfg.Certs.CACert()
		}
		if cfg.Certs.HasCert() {
			tlsCfg.Certificates = append(tlsCfg.Certificates, cfg.Certs.Cert())
		}
		creds := credentials.NewTLS(&tlsCfg)
		if cfg.ServerName != "" {
			if err = creds.OverrideServerName(cfg.ServerName); err != nil {
				return nil, errors.Wrap(err, "failed to override server name")
			}
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10*time.Second))
	defer cancel()

	if conn, err = grpc.DialContext(ctx, cfg.Addr, opts...); err != nil {
		return nil, errors.Wrap(err, "failed to dial to gRPC server")
	}
	newClient := &client{conn: conn, headers: Headers{}}
	if cfg.UseReflection {
		newClient.client = grpcreflect.NewClient(
			context.Background(), grpc_reflection_v1alpha.NewServerReflectionClient(conn),
		)
	}
	return newClient, nil
}

func NewClientOnce(cfg *ClientCfg) (cli Client, err error) {
	once.Do(func() {
		defaultCli, err = NewClient(cfg)
	})
	return defaultCli, err
}

func (c *client) Headers() Headers {
	return c.headers
}

func (c *client) Invoke(ctx context.Context, fqrn string, req, resp interface{}) error {
	method, err := fullQualifiedRPCNameToMethod(fqrn)
	if err != nil {
		return err
	}
	// logRequest(req)
	connectBackOff(c.conn)
	return c.conn.Invoke(ctx, method, req, resp)
}

func fullQualifiedRPCNameToMethod(name string) (string, error) {
	spName := strings.Split(name, rpcNameDelimiter)
	if len(spName) < 3 {
		return "", errors.New("invalid FullQualifiedRPCName format")
	}
	return fmt.Sprintf(
		"/%s/%s", strings.Join(spName[:len(spName)-1], rpcNameDelimiter), spName[len(spName)-1],
	), nil
}

func connectBackOff(conn *grpc.ClientConn) {
	if conn == nil {
		return
	}
	if conn.GetState() == connectivity.TransientFailure {
		conn.ResetConnectBackoff()
	}
}

func logRequest(req interface{}) {
	b, err := json.MarshalIndent(&req, "", "  ")
	if err != nil {
		return
	}
	log.Printf("request:\n%s", b)
}
