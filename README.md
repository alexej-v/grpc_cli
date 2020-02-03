# grpc_cli
Simple gRPC client

### Installation
``` sh
go get -u github.com/alexej-v/grpc_cli
```

### Run
``` sh
grpc_cli --port 7002 --file protofile_name.proto --path ~/Path --path ~/Path2
```

### Usage example:
Create connection:
``` sh
grpc_cli --host <host-with-no-http-in-url> --port <port> --path <proto-file-folder> --file <protofile-filename>
```
Set package and service, use <tab> for hints:
``` sh
package <package-name> 
service <service-name>
```

Add headers, if needed:
``` sh
set header Authorization Bearer <token>
```

and, at last, call:
``` sh
call GetFullOrder {"order_id": "<order_id>"}
```

#### Example:
``` sh
grpc_cli --host example.host.org --port 82 --path ./ --file serviceName.proto
package host.exampe.api.service
service ServiceName
set header Authorization Bearer JUG9435rho.47hTGfghdsuRYBUhgNFEgfgjhdWMUJGdghdfgghggFfdghW
call GetOrder {"order_id": "<order_id>"}
```
