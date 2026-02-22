# Data Handling

Bventy manages data with a focus on operational integrity and user protection.

## Data Classification

### Public Data
Includes vendor profile descriptions, portfolio images, and general marketplace listings. This data is intended for discovery and is accessible to all registered users.

### Protected Data
Includes event details and quote proposals. Access is restricted to the specific organizer and vendor involved in the interaction.

### Sensitive Data
Includes contact information (email, phone). This data is "gated" and only revealed upon explicit quote acceptance. 

## Storage and Retention

- **Database**: PostgreSQL handles all relational data. Backups are performed regularly to ensure persistence.
- **Media**: All files are stored in Cloudflare R2. We use signed URLs to ensure that only authorized users can access specific attachments.
- **Expiry**: We implement automatic revocation of access to sensitive data once its business purpose (the quote lifecycle) is concluded.

## Deletion
Users can request the deletion of their accounts. In such cases, we remove personal profile data while retaining anonymized marketplace transaction records for system audit purposes.
