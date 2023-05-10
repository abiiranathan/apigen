package handlers

import (
	"errors"
	"net/http"
	"net/mail"
	"reflect"
	"strconv"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/gofiber/fiber/v2"
)

var ErrUnsupportedObject = errors.New("object must be struct, slice, array or their pointers")

type Validator struct {
	validator *validator.Validate
}

func NewValidator(tagName string) *Validator {
	val := validator.New()
	val.SetTagName(tagName)
	enLocale := en.New()
	uni := ut.New(enLocale, enLocale)

	trans, _ := uni.GetTranslator("en")
	en_translations.RegisterDefaultTranslations(val, trans)
	return &Validator{validator: val}
}

func (val *Validator) SetTagName(tagName string) {
	val.validator.SetTagName(tagName)
}

// Validates structs inside a slice, returns validator.ValidationErrors
// Not that it returns the first encountered error
func (val *Validator) validateSlice(value reflect.Value) error {
	count := value.Len()
	for i := 0; i < count; i++ {
		if err := val.validator.Struct(value.Index(i).Interface()); err != nil {
			return err.(validator.ValidationErrors)
		}
	}
	return nil
}

// Validates structs, pointers to structs and slices/arrays of structs
// Validate will panic if obj is not struct, slice, array or pointers to the same.
func (val *Validator) Validate(obj any) validator.ValidationErrors {
	value := reflect.ValueOf(obj)

	var err error
	switch value.Kind() {
	case reflect.Ptr:
		elem := value.Elem()
		switch reflect.ValueOf(elem.Interface()).Kind() {
		case reflect.Struct:
			err = val.validator.Struct(elem.Interface())
		case reflect.Slice, reflect.Array:
			err = val.validateSlice(elem)
		default:
			panic(ErrUnsupportedObject)
		}
	case reflect.Struct:
		err = val.validator.Struct(value.Interface())
	case reflect.Slice, reflect.Array:
		err = val.validateSlice(value)
	default:
		panic(ErrUnsupportedObject)
	}

	if err != nil {
		return err.(validator.ValidationErrors)
	}
	return nil
}

/*
This pattern uses a combination of character sets, character ranges,
and optional groups to match the structure of an email address.
It should match most valid email addresses, including ones with multiple dots
in the domain name and ones with internationalized domain names (IDNs).
*/
func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

var DefaultValidator = NewValidator("validate")
var ErrBadRequestBody = fiber.NewError(http.StatusBadRequest, "can not parse body of request, make sure you have set the correct content-type header")

// Validates v with go-playground validator.
// v should be pointer to struct, slice.
// If validation fails, it returns validator.ValidationErrors
//
// If the reading the body fails, it returns ErrBadRequestBody
func BodyParser(c *fiber.Ctx, v any) error {
	if err := c.BodyParser(v); err != nil {
		if errors.Is(err, fiber.ErrUnprocessableEntity) {
			return ErrBadRequestBody
		}
		return err
	}

	// Perform validation
	err := DefaultValidator.Validate(v)

	// !DANGER if this check is removed. Comparing err to nil
	// ! would be false even when validation did not fail, b'se its not a nil slice.
	// ! Wierd interface conversions between error type and slice of errors.
	if len(err) == 0 {
		return nil
	}
	return err

}

func GetParam(c *fiber.Ctx, key string) int {
	v := c.Params(key)
	value, _ := strconv.Atoi(v)
	return value
}

func GetQuery(c *fiber.Ctx, key string) int {
	v := c.Query(key)
	value, _ := strconv.Atoi(v)
	return value
}
