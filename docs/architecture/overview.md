# Architecture Overview

Bventy is built on a service-oriented architecture designed for transparency and reliability. We prioritize clear separation of concerns to ensure the platform remains maintainable and secure.

## System Layers

### Interface Layer (Frontend)
The user interface is a Next.js application that handles session management and presentation logic. It communicates with the backend exclusively through a structured API.

### Logic Layer (Backend)
The backend is a Go service responsible for marketplace state management, authentication, and integration with third-party infrastructure.

### Persistence Layer (Database)
A PostgreSQL database manages relational data, ensuring that quotes, events, and user profiles are stored with strict integrity.

### Storage Layer (R2)
Cloudflare R2 is utilized for object storage, providing a secure and performant way to handle user-uploaded media and documents.

## Design Philosophy

- **Statelessness**: The backend API is stateless, allowing for easier scaling and simplified maintenance.
- **Relational Integrity**: We use the database's relational capabilities to enforce marketplace rules at the storage level.
- **Explicit Communication**: All data exchanges between layers are typed and validated.
