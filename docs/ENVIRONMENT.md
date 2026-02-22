# Environment Variables Guide

The Bventy API is configured via environment variables. For local development, these should be placed in a `.env` file in the root directory.

## 🔑 Required Variables

### Database (PostgreSQL)
- **`DATABASE_URL`**: The full connection string for your PostgreSQL instance.
  - *Example*: `postgres://user:password@hostname:5432/dbname?sslmode=require`
  - *Note*: We recommend using Neon for serverless scaling.

### Authentication
- **`JWT_SECRET`**: A strong, unique string used to sign JSON Web Tokens.
  - *Requirement*: Minimum 32 characters in production.

### Cloudflare R2 (Object Storage)
- **`CLOUDFLARE_R2_ACCESS_KEY_ID`**: Your R2 API token access key.
- **`CLOUDFLARE_R2_SECRET_ACCESS_KEY`**: Your R2 API token secret key.
- **`CLOUDFLARE_R2_ACCOUNT_ID`**: Your Cloudflare account ID.
- **`CLOUDFLARE_R2_BUCKET_NAME`**: The name of the bucket used for media.
- **`CLOUDFLARE_R2_PUBLIC_URL`**: The public URL (Custom Domain or worker URL) for accessing uploaded files.

### Server Configuration
- **`PORT`**: The port the API will listen on.
  - *Default*: `8082`
- **`GIN_MODE`**: Set to `release` in production for optimized logging.

### Tracking (PostHog)
- **`POSTHOG_API_KEY`**: Your PostHog project API key.
- **`POSTHOG_HOST`**: Usually `https://us.i.posthog.com` or `https://eu.i.posthog.com`.

## 🛡 Security Best Practices
1. **Never commit `.env`**: Ensure it is ignored by Git.
2. **Rotate Secrets**: If a secret is leaked, rotate all credentials immediately.
3. **Production Secrets**: Use your platform's (Render, Railway, etc.) native secret management for production deployments.
