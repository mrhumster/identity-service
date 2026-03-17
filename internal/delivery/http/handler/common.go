package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mrhumster/identity-service/internal/delivery/http/dto/response"
	"github.com/mrhumster/identity-service/internal/service"
)

func getErrorMessage(fieldError validator.FieldError) string {
	switch fieldError.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "min":
		return "Value is too short"
	case "max":
		return "Value is too long"
	case "gte":
		return "Value is too small"
	case "lte":
		return "Value is too large"
	default:
		return fieldError.Error()
	}
}

func GetUserIDFromContext(c *gin.Context) (*string, error) {
	userID, exists := c.Get("userID")
	if !exists {
		return nil, errors.New("user ID not found in context")
	}
	fmt.Printf("⚠️ GetUserIDFromContext: %v %t", userID, userID)
	id, ok := userID.(string)
	if !ok {
		return nil, errors.New("invalid user ID type in context")
	}

	return &id, nil
}

func GetUserEmailFromContext(c *gin.Context) (string, error) {
	email, exists := c.Get("userEmail")
	if !exists {
		return "", errors.New("user email not found in context")
	}

	emailStr, ok := email.(string)
	if !ok {
		return "", errors.New("invalid user email type in context")
	}

	return emailStr, nil
}

type CommonHandler struct {
	tokenService *service.TokenService
}

func NewCommonHandler(tokenService *service.TokenService) *CommonHandler {
	return &CommonHandler{
		tokenService: tokenService,
	}
}

func (h *CommonHandler) GetPublicKey(c *gin.Context) {
	publicKeyPEM, err := h.tokenService.GetPublicKeyPEM()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.ErrorResponse("failed to get public key"))
		return
	}
	c.Data(http.StatusOK, "application/x-pem-file", []byte(publicKeyPEM))
}
