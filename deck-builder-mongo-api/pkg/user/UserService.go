package user

import (
	"fmt"
	"time"

	"github.com/Romma711/deck-builder/pkg/config"
	"github.com/Romma711/deck-builder/pkg/types"
	"github.com/Romma711/deck-builder/pkg/utils"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func RegisterUser(register types.RegisterPayload) (*types.User, error){
	user := &types.User{
		ID: bson.NewObjectID(),
		CreatedAt: bson.NewDateTimeFromTime(time.Now()),
		UpdatedAt: bson.NewDateTimeFromTime(time.Now()),
		Username: register.Username,
		Email:    register.Email,
		Password: register.Password,
	}

	collection := config.GetCollection("users")
	ctx, cancel := config.GetContext()
	defer cancel()
	_, err := collection.InsertOne(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("fallo al registrar al usuario: %v", err)
	}

	return user, nil
}

func LoginUser(login types.LoginPayload) (string, error) {
	collection := config.GetCollection("users")
	ctx, cancel := config.GetContext()
	defer cancel()
	var user types.User
	err := collection.FindOne(ctx, bson.M{"username": login.Username}).Decode(&user)
	if err != nil {
		return "", fmt.Errorf("fallo al iniciar sesión: %v", err)
	}

	if user.Password != login.Password {
		return "", fmt.Errorf("contraseña incorrecta")
	}
	token, err := utils.CreateJWToken(user)
	if err != nil {
		return "", fmt.Errorf("fallo al crear el token JWT: %v", err)
	}
	if token == ""{
		return "", fmt.Errorf("token JWT vacío")
	}

	return token, nil
}

func UpdateUser(userID string, update types.UpdateUserPayload) (*types.User, error) {
	collection := config.GetCollection("users")
	ctx, cancel := config.GetContext()
	defer cancel()
	updateData := bson.M{
		"username": update.Username,
		"email":    update.Email,
		"updated_at": bson.NewDateTimeFromTime(time.Now()),
	}

	id, err := utils.ParseObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("ID de usuario inválido: %v", err)
	}

	_, err = collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updateData})
	if err != nil {
		return nil, fmt.Errorf("fallo al actualizar el usuario: %v", err)
	}

	var updatedUser types.User
	err = collection.FindOne(ctx, bson.M{"_id": id}).Decode(&updatedUser)
	if err != nil {
		return nil, fmt.Errorf("fallo al recuperar el usuario actualizado: %v", err)
	}

	return &updatedUser, nil
}

func DeleteUser(userID string) error {
	collection := config.GetCollection("users")
	ctx, cancel := config.GetContext()
	defer cancel()

	id, err := utils.ParseObjectIDFromHex(userID)
	if err != nil {
		return fmt.Errorf("ID de usuario inválido: %v", err)
	}
	
	_, err = collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("fallo al eliminar el usuario: %v", err)
	}

	return nil
}