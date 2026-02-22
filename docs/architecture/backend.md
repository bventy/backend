# Backend Architecture

The Bventy backend is a focused Go service using the Gin framework. It is designed to be lean and predictable.

## Core Responsibilities

- **Authentication**: JWT-based session management.
- **Role Validation**: Enforcing permissions for organizers, vendors, and administrators.
- **Marketplace Engine**: Managing the lifecycle of quote requests and responses.
- **Media Integration**: Handling secure uploads and generating access patterns for Cloudflare R2.
- **Activity Tracking**: Recording operational events for system audit and health.

## Service Patterns

We follow a pattern of explicit handlers and services:

- **Handlers**: Manage HTTP request binding, validation, and response formatting.
- **Services**: Contain the core business logic and coordinate between the database and external infrastructure.
- **Middlewares**: Address cross-cutting concerns like logging, authentication, and role checking.

## Security Controls

- **Input Sanitization**: All incoming data is validated against strict types.
- **Explicit SQL**: We write SQL directly to ensure full visibility into database interactions.
- **Gated Responses**: Sensitive data is filtered at the handler level based on the requester's context.
