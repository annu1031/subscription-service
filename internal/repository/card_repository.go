package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"subscription-management/internal/model"
)

type CardRepository interface {
	Create(ctx context.Context, card *model.Card) error
	GetByID(ctx context.Context, id string) (*model.Card, error)
	GetByUserID(ctx context.Context, userID string) ([]model.Card, error)
	Update(ctx context.Context, card *model.Card) error
	Delete(ctx context.Context, id string) error
	SetDefault(ctx context.Context, userID, cardID string) error
	DeleteByUserID(ctx context.Context, userID string) error
}

type SQLCardRepository struct {
	db *sqlx.DB
}

func NewCardRepository(db *sqlx.DB) CardRepository {
	return &SQLCardRepository{
		db: db,
	}
}

func (r *SQLCardRepository) Create(ctx context.Context, card *model.Card) error {
	if card.ID == "" {
		card.ID = uuid.New().String()
	}
	
	cardLength := len(card.CardNumber)
	if cardLength >= 4 {
		card.LastFourDigits = card.CardNumber[cardLength-4:]
	}
	
	card.CreatedAt = time.Now()
	card.UpdatedAt = time.Now()
	
	query := `
		INSERT INTO cards (
			id, user_id, card_number, card_holder_name, 
			expiry_month, expiry_year, card_type, 
			last_four_digits, is_default, created_at, updated_at
		) VALUES (
			:id, :user_id, :card_number, :card_holder_name, 
			:expiry_month, :expiry_year, :card_type, 
			:last_four_digits, :is_default, :created_at, :updated_at
		)
	`
	
	_, err := r.db.NamedExecContext(ctx, query, card)
	return err
}

func (r *SQLCardRepository) GetByID(ctx context.Context, id string) (*model.Card, error) {
	var card model.Card
	
	query := `
		SELECT * FROM cards WHERE id = ?
	`
	
	err := r.db.GetContext(ctx, &card, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil 
		}
		return nil, err
	}
	
	return &card, nil
}

func (r *SQLCardRepository) GetByUserID(ctx context.Context, userID string) ([]model.Card, error) {
	var cards []model.Card
	
	query := `
		SELECT * FROM cards WHERE user_id = ?
	`
	
	err := r.db.SelectContext(ctx, &cards, query, userID)
	if err != nil {
		return nil, err
	}
	
	return cards, nil
}

func (r *SQLCardRepository) Update(ctx context.Context, card *model.Card) error {
	card.UpdatedAt = time.Now()
	
	query := `
		UPDATE cards SET
			card_holder_name = :card_holder_name,
			expiry_month = :expiry_month,
			expiry_year = :expiry_year,
			is_default = :is_default,
			updated_at = :updated_at
		WHERE id = :id
	`
	
	_, err := r.db.NamedExecContext(ctx, query, card)
	return err
}

func (r *SQLCardRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM cards WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *SQLCardRepository) SetDefault(ctx context.Context, userID, cardID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	unsetQuery := `UPDATE cards SET is_default = false WHERE user_id = ?`
	_, err = tx.ExecContext(ctx, unsetQuery, userID)
	if err != nil {
		tx.Rollback()
		return err
	}
	
	setQuery := `UPDATE cards SET is_default = true WHERE id = ? AND user_id = ?`
	result, err := tx.ExecContext(ctx, setQuery, cardID, userID)
	if err != nil {
		tx.Rollback()
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return err
	}
	
	if rowsAffected == 0 {
		tx.Rollback()
		return errors.New("card not found or doesn't belong to user")
	}
	
	return tx.Commit()
}


func (r *SQLCardRepository) DeleteByUserID(ctx context.Context, userID string) error {
    query := `DELETE FROM cards WHERE user_id = ?`
    _, err := r.db.ExecContext(ctx, query, userID)
    return err
}