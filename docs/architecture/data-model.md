# Data Model

Bventy uses a relational database to ensure that the complex dependencies of a marketplace (users, vendors, events, and quotes) remain consistent.

## Core Tables

### Users and Roles
Manages identity and access levels. A user can be an organizer or a vendor (or both).

### Events
Created by organizers to provide context for quote requests. Events contain metadata like date, guest count, and requirements.

### Quotes
The central entity of the marketplace. Quotes track the interaction between an organizer and a vendor, including status, pricing, and communication history.

### Media and Attachments
Tracks the location and ownership of files stored in R2. This includes vendor gallery images and proposal documents.

### Activity Logs
Records system-wide events to provide an audit trail for administrative review and platform metrics.

## Integrity Rules

- **Foreign Keys**: Enforced for all relational dependencies.
- **Unique Constraints**: Used for slugs and sensitive identifiers to prevent data duplication.
- **Indexing**: Optimized for common search and dashboard queries.
