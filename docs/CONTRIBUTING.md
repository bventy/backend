# Contributing Guidelines

Thank you for your interest in contributing to Bventy! We welcome community contributions to help make this marketplace backend more robust and feature-rich.

## 🛠 Local Development Setup

1. **Fork & Clone**:
   ```bash
   git clone https://github.com/bventy/backend.git
   cd backend
   ```
2. **Install Go**: Ensure you have Go 1.22+ installed.
3. **Database Setup**:
   - Install PostgreSQL.
   - Run the initialization scripts in `scripts/seed.sql` to setup the base schema if not using migrations.
4. **Environment**:
   - Copy `.env.example` (if available) or create `.env` following the [Environment Guide](ENVIRONMENT.md).
5. **Run**:
   ```bash
   go run cmd/api/main.go
   ```

## 📜 Coding Standards

### Clean Code
- **Explicit Imports**: Use grouped imports.
- **Naming**: Follow Go idiomatic naming (PascalCase for exported, camelCase for internal).
- **Errors**: Always handle errors. Use wrapping where context is needed: `fmt.Errorf("failed to do x: %w", err)`.

### Architecture Adherence
- Do NOT put business logic in handlers. Keep handlers focused on request/response.
- Use `internal/services` for logic that interacts with external systems (R2, SMTP, etc.).
- Ensure all database queries are parameterized.

## 🚀 Branching & PRs
1. Create a feature branch: `git checkout -b feature/your-feature-name`.
2. Commit your changes with descriptive messages: `git commit -m "feat: add analytics for quote conversion"`.
3. Push to your fork and open a Pull Request.
4. Ensure your code passes all lint checks and build steps.

## 🤝 Community & Conduct
- Be respectful and professional.
- Use GitHub Issues for reporting bugs or suggesting features.

---
We look forward to your contributions!
