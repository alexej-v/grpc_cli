package grpc


type RPC struct {
	Name               string
	FullyQualifiedName string
	RequestType        *Type
	ResponseType       *Type
	IsServerStreaming  bool
	IsClientStreaming  bool
}

// Type is a type for representing requests/responses.
type Type struct {
	// Name is the name of Type.
	Name string

	// FullyQualifiedName is the name that contains the package name this Type belongs.
	FullyQualifiedName string

	// New instantiates a new instance of Type.  It is used for decode requests and responses.
	New func() (interface{}, error)
}
