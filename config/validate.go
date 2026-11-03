package config

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var tagValidator = func() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name, opts, _ := strings.Cut(f.Tag.Get("koanf"), ",")
		switch {
		case strings.Contains(opts, "squash"):
			return squashed
		case name == "":
			return f.Name
		}
		return name
	})
	return v
}()

func validate(cfg any) error {
	var errs []error
	var verrs validator.ValidationErrors
	if err := tagValidator.Struct(cfg); errors.As(err, &verrs) {
		for _, e := range verrs {
			errs = append(errs, fmt.Errorf("%s: %s", fieldPath(e), message(e)))
		}
	} else if err != nil {
		errs = append(errs, err)
	}
	if v, ok := cfg.(Validator); ok {
		errs = append(errs, v.Validate())
	}
	return errors.Join(errs...)
}

const squashed = "\x00"

func fieldPath(e validator.FieldError) string {
	_, path, _ := strings.Cut(e.Namespace(), ".")
	return strings.ReplaceAll(path, squashed+".", "")
}

var comparisons = map[string]string{"gt": ">", "gte": ">=", "lt": "<", "lte": "<=", "min": ">=", "max": "<="}

func message(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "required"
	case "url":
		return fmt.Sprintf("%q is not a valid URL", fmt.Sprint(e.Value()))
	case "oneof":
		return fmt.Sprintf("invalid %q (%s)", fmt.Sprint(e.Value()), strings.ReplaceAll(e.Param(), " ", " | "))
	case "gt", "gte", "lt", "lte", "min", "max":
		op := comparisons[e.Tag()]
		switch e.Kind() {
		case reflect.String, reflect.Slice, reflect.Map:
			return fmt.Sprintf("length must be %s %s (got %v)", op, e.Param(), e.Value())
		}
		return fmt.Sprintf("must be %s %s (got %v)", op, e.Param(), e.Value())
	}
	return fmt.Sprintf("rule %q not met (got %v)", e.ActualTag(), e.Value())
}
