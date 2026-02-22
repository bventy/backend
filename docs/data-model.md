# Data Model

The system utilizes a relational PostgreSQL database (Neon) to maintain data integrity and support complex marketplace queries.

## Primary Tables

### users
The core identity table.
- **id**: Unique identifier (UUID/String).
- **email**: Used for authentication and notifications.
- **role**: Defines permissions (admin, super_admin, user/organizer).
- **is_vendor**: Boolean flag for vendor profile existence.

### events
Managed by organizers.
- **id**: Unique identifier.
- **owner_id**: Foreign key to users.
- **title / type**: Descriptive event information.
- **event_date**: Scheduled timestamp.

### quote_requests
The engine of the marketplace.
- **id**: Unique identifier.
- **event_id**: Foreign key to events.
- **organizer_id / vendor_id**: Identifies the two parties.
- **status**: Tracks current lifecycle state.
- **message / vendor_response**: Communication logs.
- **quoted_price**: Finalized pricing.
- **attachment_url**: Link to rate card in R2.

### platform_activity_log
Internal audit and analytics.
- **id**: Unique identifier.
- **user_id**: Responsible actor.
- **action**: Human-readable description of the event.
- **entity_type / entity_id**: Context for the action.

### vendor_gallery_images
- **id**: Unique identifier.
- **vendor_id**: Owner of the image.
- **url**: R2 link to the image.
- **position**: Order of display in the profile.

We prefer explicit foreign keys and indexing on IDs to maintain performance as the marketplace grows.
