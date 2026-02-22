# Quote State Diagram

The lifecycle of a quote is managed by a finite state machine. This ensures that every interaction is valid based on the current context.

```mermaid
stateDiagram-v2
    [*] --> Pending: Request Created
    Pending --> Responded: Vendor Proposals
    Responded --> RevisionRequested: Organizer Asks for Changes
    RevisionRequested --> Responded: Vendor Updates Proposal
    Responded --> Accepted: Organizer Accepts
    Accepted --> Completed: Event Finalized
    Accepted --> Expired: Time Limit Reached
    Pending --> Cancelled: Organizer Cancels
    Responded --> Cancelled: Organizer Cancels
    Completed --> [*]
    Cancelled --> [*]
    Expired --> [*]
```

## State Definitions

- **Pending**: The initial request is awaiting a vendor response.
- **Responded**: A proposal has been submitted and is under review.
- **RevisionRequested**: Feedback has been provided; a modified response is expected.
- **Accepted**: Terms are agreed upon; contact details are unlocked.
- **Completed**: The engagement has reached its natural conclusion.
- **Cancelled**: The request has been terminated by the organizer.
- **Expired**: Access has been revoked due to inactivity or time limits.
