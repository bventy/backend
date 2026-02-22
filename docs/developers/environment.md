# Environment Reference

The following variables are required to run the Bventy backend. We recommend using a `.env` file for local development.

## Core API

- `PORT`: The port the service will listen on (default: `8080`).
- `DB_URL`: The full connection string for your PostgreSQL database.
- `JWT_SECRET`: A secure string used to sign session tokens.

## Media (Cloudflare R2)

- `R2_BUCKET_NAME`: The name of your R2 bucket.
- `R2_ACCOUNT_ID`: Your Cloudflare account ID.
- `R2_ACCESS_KEY_ID`: The access key for your R2 credentials.
- `R2_SECRET_ACCESS_KEY`: The secret key for your R2 credentials.
- `R2_PUBLIC_URL`: The public base URL for your R2 bucket (if using a custom domain).

## Security and Rules

- `GIN_MODE`: Set to `release` in production to disable verbose logging.
- `ALLOWED_ORIGINS`: A comma-separated list of origins for CORS policy.

## Analytics

- `POSTHOG_API_KEY`: Key for system activity tracking.
- `POSTHOG_HOST`: Your PostHog instance host.
