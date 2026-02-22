# Contact Expiry Logic

Protecting user privacy and maintaining a clean marketplace requires that contact information is only available when necessary.

## Expiry Design

When an organizer accepts a vendor's quote, the system "unlocks" access to contact details (email, phone number). However, this access is not permanent.

### Automatic Closure
Contact permissions automatically expire based on the project's timeline or a period of inactivity. This prevents a permanent leak of private information.

### Lifecycle Resolution
Once a quote is marked as completed or archived, the system revokes the active "unlock" state. The records remain for audit purposes, but the live communication channel is closed.

## Privacy Benefits

- **Decreased Surface Area**: Private data is only exposed during the active phase of a project.
- **Dignity by Design**: Users do not need to manually request data deletion; the system handles access revocation automatically.
- **Spam Prevention**: Expiry ensures that contact details cannot be harvested for long-term marketing lists.
