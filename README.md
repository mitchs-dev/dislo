# Dislo: Distributed Lock Service

Dislo is a distributed lock management system implemented in Go, providing a gRPC API for safely coordinating access to shared resources across multiple clients and instances. It supports concurrent locking, queueing, and namespace isolation, making it suitable for distributed systems and microservices architectures.

## Features

- **Distributed Locking:** Acquire and release locks on named resources across multiple clients and instances.
- **Namespaces & Instances:** Organize locks by namespaces and instances for multi-tenant or sharded environments.
- **Queue Management:** Clients are queued for locks, ensuring fair access and preventing race conditions.
- **gRPC API:** Well-defined protocol buffer API for interoperability.
- **Configurable:** Flexible configuration via JSON or YAML files.
- **Logging:** Structured logging with configurable formats (JSON or text).

## Planned Features

- **Cluster Mode:** Cluster mode will provide dislo with the ability to scale based on demands to ensure that locks are available no matter the size of the workloads.
- **Persistence:** Enable your lock statuses to persist through restarts
- **Authentication**: Lightweight client authentication while still maintaining maximum performance.
- **Additional SDK Language Support**: Enabling Python, NodeJS, and other langauges to use dislo.

## Security

See [SECURITY.md](SECURITY.md) for more details on security practices and reporting vulnerabilities.

### Authentication and Encryption

Dislo is designed for internal use within trusted networks. It does not implement authentication or encryption by default.

For production use, consider deploying Dislo behind a secure network layer (e.g., VPN, TLS) and implement your own authentication and authorization mechanisms.

> [!note]
> Client authentation is a planned feature. However, performance remains a priority for the dislo project.

## Support

See [SUPPORT.md](SUPPORT.md) for support options and how to get help.

### Client SDK Languages

The following languages are supports for the client SDK: 

- Go

> [!note]
> While the languages above are the only supported SDK languages, you can still use the `dislo.proto` file to generate your own files to intereact with dislo. However, the dislo contributors will **not provide support for 3rd-party scripts, applications, etc**. Contributors _will_ provide support for the `dislo.proto` file, however.

## Getting Started

### Prerequisites

- Go 1.16 or later
- [protoc](https://grpc.io/docs/protoc-installation/) (for regenerating gRPC code)
- Docker (optional, for running with `docker-compose`)

### Installation

1. **Clone the repository:**
   ```sh
   git clone https://github.com/yourusername/dislo.git
   cd dislo
   ```

2. **Build the project:**
   ```sh
   cd app
   go build -o dislo ./cmd
   ```

3. **(Optional) Generate gRPC code:**
   ```sh
   cd proto
   ./proto.sh
   ```

## Configuration

- The main configuration is located in [`internal/configuration/default.json`](internal/configuration/default.json).
- You can override settings by providing your own configuration file, though, this is optional.
- Key configuration options include server host/port, instances, namespaces, and cluster settings.

## Running the Server

### Using Go

```sh
cd app
go run ./cmd/main.go
```

### Using Docker Compose

```sh
docker-compose up --build
```

The server will start and listen on the configured host and port.


## gRPC API

The API is defined in [`proto/dislo.proto`](proto/dislo.proto). Main RPCs:

- `Lock(Request) returns (Response)`
- `Unlock(Request) returns (Response)`
- `Create(Request) returns (Response)`
- `Delete(Request) returns (Response)`
- `Status(Request) returns (Response)`
- `List(Request) returns (Response)`

See the [generated Go code](pkg/generated/dislo/dislo.pb.go) for message and enum details.


## Example Client

An example Go client is provided in [`example/client/main.go`](example/client/main.go). It demonstrates:

- Connecting to the Dislo server
- Acquiring and releasing locks
- Handling lock contention and queueing

To run the example client:

```sh
cd example/client
go run main.go
```

## Development

- **Regenerate gRPC code:**  
  Run `./proto/proto.sh` from the `proto` directory after modifying `dislo.proto`.
- **Configuration:**  
  Create your own configuration file in YAML or JSON. You can use the [default configuration](internal/configuration/default.json) as a template. If no configuration file is given, the default configuration will be used.
- **Cluster Mode:**  
  Cluster mode is planned but not yet implemented. The server currently runs in single-node mode.

## Contributing

Contributions are welcome! Please open issues or submit pull requests for improvements or new features.

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on contributing to Dislo.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
