# Developer Setup

Thank you for contributing to Bventy. This guide will help you set up a local development environment for the backend service.

## Prerequisites

- **Go**: Version 1.22 or later.
- **PostgreSQL**: A local or remote instance of PostgreSQL.
- **Cloudflare R2**: A bucket and set of credentials for media testing.

## Installation

1.  **Clone the Repository**:
    ```bash
    git clone https://github.com/bventy/backend.git
    cd backend
    ```

2.  **Environment Configuration**:
    Copy the example environment file and fill in your credentials:
    ```bash
    cp .env.example .env
    ```
    See the [Environment Reference](environment.md) for details on each key.

3.  **Database Migration**:
    Ensure your database is reachable and run the initial schema:
    ```bash
    # (Migration steps specific to your setup)
    ```

4.  **Run the Service**:
    ```bash
    go run cmd/server/main.go
    ```
    The API should now be available at `http://localhost:8080`.

## Testing

We prioritize reliable integrations. Run the test suite to ensure your environment is configured correctly:
```bash
go test ./...
```
