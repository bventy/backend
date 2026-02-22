# Bventy API

The robust, industry-grade Go backend powering the [Bventy](https://bventy.in) marketplace. Built for high performance, security, and scalability.

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![Gin Framework](https://img.shields.io/badge/Gin-1.9+-00ADD8?style=for-the-badge&logo=go)](https://gin-gonic.com/)
[![License](https://img.shields.io/badge/License-Proprietary-red?style=for-the-badge)](LICENSE)

## 🚀 Overview

Bventy API is a highly modular backend system designed to handle complex marketplace interactions between event organizers and service vendors. It manages everything from identity and access control to real-time quote request lifecycles and advanced analytics.

### Key Features
- **Hybrid Marketplace Logic**: Specialized handlers for quote requests, vendor responses, and organizer actions.
- **Granular RBAC**: Role-based access control protecting critical admin and vendor endpoints.
- **Media Engine**: Seamless integration with Cloudflare R2 for high-performance file and image handling.
- **Analytics Layer**: Unified tracking system for platform activity and marketplace growth.
- **Secure by Design**: JWT-based authentication with robust middleware validation.

## 🛠 Tech Stack
- **Language**: Go (1.22+)
- **Framework**: Gin Gonic
- **Database**: PostgreSQL (Hosted on Neon)
- **Object Storage**: Cloudflare R2 (S3 Compatible)
- **Tracking**: PostHog
- **Authentication**: JWT (JSON Web Tokens)

## 📁 Repository Structure
```text
.
├── cmd/                # Entry points
│   └── api/            # Main server package
├── internal/           # Private application code
│   ├── auth/           # Identity & JWT logic
│   ├── db/             # Data access & migrations
│   ├── handlers/       # Request routing & controllers
│   ├── middleware/     # Auth & Role validation
│   ├── models/         # Database & API schemas
│   ├── routes/         # Routing definitions
│   └── services/       # Business logic (e.g., Media)
├── docs/               # In-depth technical documentation
└── scripts/            # Database seeding & utility scripts
```

## 🚥 Getting Started

### Prerequisites
- Go 1.22 or higher
- PostgreSQL instance (Neon recommended)
- Cloudflare R2 bucket credentials

### Quick Start
1. **Clone the repository**:
   ```bash
   git clone https://github.com/bventy/backend.git
   cd backend
   ```
2. **Setup environment variables**:
   Create a `.env` file in the root directory. See [docs/ENVIRONMENT.md](docs/ENVIRONMENT.md) for details.
3. **Install dependencies**:
   ```bash
   go mod download
   ```
4. **Run the server**:
   ```bash
   go run cmd/api/main.go
   ```

## 📖 Documentation
- [Architecture & Layering](docs/ARCHITECTURE.md) - Deep dive into how the system is built.
- [API Reference](docs/API.md) - Endpoint definitions and requirements.
- [Environment Variables](docs/ENVIRONMENT.md) - Detailed guide on configuration.
- [Contributing](docs/CONTRIBUTING.md) - Guidelines for developers.

## 🔒 Security
We take security seriously. Please do not expose sensitive credentials in PRs. See [docs/ENVIRONMENT.md](docs/ENVIRONMENT.md) for secure configuration practices.

---
© 2026 Bventy. All rights reserved.