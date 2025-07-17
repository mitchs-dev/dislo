# Security Policy

## Supported Versions

Dislo is developed and tested with Go 1.16 and above. Only the latest release is actively supported.

## Reporting a Vulnerability

If you discover a security vulnerability in Dislo, please report it privately to the maintainer:

- Mitchell Stanton <email@mitchs.dev>

Do **not** open public issues for security vulnerabilities.

## Security Considerations

- Dislo does **not** implement authentication or encryption by default.
- It is intended for use within trusted networks.
- For production deployments, run Dislo behind a secure network layer (e.g., VPN, TLS proxy) and implement your own authentication and authorization as needed.
- See [README.md](README.md#security) for more details.

## Disclosure Policy

We will investigate all reported vulnerabilities and respond as quickly as possible. Once a fix is available, users will be notified via a new release and changelog entry.

---

_Dislo is an open-source project licensed under the MIT License. See [LICENSE](LICENSE) for details._