# Bitespeed Identity Reconciliation Service

A Go web service that implements identity reconciliation across customer contacts using SQLite.

## Overview

This service manages customer identity across multiple purchases by linking contacts that share either email or phone number. It implements a primary/secondary contact hierarchy where the oldest contact becomes the primary.

## Tech Stack

- **Language**: Go
- **Database**: SQLite
- **Web Framework**: Gorilla Mux
- **Port**: 8080 (configurable via PORT environment variable)

## API Endpoint

### POST /identify

Consolidates customer contacts based on email and/or phone number.

#### Request Body
```json
{
  "email": "string",
  "phoneNumber": "string"
}
```

At least one of `email` or `phoneNumber` must be provided.

#### Response Body
```json
{
  "contact": {
    "primaryContatctId": number,
    "emails": ["string"],
    "phoneNumbers": ["string"],
    "secondaryContactIds": [number]
  }
}
```

## Identity Reconciliation Logic

1. **New Customer**: If no existing contacts match, creates a new primary contact
2. **Linking Contacts**: Contacts are linked if they share email or phone number
3. **Secondary Contact**: When new information is provided for an existing contact, creates a secondary contact linked to the primary
4. **Primary Transition**: If a new request links contacts, the oldest becomes primary and others become secondary

## Getting Started

### Prerequisites

- Go 1.21 or higher
- GCC (for SQLite driver)

### Installation

```bash
# Clone the repository
git clone <your-repo-url>
cd bitespeed

# Install dependencies
go mod tidy

# Build the application
go build -o bitespeed .

# Run the server
./bitespeed
```

The server will start on port 8080 by default.

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| PORT | Server port | 8080 |
| DATABASE_URL | SQLite database file path | ./bitespeed.db |

## Example Usage

### Create a new primary contact
```bash
curl -X POST http://localhost:8080/identify \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","phoneNumber":"1234567890"}'
```

Response:
```json
{
  "contact": {
    "primaryContatctId": 1,
    "emails": ["user@example.com"],
    "phoneNumbers": ["1234567890"],
    "secondaryContactIds": []
  }
}
```

### Link with new phone number
```bash
curl -X POST http://localhost:8080/identify \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","phoneNumber":"0987654321"}'
```

Response:
```json
{
  "contact": {
    "primaryContatctId": 1,
    "emails": ["user@example.com"],
    "phoneNumbers": ["1234567890", "0987654321"],
    "secondaryContactIds": [2]
  }
}
```

## Database Schema

```sql
CREATE TABLE contacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    phone_number TEXT,
    email TEXT,
    linked_id INTEGER,
    link_precedence TEXT CHECK(link_precedence IN ('primary', 'secondary')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    FOREIGN KEY (linked_id) REFERENCES contacts(id)
);
```

## Project Structure

```
bitespeed/
├── main.go                           # Entry point
├── go.mod, go.sum                    # Go dependencies
├── internal/
│   ├── database/db.go               # Database connection
│   ├── models/contact.go            # Data models
│   ├── handlers/identify.go         # HTTP handler
│   └── service/reconciliation.go    # Business logic
└── migrations/
    └── 001_create_contacts_table.sql # Schema
```

## License

MIT
