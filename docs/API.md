# Bventy API Reference

Welcome to the Bventy API. This guide is for developers and partners looking to understand how the Bventy engine works. Our API follows the same principles of transparency and deliberate communication that define our marketplace.

---

## The Bventy Philosophy

Unlike traditional "gig" platforms, Bventy is built on **deliberate communication**. Our API doesn't just pass data; it enforces a workflow that protects everyone's time, privacy, and dignity.

1.  **Privacy by Design**: Personal contact details (WhatsApp, Email, Phone) are encrypted and gated. They are only revealed once an organizer and vendor have mutually agreed on a quote.
2.  **Structured Lifecycle**: Every interaction follows a predictable sequence. This eliminates the "transactional chaos" common in event planning.
3.  **Contact Expiry**: To keep your workspace clean and secure, access to contact details automatically expires after the event is completed or the quote is archived.

---

## Getting Started

### Base URL
All API requests should be directed to:
`https://api.bventy.in` (Production)

### Authentication
Bventy uses a dual-layered authentication system for security and cross-subdomain compatibility.

-   **Session Cookies**: Our primary method for web clients. Secure, HttpOnly, and SameSite cookies allow you to stay logged in across `app.bventy.in`, `vendor.bventy.in`, and `auth.bventy.in`.
-   **JWT Tokens**: For stateless requests, we provide a standard JSON Web Token (JWT) in the `Authorization` header.

```bash
Authorization: Bearer <your_session_token>
```

---

## Identity & Profiles

Every journey on Bventy starts with an identity. We keep profile management simple and focused.

### Get My Profile
`GET /me`
Get your identity, roles, and group memberships.
-   **Response**: Returns your full name, username, email verification status, and active group memberships.

### Update Profile
`PUT /me`
Update your personal details.
-   **Fields**: `full_name`, `username`, `phone`, `city`, `bio`, `profile_image_url`.
-   **Note**: Usernames must be unique across the platform.

### Profile Image
`POST /users/profile-image`
Upload a profile picture. Images are compressed and optimized automatically.
-   **Payload**: `multipart/form-data` with a `file` field.

---

## Vendor Ecosystem

Bventy connects event organizers with verified vendors. Each profile showcases a vendor's expertise and portfolio.

### Explore Vendors
`GET /vendors`
List all verified vendors on the platform.
-   **Response**: A collection of vendor cards including business names, categories, average ratings, and primary portfolio images.

### Onboarding
`POST /vendor/onboard` (Verified Email Required)
Start your journey as a Bventy vendor.
-   **Payload**: `business_name`, `category`, `city`, `whatsapp_link`, `bio`.
-   **Status**: New profiles enter a `pending` state for moderation to maintain platform quality.

### Gallery & Portfolio
Share your work using high-quality images and documents.
-   **Add Image**: `POST /vendors/:id/gallery` (Max 25 images)
-   **Add Portfolio**: `POST /vendors/:id/portfolio` (PDF only, max 20 files)

---

## Event Coordination

Events are the foundation for every quote request.

### Create an Event
`POST /events`
Define the scope of your upcoming event.
-   **Payload**: `title`, `city`, `event_date` (ISO), `event_type`, `budget_min`, `budget_max`.
-   **Groups**: Events can be owned by a group if `organizer_group_id` is provided.

### Shortlisting
`POST /events/:id/shortlist/:vendorID`
Keep track of the vendors you love for a specific event.

---

## The Quote Lifecycle

The quote lifecycle is a structured workflow that moves from discovery to fulfillment.

### 1. Request a Quote
`POST /quotes/request`
Organizers initiate contact by providing requirements to a specific vendor.
-   **Payload**: `event_id`, `vendor_id`, `message`, `budget_range`, `deadline`.
-   **Privacy**: At this stage, NO personal contact info is shared.

### 2. Vendor Response
`PATCH /quotes/respond/:id`
Vendors review the requirements and provide a formal proposal.
-   **Payload**: `quoted_price`, `vendor_response` (Message), `attachment_url`.

### 3. The Decision
Organizers have three clear choices:
-   **Accept**: `PATCH /quotes/accept/:id`. This unlocks contact details for both parties.
-   **Revision**: `PATCH /quotes/revision/:id`. Request a change or clarification.
-   **Reject**: `PATCH /quotes/reject/:id`. Decline the proposal.

### 4. Unlocking Contact
`GET /quotes/:id/contact` (Accepted Status Only)
Once a quote is accepted, contact details are shared.
-   **Response**: Returns verified phone, email, and WhatsApp links.
-   **Expiry**: Access is temporary and typically expires 15 days after the event.

---

## Platform Administration

Our admin tools ensure the marketplace remains healthy, respectful, and high-performing.

### Performance Metrics
Access aggregated, privacy-preserving insights.
-   **Overview**: `GET /admin/metrics/overview`
-   **Growth**: `GET /admin/metrics/growth`
-   **Marketplace**: `GET /admin/metrics/marketplace`

### Moderation
-   **Approve Vendor**: `PATCH /admin/vendors/:id/approve`
-   **User Management**: `DELETE /admin/users/:id`

---

## Handling Success & Failure

We use standard HTTP status codes to communicate clearly.

| Code | Meaning | Human Translation |
| :--- | :--- | :--- |
| **200** | OK | Everything worked as expected. |
| **201** | Created | A new resource (like an event or user) was successfully born. |
| **400** | Bad Request | The request was missing something important. |
| **401** | Unauthorized | We don't know who you are. Please log in first. |
| **403** | Forbidden | You don't have permission for this specific action. |
| **404** | Not Found | The resource you're looking for has moved or doesn't exist. |
| **409** | Conflict | Something already exists (like a duplicate email). |
| **500** | Server Error | An internal error occurred. |

---

## Trust & Security

-   **Data Retention**: We automatically cleanup sensitive logs every 30 days.
-   **DDoS Protection**: We use rate limiting to prevent abuse and ensure platform stability.
-   **Encryption**: All data is served over TLS (HTTPS).

---
© 2026 Bventy.
