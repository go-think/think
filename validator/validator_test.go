package validator

import (
	"testing"

	"github.com/go-think/think/contract"
	"github.com/stretchr/testify/assert"
)

type sampleUserForm struct {
	Name                 string `json:"name"`
	Email                string `json:"email"`
	Age                  int    `json:"age"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
	Role                 string `json:"role"`
}

func TestValidator_Passes(t *testing.T) {
	form := sampleUserForm{
		Name:                 "Taylor",
		Email:                "taylor@example.com",
		Age:                  25,
		Password:             "secret123",
		PasswordConfirmation: "secret123",
		Role:                 "admin",
	}

	rules := map[string]string{
		"name":     "required|min:2|max:50",
		"email":    "required|email",
		"age":      "required|integer|min:18",
		"password": "required|min:6|confirmed",
		"role":     "required|in:admin,user,manager",
	}

	validated, err := Validate(form, rules)
	assert.NoError(t, err)
	assert.NotNil(t, validated)
	assert.Equal(t, "Taylor", validated["name"])
	assert.Equal(t, "taylor@example.com", validated["email"])
}

func TestValidator_Fails(t *testing.T) {
	form := sampleUserForm{
		Name:                 "",
		Email:                "invalid-email",
		Age:                  15,
		Password:             "123",
		PasswordConfirmation: "mismatch",
		Role:                 "hacker",
	}

	rules := map[string]string{
		"name":     "required",
		"email":    "required|email",
		"age":      "required|integer|min:18",
		"password": "required|min:6|confirmed",
		"role":     "required|in:admin,user",
	}

	validated, err := Validate(form, rules)
	assert.Error(t, err)
	assert.Nil(t, validated)

	ve, ok := err.(*contract.ValidationException)
	assert.True(t, ok)
	assert.Equal(t, 422, ve.Status)
	assert.Contains(t, ve.Errors["name"][0], "required")
	assert.Contains(t, ve.Errors["email"][0], "valid email")
	assert.Contains(t, ve.Errors["age"][0], "at least 18")
	assert.Contains(t, ve.Errors["password"][0], "at least 6")
	assert.Contains(t, ve.Errors["password"][1], "confirmation does not match")
	assert.Contains(t, ve.Errors["role"][0], "invalid")
}

func TestValidator_CustomMessages(t *testing.T) {
	data := map[string]any{"email": "not-an-email"}
	rules := map[string]string{"email": "email"}
	messages := map[string]string{
		"email.email": "Custom: please provide a legitimate email!",
	}

	v := Make(data, rules, messages)
	assert.True(t, v.Fails())
	assert.Equal(t, "Custom: please provide a legitimate email!", v.Errors()["email"][0])
}
