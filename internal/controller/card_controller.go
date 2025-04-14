package controller

import (
	"net/http"
	

	"github.com/labstack/echo/v4"

	"subscription-management/internal/model"
	"subscription-management/internal/service"
)


type CardController struct {
	cardService service.CardService
}


func NewCardController(cardService service.CardService) *CardController {
	return &CardController{
		cardService: cardService,
	}
}


func (cc *CardController) RegisterRoutes(e *echo.Echo) {
	cards := e.Group("/api/cards")
	
	
	cards.POST("", cc.CreateCard)
	cards.GET("", cc.GetUserCards)
	cards.GET("/:id", cc.GetCard)
	cards.PUT("/:id", cc.UpdateCard)
	cards.DELETE("/:id", cc.DeleteCard)
	cards.PUT("/:id/default", cc.SetDefaultCard)
	cards.DELETE("/all", cc.DeleteAllUserCards)
}


type CreateCardRequest struct {
	UserID         string `json:"userId" validate:"required"`
	CardNumber     string `json:"cardNumber" validate:"required"`
	CardHolderName string `json:"cardHolderName" validate:"required"`
	ExpiryMonth    int    `json:"expiryMonth" validate:"required,min=1,max=12"`
	ExpiryYear     int    `json:"expiryYear" validate:"required"`
}


func (cc *CardController) CreateCard(c echo.Context) error {
	var req CreateCardRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	
	card := &model.Card{
		UserID:         req.UserID,
		CardNumber:     req.CardNumber,
		CardHolderName: req.CardHolderName,
		ExpiryMonth:    req.ExpiryMonth,
		ExpiryYear:     req.ExpiryYear,
	}
	
	if err := cc.cardService.CreateCard(c.Request().Context(), card); err != nil {
        switch err {
        case service.ErrInvalidCardNumber:
            return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid card number"})
        case service.ErrInvalidExpiryDate:
            return c.JSON(http.StatusBadRequest, map[string]string{"error": "Card has expired or has an invalid expiry date"})
        case service.ErrInvalidCardHolderName:
            return c.JSON(http.StatusBadRequest, map[string]string{"error": "Card holder name is required"})
        default:
            if err.Error() == "card already exists for this user" {
                return c.JSON(http.StatusConflict, map[string]string{"error": "This card is already registered for this user"})
            }
            return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create card"})
        }
    }
	
	
	card.CardNumber = "XXXX-XXXX-XXXX-" + card.LastFourDigits
	
	return c.JSON(http.StatusCreated, card)
}


func (cc *CardController) GetUserCards(c echo.Context) error {
	userID := c.QueryParam("userId")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID is required"})
	}
	
	cards, err := cc.cardService.GetUserCards(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve cards"})
	}
	
	
	for i := range cards {
		cards[i].CardNumber = "XXXX-XXXX-XXXX-" + cards[i].LastFourDigits
	}
	
	return c.JSON(http.StatusOK, cards)
}


func (cc *CardController) GetCard(c echo.Context) error {
	id := c.Param("id")
	userID := c.QueryParam("userId")
	
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID is required"})
	}
	
	card, err := cc.cardService.GetCard(c.Request().Context(), id, userID)
	if err != nil {
		switch err {
		case service.ErrCardNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Card not found"})
		case service.ErrUnauthorized:
			return c.JSON(http.StatusForbidden, map[string]string{"error": "Unauthorized access to this card"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve card"})
		}
	}
	
	
	card.CardNumber = "XXXX-XXXX-XXXX-" + card.LastFourDigits
	
	return c.JSON(http.StatusOK, card)
}


type UpdateCardRequest struct {
	UserID         string `json:"userId" validate:"required"`
	CardHolderName string `json:"cardHolderName" validate:"required"`
	ExpiryMonth    int    `json:"expiryMonth" validate:"required,min=1,max=12"`
	ExpiryYear     int    `json:"expiryYear" validate:"required"`
}


func (cc *CardController) UpdateCard(c echo.Context) error {
	id := c.Param("id")
	
	var req UpdateCardRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	
	card := &model.Card{
		ID:             id,
		UserID:         req.UserID,
		CardHolderName: req.CardHolderName,
		ExpiryMonth:    req.ExpiryMonth,
		ExpiryYear:     req.ExpiryYear,
	}
	
	if err := cc.cardService.UpdateCard(c.Request().Context(), card); err != nil {
		switch err {
		case service.ErrCardNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Card not found"})
		case service.ErrUnauthorized:
			return c.JSON(http.StatusForbidden, map[string]string{"error": "Unauthorized access to this card"})
		case service.ErrInvalidExpiryDate:
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Card has expired or has an invalid expiry date"})
		case service.ErrInvalidCardHolderName:
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Card holder name is required"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update card"})
		}
	}
	
	return c.JSON(http.StatusOK, map[string]string{"message": "Card updated successfully"})
}


func (cc *CardController) DeleteCard(c echo.Context) error {
	id := c.Param("id")
	userID := c.QueryParam("userId")
	
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID is required"})
	}
	
	if err := cc.cardService.DeleteCard(c.Request().Context(), id, userID); err != nil {
		switch err {
		case service.ErrCardNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Card not found"})
		case service.ErrUnauthorized:
			return c.JSON(http.StatusForbidden, map[string]string{"error": "Unauthorized access to this card"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete card"})
		}
	}
	
	return c.JSON(http.StatusOK, map[string]string{"message": "Card deleted successfully"})
}


func (cc *CardController) SetDefaultCard(c echo.Context) error {
	id := c.Param("id")
	userID := c.QueryParam("userId")
	
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID is required"})
	}
	
	if err := cc.cardService.SetDefaultCard(c.Request().Context(), userID, id); err != nil {
		switch err {
		case service.ErrCardNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Card not found"})
		case service.ErrUnauthorized:
			return c.JSON(http.StatusForbidden, map[string]string{"error": "Unauthorized access to this card"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to set card as default"})
		}
	}
	
	return c.JSON(http.StatusOK, map[string]string{"message": "Card set as default successfully"})
}



func (cc *CardController) DeleteAllUserCards(c echo.Context) error {
    userID := c.QueryParam("userId")
    
    if userID == "" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID is required"})
    }
    
    if err := cc.cardService.DeleteAllUserCards(c.Request().Context(), userID); err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete cards"})
    }
    
    return c.JSON(http.StatusOK, map[string]string{"message": "All cards deleted successfully"})
}