package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/mrhumster/identity-service/internal/delivery/http/dto/request"
	"github.com/mrhumster/identity-service/internal/delivery/http/dto/response"
	"github.com/mrhumster/identity-service/internal/domain/models"
	"github.com/mrhumster/identity-service/internal/service"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var user request.UserRequest
	if err := c.ShouldBindJSON(&user); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			errors := make(map[string]string)
			for _, fieldError := range validationErrors {
				errors[fieldError.Field()] = getErrorMessage(fieldError)
			}
			c.JSON(http.StatusBadRequest, gin.H{"errors": errors})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if user.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password can't be empty"})
		return
	}
	var u models.User
	u.FillInTheRequest(user)
	id, err := h.service.CreateUser(c, u)
	if err != nil {
		switch err {
		case service.ErrUserAlreadyExists:
			c.JSON(http.StatusConflict, gin.H{
				"error": "user already exists",
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *UserHandler) ReadUser(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.ReadUser(c, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		c.Abort()
		return
	}
	var u response.UserResponse
	u.FillInTheModel(user)
	c.JSON(http.StatusOK, u)
}

func (h *UserHandler) Update(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var user request.UpdateUserRequest
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err = h.service.UpdateUser(c, id, user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updatedUser, err := h.service.ReadUser(c, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var response response.UserResponse
	response.FillInTheModel(updatedUser)
	c.JSON(http.StatusOK, response)
}

func (h *UserHandler) Delete(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.service.DeleteUser(c, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, id)
}

func (h *UserHandler) ReadUsers(c *gin.Context) {
	page := int64(1)
	limit := int64(10)

	pageStr := c.Query("page")

	if pageStr != "" {
		page, _ = strconv.ParseInt(pageStr, 10, 64)
	}

	if page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "page is incorrect"})
		return
	}

	limitStr := c.Query("limit")
	if limitStr != "" {
		limit, _ = strconv.ParseInt(limitStr, 10, 64)
	}
	if limit < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit is incorrect"})
		return
	}

	users, total, err := h.service.ReadUserList(c, limit, page)
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	var usersReponse []response.UserResponse
	for _, user := range users {
		var u response.UserResponse
		u.FillInTheModel(&user)
		usersReponse = append(usersReponse, u)
	}
	c.JSON(http.StatusOK, response.UsersListReponse{
		Users: usersReponse,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

func (h *UserHandler) GetAuthUser(c *gin.Context) {
	userUUID := c.MustGet("user").(uuid.UUID)
	user, err := h.service.ReadUser(c, userUUID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err.Error())
		return
	}
	var resp response.UserResponse
	resp.FillInTheModel(user)
	c.JSON(http.StatusOK, resp)
}
