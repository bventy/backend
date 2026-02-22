# Bventy

## What This Is

Bventy is a structured marketplace platform designed to improve how event organizers and service vendors connect. 

In many existing platforms, communication is often chaotic and transactional. Bventy provides a framework where requirements are clarified before personal contact occurs, ensuring that both parties have the information they need to decide if they are a mutual fit.

---

## Design Principles

- **Gated communication**: Contact details are protected until a quote is explicitly accepted.
- **Structured quote workflow**: A predictable sequence of steps from request to fulfillment.
- **Vendor dignity**: Providing vendors with the context they need to price their services fairly.
- **Contact expiry by design**: Communication channels automatically close when their purpose is fulfilled or a deadline passes.
- **Privacy-first architecture**: Data collection is limited to what is necessary for platform operations.
- **Transparent analytics**: Insights are focused on system health and marketplace trends, not user behavior tracking.

---

## Marketplace Lifecycle

The system is built around a deliberate lifecycle to maintain clarity and focus:

*   **Discovery**: Organizers find vendors whose work aligns with their event needs.
*   **Request Quote**: Organizers provide specific event parameters to initiate a request.
*   **Vendor Responds**: Vendors evaluate the request and provide a priced proposal.
*   **Organizer Accepts**: If the terms are suitable, the organizer approves the quote.
*   **Contact Unlock**: Secure contact information is revealed to both parties.
*   **Expiry & Archive**: Completed or inactive requests are archived to keep the workspace clean.

---

## Privacy Approach

We take a deliberate approach to user data.

- **No session replay**: We do not record user screens or mouse movements.
- **No invasive tracking**: We avoid third-party scripts that profile users across the web.
- **Operational analytics only**: We gather data primarily to monitor system stability and platform metrics.
- **Vendor contact never public**: Personal emails and phone numbers are never exposed in profile views.
- **Contact automatically expires**: The system manages the availability of contact data based on the quote's lifecycle.

---

## Architecture Overview

The platform is composed of several focused layers:

- **Frontend**: A Next.js application managing the user experience.
- **Backend**: A Go service providing the marketplace logic and API.
- **Database**: PostgreSQL for relational data and state permanence.
- **R2 Storage**: Secure object storage for attachments and media.
- **Activity Log**: A unified layer for audit trails and platform health.

---

## License

This project is licensed under the **GNU Affero General Public License v3 (AGPL-3.0)**. 

In plain language, this means the software is free to use, modify, and distribute. However, if you modify Bventy and deploy it as a network service, you are required to make your modified source code available to your users under the same license. This ensures that improvements to the community version of the platform remain available to everyone.

---

## Contributing

We welcome contributions that align with our principles of clarity and privacy.

- **Local Setup**: See the [Developer Setup Guide](docs/developers/setup.md) for environment configuration.
- **Issues**: Open a GitHub issue to discuss bugs or architectural improvements.
- **Proposals**: For significant changes, we appreciate a clear description of the problem and the proposed solution.
- **Standards**: We value clean, readable code and comprehensive documentation.

---

## Roadmap

Our current focus includes:

- **Reviews**: A structured feedback system for vendors and organizers.
- **Vendor performance scoring**: Helping organizers make informed decisions based on platform activity.
- **Escrow system**: Investigating secure payment handling to protect both parties.
- **Commission layer**: Logic for platform sustainability.
- **Mobile clients**: Native experiences for better on-the-go management.

This roadmap is subject to change based on community needs and project stability. No specific timelines are promised.

---
© 2026 Bventy.