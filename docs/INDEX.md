# zen-gc Documentation

Welcome to the **zen-gc** documentation.

zen-gc is a free Apache-2.0 **Kubernetes garbage collection controller** that provides declarative cleanup policies for Kubernetes resources. Define TTL-based, selector-based, and condition-based policies for any resource type.

## Where this fits with Zen Mesh

**zen-gc** is an open-source community project from the **[Zen Mesh](https://zen-mesh.io)** team. Zen Mesh is a commercial webhook delivery and data-plane operations platform for private Kubernetes networks.

- **zen-gc** is a **free, independent OSS Kubernetes cleanup controller** — useful on its own, no Zen Mesh required.
- **Zen Mesh** is a **commercial platform** for webhook/edge/data-plane delivery, logs, evidence, and operational truth.
- The projects are **complementary but separate**. zen-gc does not require Zen Mesh, and Zen Mesh does not require zen-gc.

## Getting Started

- [Installation Guide](DEVELOPMENT.md#installation)
- [Quick Start](DEVELOPMENT.md#quick-start)

## User Documentation

- [User Guide](USER_GUIDE.md) — how to create and manage garbage collection policies
- [Operator Guide](OPERATOR_GUIDE.md) — deploying and operating zen-gc in production
- [Architecture](ARCHITECTURE.md) — system design and reconciliation model
- [API Reference](API_REFERENCE.md) — full CRD and policy specification
- [Project Structure](PROJECT_STRUCTURE.md) — codebase layout and ecosystem boundary

## Development

- [Development Setup](DEVELOPMENT.md)
- [Contributing Guidelines](../CONTRIBUTING.md)
- [Release Process](RELEASE.md)

## Resources

- [GitHub Repository](https://github.com/zen-mesh/zen-gc)
- [README](../README.md) — project overview and quick start
- [Zen Mesh](https://zen-mesh.io) — commercial platform from the same team
- [Zen Mesh Documentation](https://docs.zen-mesh.io)
- [Security Policy](../SECURITY.md)
- [Code of Conduct](../CODE_OF_CONDUCT.md)
- [Changelog](../CHANGELOG.md)

