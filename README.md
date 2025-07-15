

<h1 align="center">external-haproxy-operator</h1>
<p align="center">
  <i>Kubernetes operator for managing external HAProxy backends as custom resources.</i>
  <br/><br/>
  <a target="_blank" href="https://github.com/ullbergm/external-haproxy-operator/releases"><img src="https://img.shields.io/github/v/release/ullbergm/external-haproxy-operator?logo=hackthebox&color=609966&logoColor=fff" alt="Current Version"/></a>
  <a target="_blank" href="https://github.com/ullbergm/external-haproxy-operator"><img src="https://img.shields.io/github/last-commit/ullbergm/external-haproxy-operator?logo=github&color=609966&logoColor=fff" alt="Last commit"/></a>
  <a href="https://github.com/ullbergm/external-haproxy-operator/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-Apache%202.0-609966?logo=opensourceinitiative&logoColor=fff" alt="License Apache 2.0"/></a>
  <a href="https://codecov.io/gh/ullbergm/external-haproxy-operator"><img src="https://codecov.io/gh/ullbergm/external-haproxy-operator/graph/badge.svg?token=6HTWM4O7WK" alt="Test Coverage"/></a>
  <br />
  <a href="https://buymeacoffee.com/magnus.ullberg"><img src="https://img.shields.io/badge/Buy%20me%20a-coffee-ff1414.svg?color=aa1414&logoColor=fff&label=Buy%20me%20a" alt="buy me a coffee"/></a>
  <a href="https://ullberg.us/cv.pdf"><img src="https://img.shields.io/badge/Offer%20me%20a-job-00d414.svg?color=0000f4&logoColor=fff&label=Offer%20me%20a" alt="offer me a job"/></a>
</p>

<details open="open">
<summary>Table of Contents</summary>

- [🎯 Features](#-features)
- [🚀 Getting Started](#-getting-started)
- [⚙️ Usage](#️-usage)
  - [📝 Configuration](#-configuration)
  - [🗂️ Distribution](#-distribution)
  - [🗑️ Uninstall](#-uninstall)
- [🗂️ Contributing](#-contributing)
- [🏆 Authors & contributors](#-authors--contributors)
- [🗂️ Security](#-security)
- [📄 License](#-license)
- [🌎 Acknowledgements](#-acknowledgements)

</details>

## 🎯 Features

- Declarative management of HAProxy backends via CRDs
- Automated backend registration and updates
- Metrics and monitoring integration ([metrics docs](docs/monitoring/metrics.md))
- RBAC and security best practices
- Helm and YAML bundle installation options

---

## 🚀 Getting Started

### Prerequisites

- Go v1.24.0+
- Docker v17.03+
- kubectl v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster

### Build and Push Operator Image

```sh
make docker-build docker-push IMG=<your-registry>/external-haproxy-operator:tag
```

### Install CRDs

```sh
make install
```

### Deploy Operator

```sh
make deploy IMG=<your-registry>/external-haproxy-operator:tag
```

### Apply Example Backend CRs

```sh
kubectl apply -k config/samples/
```

---

## ⚙️ Usage

- See `config/samples/` for example CRs
- See [docs/monitoring/metrics.md](docs/monitoring/metrics.md) for metrics
- For API details, see `api/v1alpha1/`

---

## 📝 Configuration

Configuration is managed via Kubernetes CRDs. For advanced configuration, see the [API documentation](api/v1alpha1/) and [metrics documentation](docs/monitoring/metrics.md).

---

## 🗂️ Distribution

### YAML Bundle

Build installer:
```sh
make build-installer IMG=<your-registry>/external-haproxy-operator:tag
```
Apply bundle:
```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/external-haproxy-operator/<tag or branch>/dist/install.yaml
```

### Helm Chart

Build chart:
```sh
operator-sdk edit --plugins=helm/v1-alpha
```
Chart is generated under `dist/chart/`.

> _If you change the project, update the Helm chart and re-apply custom values as needed._

---

## 🗑️ Uninstall

```sh
kubectl delete -k config/samples/
make uninstall
make undeploy
```

---

## 🗂️ Contributing

Contributions are welcome! Please open issues or pull requests. See [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html) for operator development best practices.

Run `make help` for available targets.

---

## 🏆 Authors & contributors

The original setup of this repository is by [Magnus Ullberg](https://github.com/ullbergm).

For a full list of all authors and contributors, see [the contributors page](https://github.com/ullbergm/external-haproxy-operator/contributors).

---

## 🗂️ Security

external-haproxy-operator follows good practices of security, but 100% security cannot be assured.
external-haproxy-operator is provided **"as is"** without any **warranty**. Use at your own risk.

_For more information and to report security issues, please refer to our [security documentation](docs/SECURITY.md)._

---

## 📄 License

This project is licensed under the **Apache License 2.0**.

See [LICENSE](LICENSE) for more information.

---

## 🌎 Acknowledgements

- [HAProxy](https://www.haproxy.org/)
- [Kubebuilder](https://book.kubebuilder.io/)
- [Operator SDK](https://sdk.operatorframework.io/)
