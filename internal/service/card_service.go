package service

import (
	"context"
	"errors"
	"regexp"
	"time"

	"subscription-management/internal/model"
	"subscription-management/internal/repository"
)

var (
	ErrInvalidCardNumber     = errors.New("invalid card number")
	ErrInvalidExpiryDate     = errors.New("card has expired or has an invalid expiry date")
	ErrInvalidCardHolderName = errors.New("card holder name is required")
	ErrCardNotFound          = errors.New("card not found")
	ErrUnauthorized          = errors.New("unauthorized access to card")
)

type CardService interface {
	CreateCard(ctx context.Context, card *model.Card) error
	GetCard(ctx context.Context, id, userID string) (*model.Card, error)
	GetUserCards(ctx context.Context, userID string) ([]model.Card, error)
	UpdateCard(ctx context.Context, card *model.Card) error
	DeleteCard(ctx context.Context, id, userID string) error
	SetDefaultCard(ctx context.Context, userID, cardID string) error
	DeleteAllUserCards(ctx context.Context, userID string) error
}

type DefaultCardService struct {
	cardRepo repository.CardRepository
}

func NewCardService(cardRepo repository.CardRepository) CardService {
	return &DefaultCardService{
		cardRepo: cardRepo,
	}
}

func (s *DefaultCardService) CreateCard(ctx context.Context, card *model.Card) error {
    
    cardLength := len(card.CardNumber)
    lastFour := ""
    if cardLength >= 4 {
        lastFour = card.CardNumber[cardLength-4:]
    }
    
    
    existingCards, err := s.cardRepo.GetByUserID(ctx, card.UserID)
    if err != nil {
        return err
    }
    
    for _, existingCard := range existingCards {
        if existingCard.LastFourDigits == lastFour {
            return errors.New("card already exists for this user")
        }
    }
    
    
    if err := validateCard(card); err != nil {
        return err
    }
    
    card.CardType = determineCardType(card.CardNumber)
    
    return s.cardRepo.Create(ctx, card)
}

func (s *DefaultCardService) GetCard(ctx context.Context, id, userID string) (*model.Card, error) {
	card, err := s.cardRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	
	if card == nil {
		return nil, ErrCardNotFound
	}
	
	if card.UserID != userID {
		return nil, ErrUnauthorized
	}
	
	return card, nil
}

func (s *DefaultCardService) GetUserCards(ctx context.Context, userID string) ([]model.Card, error) {
	return s.cardRepo.GetByUserID(ctx, userID)
}

func (s *DefaultCardService) UpdateCard(ctx context.Context, card *model.Card) error {
	
	existingCard, err := s.GetCard(ctx, card.ID, card.UserID)
	if err != nil {
		return err
	}
	
	card.CardNumber = existingCard.CardNumber
	card.LastFourDigits = existingCard.LastFourDigits
	card.CardType = existingCard.CardType
	
	if card.CardHolderName == "" {
		return ErrInvalidCardHolderName
	}
	
	if !isValidExpiryDate(card.ExpiryMonth, card.ExpiryYear) {
		return ErrInvalidExpiryDate
	}
	
	return s.cardRepo.Update(ctx, card)
}

func (s *DefaultCardService) DeleteCard(ctx context.Context, id, userID string) error {
	
	_, err := s.GetCard(ctx, id, userID)
	if err != nil {
		return err
	}
	
	return s.cardRepo.Delete(ctx, id)
}


func (s *DefaultCardService) SetDefaultCard(ctx context.Context, userID, cardID string) error {
	
	_, err := s.GetCard(ctx, cardID, userID)
	if err != nil {
		return err
	}
	
	return s.cardRepo.SetDefault(ctx, userID, cardID)
}


func validateCard(card *model.Card) error {
	if !isValidCardNumber(card.CardNumber) {
		return ErrInvalidCardNumber
	}

	if !isValidExpiryDate(card.ExpiryMonth, card.ExpiryYear) {
		return ErrInvalidExpiryDate
	}

	if card.CardHolderName == "" {
		return ErrInvalidCardHolderName
	}
	
	return nil
}

func isValidCardNumber(number string) bool {
	re := regexp.MustCompile(`[\s-]`)
	number = re.ReplaceAllString(number, "")
	
	matched, _ := regexp.MatchString(`^\d+$`, number)
	if !matched {
		return false
	}
	
	if len(number) < 13 || len(number) > 19 {
		return false
	}

	sum := 0
	alt := false
	
	for i := len(number) - 1; i >= 0; i-- {
		n := int(number[i] - '0')
		
		if alt {
			n *= 2
			if n > 9 {
				n = n%10 + 1
			}
		}
		
		sum += n
		alt = !alt
	}
	
	return sum%10 == 0
}

func isValidExpiryDate(month, year int) bool {
	if month < 1 || month > 12 {
		return false
	}
	
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	if year < currentYear || (year == currentYear && month < currentMonth) {
		return false
	}

	if year > currentYear+10 {
		return false
	}
	
	return true
}

func determineCardType(number string) string {
	re := regexp.MustCompile(`[\s-]`)
	number = re.ReplaceAllString(number, "")

	if matched, _ := regexp.MatchString(`^4\d{12}(\d{3})?$`, number); matched {
		return "VISA"
	}

	if matched, _ := regexp.MatchString(`^5[1-5]\d{14}$`, number); matched {
		return "MASTERCARD"
	}

	if matched, _ := regexp.MatchString(`^3[47]\d{13}$`, number); matched {
		return "AMEX"
	}

	if matched, _ := regexp.MatchString(`^6(?:011|5\d{2})\d{12}$`, number); matched {
		return "DISCOVER"
	}

	return "UNKNOWN"
}


func (s *DefaultCardService) DeleteAllUserCards(ctx context.Context, userID string) error {
    return s.cardRepo.DeleteByUserID(ctx, userID)
}