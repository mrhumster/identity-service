package handler

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetErrorMessage_Simple(t *testing.T) {
	validate := validator.New()

	t.Run("required tag", func(t *testing.T) {
		s := struct {
			Field string `validate:"required"`
		}{}
		err := validate.Struct(s)
		require.Error(t, err)
		fieldError := err.(validator.ValidationErrors)[0]
		assert.Equal(t, "This field is required", getErrorMessage(fieldError))
	})

	t.Run("email tag", func(t *testing.T) {
		s := struct {
			Field string `validate:"email"`
		}{Field: "invalid"}
		err := validate.Struct(s)
		require.Error(t, err)
		fieldError := err.(validator.ValidationErrors)[0]
		assert.Equal(t, "Invalid email format", getErrorMessage(fieldError))
	})

	t.Run("min tag", func(t *testing.T) {
		s := struct {
			Field string `validate:"min=3"`
		}{Field: "ab"}
		err := validate.Struct(s)
		require.Error(t, err)
		fieldError := err.(validator.ValidationErrors)[0]
		assert.Equal(t, "Value is too short", getErrorMessage(fieldError))
	})

	t.Run("max tag", func(t *testing.T) {
		s := struct {
			Field string `validate:"max=5"`
		}{Field: "abcdef"}
		err := validate.Struct(s)
		require.Error(t, err)
		fieldError := err.(validator.ValidationErrors)[0]
		assert.Equal(t, "Value is too long", getErrorMessage(fieldError))
	})

	t.Run("gte tag", func(t *testing.T) {
		s := struct {
			Field int `validate:"gte=10"`
		}{Field: 5}
		err := validate.Struct(s)
		require.Error(t, err)
		fieldError := err.(validator.ValidationErrors)[0]
		assert.Equal(t, "Value is too small", getErrorMessage(fieldError))
	})

	t.Run("lte tag", func(t *testing.T) {
		s := struct {
			Field int `validate:"lte=5"`
		}{Field: 10}
		err := validate.Struct(s)
		require.Error(t, err)
		fieldError := err.(validator.ValidationErrors)[0]
		assert.Equal(t, "Value is too large", getErrorMessage(fieldError))
	})

	t.Run("unknown tag - using numeric", func(t *testing.T) {
		s := struct {
			Field string `validate:"numeric"`
		}{Field: "abc"}
		err := validate.Struct(s)
		require.Error(t, err)
		fieldError := err.(validator.ValidationErrors)[0]
		assert.Contains(t, getErrorMessage(fieldError), "numeric")
	})
}
