# Frontend Integration

While the frontend resides in its own repository, its interaction with the backend is a critical part of the system architecture.

## API Consumption

The frontend communicates with the backend via a REST API. All requests are authenticated via a JWT passed in the `Authorization` header.

## Service Layer Pattern

The frontend implements a service-layer pattern that mirrors the backend's resource structure. 
- `quoteService`
- `vendorService`
- `authService`

This mapping ensures that the frontend and backend remain synchronized without leaking presentation logic into the API.

## Real-time Readiness

The architecture is designed to support future real-time updates through predictable state transitions in the backend, which the frontend can poll or eventually receive via streaming.
