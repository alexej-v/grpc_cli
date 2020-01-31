package proto

import (
	"fmt"
	"log"
	"github.com/alexej-v/grpc_cli/config"
	"github.com/alexej-v/grpc_cli/grpc"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/pkg/errors"
)

var (
	ErrServiceIsEmpty = errors.New("service is an empty string")
	ErrPackageIsEmpty = errors.New("package is an empty string")
	ErrPackageUnknown = errors.New("unknown package name")
	ErrServiceUnknown = errors.New("unknown service name")
	ErrRPCUnknown     = errors.New("unknown RPC name")
)

type Spec interface {
	PackageNames() (pkgNames []string)
	ServiceNames(pkgName string) (svcNames []string, err error)
	RPCs(pkgName, svcName string) ([]*grpc.RPC, error)
	RPC(pkgName, svcName, rpcName string) (*grpc.RPC, error)
	Messages(fqn string)
	Describe(cfg *config.Config)
}

type spec struct {
	pkgNames map[string]struct{}
	// key: package name, val: service descriptors belong to the package.
	svcDescs map[string][]*desc.ServiceDescriptor
	// key: fully qualified service name, val: method descriptors belong to the service.
	rpcDescs map[string][]*desc.MethodDescriptor
	// key: fully qualified message name, val: the message descriptor.
	msgDescs map[string]*desc.MessageDescriptor
}

func Parse(filePath []string, importPath []string) (*spec, error) {
	parser := protoparse.Parser{}
	parser.ImportPaths = importPath
	descriptors, err := parser.ParseFiles(filePath...)
	if err != nil {
		return nil, errors.Wrap(err, "proto: failed to parse proto files")
	}

	pkgNames := make(map[string]struct{})
	svcDescs := make(map[string][]*desc.ServiceDescriptor)
	rpcDescs := make(map[string][]*desc.MethodDescriptor)
	msgDescs := make(map[string]*desc.MessageDescriptor)

	for _, descriptor := range descriptors {
		pkgNames[descriptor.GetPackage()] = struct{}{}
		svcDescs[descriptor.GetPackage()] = append(svcDescs[descriptor.GetPackage()], descriptor.GetServices()...)
		for _, sd := range descriptor.GetServices() {
			rpcDescs[sd.GetFullyQualifiedName()] = append(rpcDescs[sd.GetFullyQualifiedName()], sd.GetMethods()...)
		}
		for _, md := range descriptor.GetMessageTypes() {
			msgDescs[md.GetFullyQualifiedName()] = md
		}
	}
	return &spec{
		pkgNames: pkgNames,
		svcDescs: svcDescs,
		rpcDescs: rpcDescs,
		msgDescs: msgDescs,
	}, nil
}

func (s *spec) PackageNames() (pkgNames []string) {
	pkgNames = make([]string, 0, len(s.pkgNames))
	for pkgName, _ := range s.pkgNames {
		pkgNames = append(pkgNames, pkgName)
	}
	return
}

func (s *spec) ServiceNames(pkgName string) (svcNames []string, err error) {
	if pkgName == "" {
		return nil, ErrPackageIsEmpty
	}

	descs, ok := s.svcDescs[pkgName]
	if !ok {
		return nil, ErrPackageUnknown
	}
	svcNames = make([]string, len(descs))
	for i, d := range descs {
		svcNames[i] = d.GetName()
	}
	return svcNames, nil
}

func (s *spec) RPCs(pkgName, svcName string) ([]*grpc.RPC, error) {
	// Check whether pkgName is a valid package or not.
	_, err := s.ServiceNames(pkgName)
	if err != nil {
		return nil, err
	}

	if svcName == "" {
		return nil, ErrServiceIsEmpty
	}

	fqsn := fmt.Sprintf("%s.%s", pkgName, svcName)
	rpcDescs, ok := s.rpcDescs[fqsn]
	if !ok {
		return nil, ErrServiceUnknown
	}

	rpcs := make([]*grpc.RPC, len(rpcDescs))
	for i, d := range rpcDescs {
		rpc, err := s.RPC(pkgName, svcName, d.GetName())
		if err != nil {
			panic(fmt.Sprintf("RPC must not return an error, but got '%s'", err))
		}
		rpcs[i] = rpc
	}
	return rpcs, nil
}

func (s *spec) RPC(pkgName, svcName, rpcName string) (*grpc.RPC, error) {
	// Check whether pkgName is a valid package or not.
	_, err := s.ServiceNames(pkgName)
	if err != nil {
		return nil, err
	}

	if svcName == "" {
		return nil, ErrServiceIsEmpty
	}

	fqsn := fmt.Sprintf("%s.%s", pkgName, svcName)
	rpcDescs, ok := s.rpcDescs[fqsn]
	if !ok {
		return nil, ErrServiceUnknown
	}

	for _, d := range rpcDescs {
		if d.GetName() == rpcName {
			return &grpc.RPC{
				Name:               d.GetName(),
				FullyQualifiedName: d.GetFullyQualifiedName(),
				RequestType: &grpc.Type{
					Name:               d.GetInputType().GetName(),
					FullyQualifiedName: d.GetInputType().GetFullyQualifiedName(),
					New: func() (interface{}, error) {
						m := dynamic.NewMessage(d.GetInputType())
						return m, nil
					},
				},
				ResponseType: &grpc.Type{
					Name:               d.GetOutputType().GetName(),
					FullyQualifiedName: d.GetOutputType().GetFullyQualifiedName(),
					New: func() (interface{}, error) {
						m := dynamic.NewMessage(d.GetOutputType())
						return m, nil
					},
				},
				IsServerStreaming: d.IsServerStreaming(),
				IsClientStreaming: d.IsClientStreaming(),
			}, nil
		}
	}
	return nil, ErrRPCUnknown
}

func (s *spec) Messages(fqn string) {
	log.Printf("%+v", s.msgDescs[fqn])
}

func (s *spec) Describe(cfg *config.Config) {
	switch cfg.Describe {
	case "pkg":
		log.Printf("%v", s.PackageNames())
	case "svc":
		svcNames, err := s.ServiceNames(cfg.Default.Package)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%v", svcNames)
	case "rpc":
		rpc, err := s.RPC(cfg.Default.Package, cfg.Default.Service, cfg.Default.Method)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%+v", *rpc)
		log.Printf("%+v", *rpc.RequestType)
		log.Printf("%+v", *rpc.ResponseType)
		s.Messages(rpc.RequestType.FullyQualifiedName)
		s.Messages(rpc.ResponseType.FullyQualifiedName)
	}
}
