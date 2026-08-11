package api

import (
	"errors"
	"fmt"
	"strings"

	"github.com/faelic/monierave/db/util"
	"github.com/go-playground/validator/v10"
)

var validCurrency validator.Func = func(fieldLevel validator.FieldLevel) bool {
	if currency, ok := fieldLevel.Field().Interface().(string); ok {
		return util.IsSupportedCurrency(currency)
	}
	return false
}

func friendlyValidationError(err error) error {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return err
	}

	messages := make([]string, 0, len(ve))

	for _, fe := range ve {
		field := displayFieldName(toSnakeCase(fe.Field()))
		messages = append(messages, validationMessage(field, fe))
	}

	return errors.New(strings.Join(messages, ", "))
}

func validationMessage(field string, fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)

	case "alphanum":
		return fmt.Sprintf("%s can only contain letters and numbers", field)

	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)

	case "currency":
		return fmt.Sprintf("%s is not a supported currency", field)

	case "min":
		switch fe.Field() {
		case "Password":
			return fmt.Sprintf("%s must be at least %s characters", field, fe.Param())
		default:
			return fmt.Sprintf("%s must be at least %s", field, fe.Param())
		}

	case "max":
		return fmt.Sprintf("%s must be at most %s", field, fe.Param())

	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, fe.Param())

	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}

func displayFieldName(field string) string {
	switch field {
	case "username":
		return "username"
	case "password":
		return "password"
	case "full_name":
		return "full name"
	case "email":
		return "email"
	case "currency":
		return "currency"
	case "amount":
		return "amount"
	case "balance":
		return "balance"
	case "page_id":
		return "page id"
	case "page_size":
		return "page size"
	case "id":
		return "id"
	case "from_account_id":
		return "from account id"
	case "to_account_number":
		return "to account number"
	case "destination_account_number":
		return "destination account number"
	default:
		return strings.ReplaceAll(field, "_", " ")
	}
}

func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}
