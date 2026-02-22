# API Reference

The Bventy API is a RESTful system using JSON for request and response payloads. All protected endpoints require a valid JWT in the `Authorization` header.

## 🔑 Authentication
- **Base URL**: `https://api.bventy.in` (Production)
- **Header**: `Authorization: Bearer <your_jwt_token>`

---

## 🌍 Public Endpoints

### Health Check
- **GET** `/health`
- returns `200 OK` if the server is healthy.

### Vendors
- **GET** `/vendors`: List all verified vendors.
- **GET** `/vendors/slug/:slug`: Get detailed profile for a specific vendor by their slug.

---

## 🔐 Protected Endpoints (Requires Auth)

### User & Session
- **POST** `/auth/signup`: Create a new account.
- **POST** `/auth/login`: Authenticate and receive a JWT.
- **GET** `/me`: Retrieve the current user's profile and roles.
- **PUT** `/me`: Update user profile details.

### Quote Requests (Marketplace Core)
- **POST** `/quotes/request`: Initiate a new quote request from an organizer.
- **GET** `/quotes/organizer`: List quotes requested by the current organizer.
- **GET** `/quotes/vendor`: List quotes received by the current vendor.
- **PATCH** `/quotes/respond/:id`: Vendor sends a price and message for a request.
- **PATCH** `/quotes/accept/:id`: Organizer accepts a vendor's quote.
- **PATCH** `/quotes/reject/:id`: Organizer rejects a quote.
- **PATCH** `/quotes/revision/:id`: Organizer requests a revision with feedback.
- **GET** `/quotes/:id/contact`: Unlocked contact details for accepted quotes.

### Events & Groups
- **POST** `/events`: Create a new event.
- **GET** `/events`: List your events.
- **POST** `/events/:id/shortlist/:vendorID`: Save a vendor to an event shortlist.
- **POST** `/groups`: Create a community group.
- **GET** `/groups/my`: List groups you belong to.

### Media & Assets
- **POST** `/media/upload`: Generic file upload to Cloudflare R2.
- **POST** `/vendors/:id/gallery`: Add an image to the vendor's public gallery.
- **POST** `/vendors/:id/portfolio`: Upload a PDF/document to the vendor's portfolio.

---

## 🛡 Admin Endpoints (Admin Only)

### Marketplace Analytics
- **GET** `/admin/metrics/overview`: General platform health.
- **GET** `/admin/metrics/growth`: User and vendor acquisition trends.
- **GET** `/admin/metrics/marketplace`: Quote conversion and GMV metrics.

### Moderation
- **GET** `/admin/vendors`: List all vendors (pending and verified).
- **PATCH** `/admin/vendors/:id/approve`: Verify a vendor profile.
- **PATCH** `/admin/vendors/:id/reject`: Reject a vendor application.

---

## 📉 Error Handling
The API uses standard HTTP status codes:
- `200/201`: Success.
- `400`: Bad Request (Invalid payload).
- `401`: Unauthorized (Missing or invalid token).
- `403`: Forbidden (Insufficient permissions).
- `404`: Not Found.
- `500`: Internal Server Error.

All errors return a JSON response:
```json
{
  "error": "Detailed error message here"
}
```
