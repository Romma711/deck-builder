package utils

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Romma711/deck-builder/pkg/types"
	"github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/v2/bson"
)


func CreateJWToken(user types.User) (string, error){
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"authorized":true,
			"uuid": user.ID,
			"username": user.Username,
			"exp": time.Now().Add(24 * time.Hour).Unix(), // 24 hours expiration
		})
	tokenString, err := token.SignedString([]byte(os.Getenv("SECRET_SIGNING_KEY")))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func VerifyToken(tokenString string) error {
   token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
      return []byte(os.Getenv("SECRET_SIGNING_KEY")), nil
   })
  
   if err != nil {
      return err
   }
  
   if !token.Valid {
      return fmt.Errorf("invalid token")
   }
  
   return nil
}

func GetUserFromToken(tokenString string) (types.User, error){
	var user types.User

	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	if tokenString == "" {
		log.Println("Token string is empty")
		return user, fmt.Errorf("token string is empty")
	}
	
	err := VerifyToken(tokenString)
	if err != nil {
		return user, err
	}
	
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("SECRET_SIGNING_KEY")), nil
	})
	if err != nil {
		return user, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		uuid := claims["uuid"].(string)
		user.ID, err = bson.ObjectIDFromHex(uuid)
		if err != nil {
			return user, fmt.Errorf("invalid user ID in token: %v", err)
		} 
		user.Username = claims["username"].(string)
	} else {
		return user, fmt.Errorf("invalid token claims")
	}

	return user, nil
}