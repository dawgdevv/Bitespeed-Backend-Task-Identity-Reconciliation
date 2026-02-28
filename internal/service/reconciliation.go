package service

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"bitespeed/internal/database"
	"bitespeed/internal/models"
)

// ReconciliationService handles identity reconciliation logic
type ReconciliationService struct {
	db *database.DB
}

// NewReconciliationService creates a new reconciliation service
func NewReconciliationService(db *database.DB) *ReconciliationService {
	return &ReconciliationService{db: db}
}

// Identify handles the identity reconciliation logic
func (s *ReconciliationService) Identify(req models.IdentifyRequest) (*models.IdentifyResponse, error) {
	// Find existing contacts matching email OR phone number
	linkedContacts, err := s.findLinkedContacts(req.Email, req.PhoneNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to find linked contacts: %w", err)
	}

	var primaryContact *models.Contact

	if len(linkedContacts) == 0 {
		// No existing contacts - create new primary
		primaryContact, err = s.createPrimaryContact(req.Email, req.PhoneNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to create primary contact: %w", err)
		}
	} else {
		// Find the oldest contact to be the primary
		primaryContact = s.findOldestContact(linkedContacts)

		// Check if we need to create a secondary contact
		hasNewInfo := s.hasNewInformation(linkedContacts, req.Email, req.PhoneNumber)

		if hasNewInfo {
			_, err = s.createSecondaryContact(req.Email, req.PhoneNumber, primaryContact.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to create secondary contact: %w", err)
			}
		}

		// Reconcile primary/secondary status
		err = s.reconcilePrimaryStatus(linkedContacts, primaryContact.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to reconcile primary status: %w", err)
		}
	}

	// Build the response
	return s.buildResponse(primaryContact.ID)
}

// findLinkedContacts finds all contacts linked by email or phone number
func (s *ReconciliationService) findLinkedContacts(email, phoneNumber *string) ([]*models.Contact, error) {
	contactMap := make(map[int64]*models.Contact)

	// Query by email
	if email != nil && *email != "" {
		contacts, err := s.queryContactsByEmail(*email)
		if err != nil {
			return nil, err
		}
		for _, c := range contacts {
			contactMap[c.ID] = c
		}
	}

	// Query by phone number
	if phoneNumber != nil && *phoneNumber != "" {
		contacts, err := s.queryContactsByPhoneNumber(*phoneNumber)
		if err != nil {
			return nil, err
		}
		for _, c := range contacts {
			contactMap[c.ID] = c
		}
	}

	// Also find contacts linked via linked_id
	allLinkedIDs := make(map[int64]bool)
	for _, c := range contactMap {
		if c.LinkedID != nil {
			allLinkedIDs[*c.LinkedID] = true
		}
	}

	for linkedID := range allLinkedIDs {
		linkedContacts, err := s.queryContactsByLinkedID(linkedID)
		if err != nil {
			return nil, err
		}
		for _, c := range linkedContacts {
			contactMap[c.ID] = c
		}
	}

	// Convert map to slice
	result := make([]*models.Contact, 0, len(contactMap))
	for _, c := range contactMap {
		result = append(result, c)
	}

	return result, nil
}

// queryContactsByEmail queries contacts by email
func (s *ReconciliationService) queryContactsByEmail(email string) ([]*models.Contact, error) {
	query := `SELECT id, phone_number, email, linked_id, link_precedence, created_at, updated_at, deleted_at 
			  FROM contacts WHERE email = $1 AND deleted_at IS NULL`
	return s.queryContacts(query, email)
}

// queryContactsByPhoneNumber queries contacts by phone number
func (s *ReconciliationService) queryContactsByPhoneNumber(phone string) ([]*models.Contact, error) {
	query := `SELECT id, phone_number, email, linked_id, link_precedence, created_at, updated_at, deleted_at 
			  FROM contacts WHERE phone_number = $1 AND deleted_at IS NULL`
	return s.queryContacts(query, phone)
}

// queryContactsByLinkedID queries contacts by linked_id
func (s *ReconciliationService) queryContactsByLinkedID(linkedID int64) ([]*models.Contact, error) {
	query := `SELECT id, phone_number, email, linked_id, link_precedence, created_at, updated_at, deleted_at 
			  FROM contacts WHERE linked_id = $1 AND deleted_at IS NULL`
	return s.queryContacts(query, linkedID)
}

// queryContacts executes a query and returns contacts
func (s *ReconciliationService) queryContacts(query string, args ...interface{}) ([]*models.Contact, error) {
	rows, err := s.db.Conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []*models.Contact
	for rows.Next() {
		c := &models.Contact{}
		var phone, email sql.NullString
		var linkedID sql.NullInt64
		var deletedAt sql.NullTime

		err := rows.Scan(&c.ID, &phone, &email, &linkedID, &c.LinkPrecedence, &c.CreatedAt, &c.UpdatedAt, &deletedAt)
		if err != nil {
			return nil, err
		}

		if phone.Valid {
			c.PhoneNumber = &phone.String
		}
		if email.Valid {
			c.Email = &email.String
		}
		if linkedID.Valid {
			c.LinkedID = &linkedID.Int64
		}
		if deletedAt.Valid {
			c.DeletedAt = &deletedAt.Time
		}

		contacts = append(contacts, c)
	}

	return contacts, rows.Err()
}

// findOldestContact finds the oldest contact in the list
func (s *ReconciliationService) findOldestContact(contacts []*models.Contact) *models.Contact {
	if len(contacts) == 0 {
		return nil
	}

	sort.Slice(contacts, func(i, j int) bool {
		return contacts[i].CreatedAt.Before(contacts[j].CreatedAt)
	})

	return contacts[0]
}

// hasNewInformation checks if the request contains new email or phone number
func (s *ReconciliationService) hasNewInformation(contacts []*models.Contact, email, phoneNumber *string) bool {
	existingEmails := make(map[string]bool)
	existingPhones := make(map[string]bool)

	for _, c := range contacts {
		if c.Email != nil {
			existingEmails[*c.Email] = true
		}
		if c.PhoneNumber != nil {
			existingPhones[*c.PhoneNumber] = true
		}
	}

	// Check if email is new
	if email != nil && *email != "" && !existingEmails[*email] {
		return true
	}

	// Check if phone number is new
	if phoneNumber != nil && *phoneNumber != "" && !existingPhones[*phoneNumber] {
		return true
	}

	return false
}

// createPrimaryContact creates a new primary contact
func (s *ReconciliationService) createPrimaryContact(email, phoneNumber *string) (*models.Contact, error) {
	query := `INSERT INTO contacts (phone_number, email, link_precedence, created_at, updated_at) 
			  VALUES ($1, $2, 'primary', $3, $4) RETURNING id`

	now := time.Now()
	var id int64
	err := s.db.Conn.QueryRow(query, phoneNumber, email, now, now).Scan(&id)
	if err != nil {
		return nil, err
	}

	return &models.Contact{
		ID:             id,
		PhoneNumber:    phoneNumber,
		Email:          email,
		LinkPrecedence: "primary",
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// createSecondaryContact creates a new secondary contact
func (s *ReconciliationService) createSecondaryContact(email, phoneNumber *string, linkedID int64) (*models.Contact, error) {
	query := `INSERT INTO contacts (phone_number, email, linked_id, link_precedence, created_at, updated_at) 
			  VALUES ($1, $2, $3, 'secondary', $4, $5) RETURNING id`

	now := time.Now()
	var id int64
	err := s.db.Conn.QueryRow(query, phoneNumber, email, linkedID, now, now).Scan(&id)
	if err != nil {
		return nil, err
	}

	return &models.Contact{
		ID:             id,
		PhoneNumber:    phoneNumber,
		Email:          email,
		LinkedID:       &linkedID,
		LinkPrecedence: "secondary",
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// reconcilePrimaryStatus ensures the oldest contact is primary and others are secondary
func (s *ReconciliationService) reconcilePrimaryStatus(contacts []*models.Contact, primaryID int64) error {
	for _, c := range contacts {
		if c.ID == primaryID {
			// This should be primary
			if c.LinkPrecedence != "primary" {
				err := s.updateContactPrecedence(c.ID, "primary", nil)
				if err != nil {
					return err
				}
			}
		} else {
			// This should be secondary
			if c.LinkPrecedence != "secondary" || c.LinkedID == nil || *c.LinkedID != primaryID {
				err := s.updateContactPrecedence(c.ID, "secondary", &primaryID)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// updateContactPrecedence updates a contact's link_precedence and linked_id
func (s *ReconciliationService) updateContactPrecedence(id int64, precedence string, linkedID *int64) error {
	query := `UPDATE contacts SET link_precedence = $1, linked_id = $2, updated_at = $3 WHERE id = $4`
	_, err := s.db.Conn.Exec(query, precedence, linkedID, time.Now(), id)
	return err
}

// buildResponse builds the identify response for a primary contact
func (s *ReconciliationService) buildResponse(primaryID int64) (*models.IdentifyResponse, error) {
	// Get all linked contacts (primary + secondaries)
	allContacts, err := s.getAllLinkedContacts(primaryID)
	if err != nil {
		return nil, err
	}

	emails := []string{}
	phoneNumbers := []string{}
	secondaryContactIDs := []int64{}
	primaryEmail := ""
	primaryPhone := ""

	// Find primary contact details first
	for _, c := range allContacts {
		if c.ID == primaryID {
			if c.Email != nil {
				primaryEmail = *c.Email
			}
			if c.PhoneNumber != nil {
				primaryPhone = *c.PhoneNumber
			}
		} else {
			secondaryContactIDs = append(secondaryContactIDs, c.ID)
		}
	}

	// Add primary email and phone first
	if primaryEmail != "" {
		emails = append(emails, primaryEmail)
	}
	if primaryPhone != "" {
		phoneNumbers = append(phoneNumbers, primaryPhone)
	}

	// Collect unique emails and phone numbers from all contacts
	emailSet := make(map[string]bool)
	phoneSet := make(map[string]bool)

	for _, c := range allContacts {
		if c.Email != nil && *c.Email != "" && *c.Email != primaryEmail {
			emailSet[*c.Email] = true
		}
		if c.PhoneNumber != nil && *c.PhoneNumber != "" && *c.PhoneNumber != primaryPhone {
			phoneSet[*c.PhoneNumber] = true
		}
	}

	// Add secondary emails and phones
	for email := range emailSet {
		emails = append(emails, email)
	}
	for phone := range phoneSet {
		phoneNumbers = append(phoneNumbers, phone)
	}

	return &models.IdentifyResponse{
		Contact: models.ContactResponse{
			PrimaryContactID:    primaryID,
			Emails:              emails,
			PhoneNumbers:        phoneNumbers,
			SecondaryContactIDs: secondaryContactIDs,
		},
	}, nil
}

// getAllLinkedContacts gets the primary contact and all secondary contacts
func (s *ReconciliationService) getAllLinkedContacts(primaryID int64) ([]*models.Contact, error) {
	query := `SELECT id, phone_number, email, linked_id, link_precedence, created_at, updated_at, deleted_at 
			  FROM contacts 
			  WHERE (id = $1 OR linked_id = $2) AND deleted_at IS NULL`

	return s.queryContacts(query, primaryID, primaryID)
}
