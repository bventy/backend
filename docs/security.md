# Security

Bventy is built with a security-first approach, focusing on protecting user identity and proprietary marketplace data.

## Access Control
- **JWT Auth**: All state-changing operations require a valid JWT.
- **Role-Based Gating**: Specific routes (Admin/Superadmin) require explicit roles verified by backend middleware.
- **Ownership Validation**: The system ensures users can only modify their own events or reply to received quotes.

## Media Security (R2)
- **Attachment Rules**: File uploads are validated on the frontend (5MB limit) and gated on the backend.
- **Signed URLs**: Media assets are accessed via signed or private Cloudflare R2 URLs to prevent direct, unauthorized scraping of user assets.
- **Sanitization**: Image metadata is stripped during the compression phase on the client side to protect privacy.

## Data Gating
- **Contact Gating**: Personal contact information (Email/Phone) is physically omitted from the API response until a quote is explicitly accepted.
- **Expiry Logic**: Quotes have built-in deadlines. Once a deadline passes, the system prevents further interaction to preserve the state of the marketplace as of that moment.

## Database Protection
- **Parameterized Queries**: We use `pgx` with strictly sanitized input to prevent SQL injection attacks.
- **Neon Branching**: Development and staging environments use isolated database branches to ensure production data is never exposed during development.
