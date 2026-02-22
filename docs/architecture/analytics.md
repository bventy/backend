# Analytics Layer

Analytics in Bventy are designed to provide operational clarity without compromising user privacy.

## Operational Metrics

We focus on data that helps monitor platform health:
- **Quote Volume**: Tracking the number of requests and fulfillment rates.
- **User Growth**: Monitoring new organizer and vendor acquisitions.
- **System Latency**: Logging request performance to ensure a smooth experience.

## Tracking Mechanism

The system uses a unified tracking endpoint that records activity with minimal metadata.
- **Action**: A broad category of activity (e.g., `quote_request`, `vendor_onboarding`).
- **Entity**: The specific resource involved.
- **Actor**: The user performing the action.

## Privacy Guardrails

- **No Personal Data**: Tracking logs are kept separate from user profile details.
- **Limited Scope**: We do not track user behavior outside of platform-specific marketplace actions.
- **Internal Use**: Data is utilized only to improve platform functionality and provide transparency to administrators.
