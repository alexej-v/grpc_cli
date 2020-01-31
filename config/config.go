package config

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

type Config struct {
	Default  *Default
	Server   *Server
	Input    *Input
	Describe string
	help     bool
}

type Default struct {
	ProtoPath []string
	ProtoFile []string
	Package   string
	Service   string
	Method    string
}

type Server struct {
	Host       string
	Port       string
	Reflection bool
	TLS        bool
	CACert     string
	Cert       string
	CertKey    string
	Name       string
}

type Input struct {
	Body string
}

func (s *Server) Address() (addr string) {
	addr = s.Host
	if s.Port != "" {
		addr = fmt.Sprintf("%s:%s", addr, s.Port)
	}
	return
}

func registerCfg(args []string) (cfg *Config, err error) {
	fs := pflag.NewFlagSet("main", pflag.ContinueOnError)
	fs.SortFlags = false

	cfg = &Config{
		Default: new(Default),
		Server:  new(Server),
		Input:   new(Input),
	}

	fs.StringVar(&cfg.Describe, "desc", "", "describe only")

	fs.StringVar(&cfg.Input.Body, "json", "", "json body")

	fs.StringSliceVar(&cfg.Default.ProtoPath, "path", nil, "proto path")
	fs.StringSliceVar(&cfg.Default.ProtoFile, "file", nil, "proto files path")
	fs.StringVar(&cfg.Default.Package, "package", "nil", "default package")
	fs.StringVar(&cfg.Default.Service, "service", "nil", "default service")
	fs.StringVar(&cfg.Default.Method, "method", "nil", "default service")

	fs.StringVar(&cfg.Server.Host, "host", "localhost", "gRPC server host")
	fs.StringVar(&cfg.Server.Port, "port", "50051", "gRPC server port")
	fs.BoolVar(&cfg.Server.TLS, "tls", false, "use a secure TLS connection")
	fs.StringVar(&cfg.Server.CACert, "cacert", "", "the CA certificate file for verifying the server")
	fs.StringVar(
		&cfg.Server.Cert,
		"cert", "", "the certificate file for mutual TLS auth. it must be provided with --certkey.")
	fs.StringVar(
		&cfg.Server.CertKey,
		"certkey", "", "the private key file for mutual TLS auth. it must be provided with --cert.")
	fs.StringVar(&cfg.Server.Name,
		"servername", "", "override the server name used to verify the hostname (ignored if --tls is disabled)")

	fs.BoolVarP(&cfg.help, "help", "h", false, "display help text and exit")

	if err = fs.Parse(os.Args); err != nil {
		return nil, errors.Wrap(err, "failed to parse command line arguments")
	}

	if cfg.help {
		fmt.Println(fs.FlagUsages())
		os.Exit(0)
	}
	return
}

func Init(args []string) (cfg *Config, err error) {
	return registerCfg(args)
}
