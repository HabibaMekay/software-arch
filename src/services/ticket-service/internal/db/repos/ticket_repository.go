package repos

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"ticket-service/internal/db/models"

	"github.com/jmoiron/sqlx"
)

// TicketRepository handles database operations for tickets.
type TicketRepository struct {
	db *sqlx.DB
}

// NewTicketRepository creates a new TicketRepository.
func NewTicketRepository(db *sqlx.DB) *TicketRepository {
	return &TicketRepository{db: db}
}

// EventWithTickets represents an event that has tickets
type EventWithTickets struct {
	EventID     int `json:"event_id" db:"event_id"`
	TicketCount int `json:"ticket_count" db:"ticket_count"`
}

// GetTicketByID retrieves a ticket by its ID.
func (r *TicketRepository) GetTicketByID(ticketID int) (*models.Ticket, error) {
	var ticket models.Ticket
	err := r.db.Get(&ticket, "SELECT * FROM ticket WHERE ticket_id=$1", ticketID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &ticket, nil
}

// GetTicketsByEventID retrieves tickets for a given event_id.
func (r *TicketRepository) GetTicketsByEventID(eventID int) ([]models.Ticket, error) {
	var tickets []models.Ticket
	err := r.db.Select(&tickets, "SELECT * FROM ticket WHERE event_id=$1", eventID)
	if err != nil {
		return nil, err
	}
	return tickets, nil
}

// CreateTicket creates a new ticket.
func (r *TicketRepository) CreateTicket(ticket *models.Ticket) (*models.Ticket, error) {
	var createdTicket models.Ticket
	err := r.db.QueryRowx(
		"INSERT INTO ticket (event_id, user_id, ticket_code, status) VALUES ($1, $2, $3, $4) RETURNING *",
		ticket.EventID, ticket.UserID, ticket.TicketCode, ticket.Status,
	).StructScan(&createdTicket)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			return nil, fmt.Errorf("ticket_code already exists")
		}
		return nil, err
	}
	return &createdTicket, nil
}

// UpdateTicketStatus updates the status of a ticket.
func (r *TicketRepository) UpdateTicketStatus(ticketID int, status string) (*models.Ticket, error) {
	var updatedTicket models.Ticket
	err := r.db.QueryRowx(
		"UPDATE ticket SET status=$1 WHERE ticket_id=$2 RETURNING *",
		status, ticketID,
	).StructScan(&updatedTicket)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &updatedTicket, nil
}

func (r *TicketRepository) GetTicketByCode(ticketCode string) (*models.Ticket, error) {
	ticket := &models.Ticket{}
	query := `SELECT ticket_id, event_id, user_id, ticket_code, status FROM ticket WHERE ticket_code = CAST($1 AS UUID)`
	err := r.db.QueryRow(query, ticketCode).Scan(
		&ticket.TicketID, &ticket.EventID, &ticket.UserID, &ticket.TicketCode, &ticket.Status,
	)
	if err == sql.ErrNoRows {
		log.Printf("No ticket found for ticket_code: %s", ticketCode)
		return nil, fmt.Errorf("ticket not found")
	}
	if err != nil {
		log.Printf("Database error for ticket_code %s: %v", ticketCode, err)
		return nil, err
	}
	return ticket, nil
}

// GetEventsWithTickets retrieves all events that have at least one ticket
func (r *TicketRepository) GetEventsWithTickets() ([]EventWithTickets, error) {
	var events []EventWithTickets
	query := `
		SELECT event_id, COUNT(*) as ticket_count 
		FROM ticket 
		GROUP BY event_id 
		HAVING COUNT(*) > 0
		ORDER BY event_id`

	err := r.db.Select(&events, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get events with tickets: %v", err)
	}
	return events, nil
}
