# bventy-backend

## Philosophy

Bventy exists to provide a balanced and structured foundation for service marketplaces. Most platforms prioritize aggressive growth and data harvesting over the quality of interaction between people. We take a different approach.

Our design focuses on:
- Structured interaction: Moving away from chaotic chat-first models toward clear, gated quote workflows.
- Fair exchange: Protecting both vendor and organizer interests by ensuring requirements are clear before engagement begins.
- Deliberate communication: Contact information is only unlocked when both parties have reached a mutual agreement on a quote.
- Transparency: No hidden tracking or dark patterns in how data flows through the system.

## Architecture Overview

The backend is built as a modular Go service using the Gin framework. It is designed to be lean, transparent, and easy to inspect.

- Frontend: Next.js 15 application handles all user interactions.
- Backend: Go (Gin) provides the API and marketplace logic.
- Database: PostgreSQL manages persistent state and relational data.
- R2 Storage: Cloudflare R2 is used for secure, performant attachment and image storage.
- Analytics Layer: Minimal tracking focused on operational metrics to understand system usage.

## Marketplace Lifecycle

The system enforces a specific lifecycle for every transaction:

1. Discovery: Organizers browse verified vendor profiles.
2. Request Quote: Organizers initiate a request with specific event requirements and a mandatory message.
3. Vendor Responds: Vendors provide a priced response or request adjustments.
4. Organizer Accepts: If the terms are met, the organizer accepts the quote.
5. Contact Unlock: Once accepted, contact details are made available to both parties.
6. Expiry + Archive: Completed or inactive quotes are archived to maintain a clean workspace.

## Privacy & Data

We believe in data minimalism. 
- No hidden tracking: Operational analytics are used only to improve system performance.
- Analytics: Limited to platform activity logs and high-level marketplace metrics.
- Privacy: No invasive session recording or third-party behavioral profiling.
- Intentional Gating: Contact data is strictly gated until a mutual agreement is reached.

## License Explanation

This project is licensed under the Apache License 2.0 with the Commons Clause restriction. 

We chose this model to protect the sustainability of the project. While the source code is open for review, modification, and self-hosting, the Commons Clause prevents the software from being sold as a service by third parties without permission. This ensures that the primary development team can continue to support the project independently.

## Contributing

The project is open to contributions that align with our philosophy.
- Issues: We welcome bug reports and architectural discussions via GitHub Issues.
- Improvements: Pull requests are encouraged for performance optimizations and feature refinements.
- Roadmap: Our development plan is transparent and focused on stability.

## Roadmap

This list reflects our current focus. No specific timelines are promised.

- Quote system refinement
- Reviews and feedback system
- Vendor performance scoring
- Payment escrow (future consideration)
- Commission handling logic (future consideration)

---
© 2026 Bventy.