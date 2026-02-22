# Architecture

This document provides a technical overview of how Bventy is structured. We prioritize simplicity and clarity over complex abstractions.

## Implementation Principles
- Separation of concerns: Handlers manage HTTP flow; Services manage complex logic.
- Transparency: SQL queries are written explicitly for readability and performance optimization.
- Scalability: The stateless API and serverless database allow the system to handle varying loads without manual intervention.

## Core Components

### Backend
The backend is a Go application using the Gin framework. It is responsible for:
- Identity and Access Management (JWT).
- Marketplace logic and state transitions.
- Secure media handling (S3/R2 integration).
- Unified activity tracking.

### Frontend
The frontend is a Next.js 15 application utilizing the App Router. It focuses on:
- High-performance UI rendering.
- State management for user sessions.
- Responsive design for diverse devices.
- Direct interaction with the Backend API.

### Infrastructure
- Database: PostgreSQL (Neon) for managed, serverless relational data.
- Storage: Cloudflare R2 for secure, cost-effective object storage.
- Analytics: PostHog and Umami for internal metrics and system health monitoring.

## Data Flow

Request → Middleware (Auth/Role) → Handler (Validation/Binding) → Logic/Database → Response

We avoid heavy ORM usage to maintain clear visibility into how the database is being utilized.
