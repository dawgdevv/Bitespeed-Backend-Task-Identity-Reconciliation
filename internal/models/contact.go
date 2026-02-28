package models

import "time"

// Contact represents a customer contact in the database
type Contact struct {
	ID             int64      `json:"id"`
	PhoneNumber    *string    `json:"phoneNumber,omitempty"`
	Email          *string    `json:"email,omitempty"`
	LinkedID       *int64     `json:"linkedId,omitempty"`
	LinkPrecedence string     `json:"linkPrecedence"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	DeletedAt      *time.Time `json:"deletedAt,omitempty"`
}

// IdentifyRequest represents the incoming request body
type IdentifyRequest struct {
	Email       *string `json:"email"`
	PhoneNumber *string `json:"phoneNumber"`
}

// ContactResponse represents the contact data in the response
type ContactResponse struct {
	PrimaryContactID    int64    `json:"primaryContatctId"`
	Emails              []string `json:"emails"`
	PhoneNumbers        []string `json:"phoneNumbers"`
	SecondaryContactIDs []int64  `json:"secondaryContactIds"`
}

// IdentifyResponse represents the response body
type IdentifyResponse struct {
	Contact ContactResponse `json:"contact"`
}
