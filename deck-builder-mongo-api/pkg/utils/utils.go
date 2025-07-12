package utils

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func ParseObjectIDFromHex(hex string) (bson.ObjectID, error) {
	id, err := bson.ObjectIDFromHex(hex)
	if err != nil {
		return bson.ObjectID{}, fmt.Errorf("ID de usuario inválido: %v", err)
	}
	return id, nil
}

func StructValidator (payload any) error {
	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		errs := make(map[string]string)
		for _, fieldErr := range err.(validator.ValidationErrors) {
			errs[fieldErr.Field()] = fieldErr.Tag() // podés personalizar esto
		}
		return fmt.Errorf("Validación fallida: %v", errs)
	}
	return nil
}