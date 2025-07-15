# Dislo: Distributed Lock Service

Dislo is a distributed lock management system implemented in Go, providing a gRPC API for safely coordinating access to shared resources across multiple clients and instances. It supports concurrent locking, queueing, and namespace isolation, making it suitable for distributed systems and microservices architectures.

> [!important]
> **Clustered Mode** is a planned feature. However, it is not yet available and currently, Dislo operates in a single-node mode. Clustered mode will be implemented based on demand and development availability.

---

## Features

- **Distributed Locking:** Acquire and release locks on named resources across multiple clients and instances.
- **Namespaces & Instances:** Organize locks by namespaces and instances for multi-tenant or sharded environments.
- **Queue Management:** Clients are queued for locks, ensuring fair access and preventing race conditions.
- **gRPC API:** Well-defined protocol buffer API for interoperability.
- **Cluster Support:** Designed for future cluster mode (currently single-node mode is implemented).
- **Configurable:** Flexible configuration via JSON or YAML files.
- **Logging:** Structured logging with configurable formats (JSON or text).

---

## Security

Dislo is designed for internal use within trusted networks. It does not implement authentication or encryption by default.

For production use, consider deploying Dislo behind a secure network layer (e.g., VPN, TLS) and implement your own authentication and authorization mechanisms.


> [!note]
> Authentication and encryption may be implemented in the future, but currently, it is not planned as Dislo is designed to be simple and lightweight for high-performance distributed locking.

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

---

## Configuration

- The main configuration is located in [`app/internal/configuration/default.json`](app/internal/configuration/default.json).
- You can override settings by providing your own configuration file, though, this is optional.
- Key configuration options include server host/port, instances, namespaces, and cluster settings.

---

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

---

## gRPC API

The API is defined in [`proto/dislo.proto`](proto/dislo.proto). Main RPCs:

- `Lock(Request) returns (Response)`
- `Unlock(Request) returns (Response)`
- `Create(Request) returns (Response)`
- `Delete(Request) returns (Response)`
- `Status(Request) returns (Response)`
- `List(Request) returns (Response)`

See the [generated Go code](app/pkg/generated/dislo/dislo.pb.go) for message and enum details.

---

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

---

## Development

- **Regenerate gRPC code:**  
  Run `./proto/proto.sh` from the `proto` directory after modifying `dislo.proto`.
- **Configuration:**  
  Create your own configuration file in YAML or JSON. You can use the [default configuration](app/internal/configuration/default.json) as a template. If no configuration file is given, the default configuration will be used.
- **Cluster Mode:**  
  Cluster mode is planned but not yet implemented. The server currently runs in single-node mode.

---

## Contributing

Contributions are welcome! Please open issues or submit pull requests for improvements or new features.

---

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
