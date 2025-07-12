package types

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

/*
type Card struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string        `bson:"name" json:"name"`
	Type        string        `bson:"type" json:"type"`
	SubType     string        `bson:"subtype" json:"subtype"`
	Color       string        `bson:"color" json:"color"`
	ManaValue   int           `bson:"mana_value" json:"mana_value"`
	Power       int           `bson:"power" json:"power"`
	Toughness   int           `bson:"toughness" json:"toughness"`
	Rarity      string        `bson:"rarity" json:"rarity"`
	Description string        `bson:"description" json:"description"`
	ImageURL    string        `bson:"image_url" json:"image_url"`
}
type CardPayload struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	SubType     string `json:"subtype"`
	Color       string `json:"color"`
	ManaValue   int    `json:"mana_value"`
	Power       int    `json:"power"`
	Toughness   int    `json:"toughness"`
	Rarity      string `json:"rarity"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
}
*/

type Card struct {
	Name      string   `bson:"name" json:"name" validate:"required,min=1,max=100"`
	Colors    []string `bson:"colors" json:"colors" validate:"required,min=1,dive,oneof=White Blue Black Red Green Colorless"`
	ManaValue int      `bson:"mana_value" json:"mana_value" validate:"required,min=0,max=20"`
	Type      string   `bson:"type" json:"type" validate:"required,oneof=Creature Instant Sorcery Artifact Enchantment Land Planeswalker"`
}

type CardAmount struct {
	Card   Card `bson:"card" json:"card"`
	Amount int  `bson:"amount" json:"amount"`
}

type DeckCommander struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	OwnerID     bson.ObjectID `bson:"owner_id" json:"owner_id"`
	Name        string        `bson:"name" json:"name"`
	Description string        `bson:"description" json:"description"`
	DeckSize    int           `bson:"deck_size" json:"deck_size"`
	Commander   Card          `bson:"commander" json:"commander"`
	Partner     Card          `bson:"partner,omitempty" json:"partner,omitempty"`
	Format      string        `bson:"format" json:"format"`
	Cards       []CardAmount  `bson:"cards" json:"cards"`
	CreatedAt   bson.DateTime `bson:"created_at" json:"created_at"`
	UpdatedAt   bson.DateTime `bson:"updated_at" json:"updated_at"`
}

type Deck60Size struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	OwnerID     bson.ObjectID `bson:"owner_id" json:"owner_id"`
	Name        string        `bson:"name" json:"name"`
	Description string        `bson:"description" json:"description"`
	DeckSize    int           `bson:"deck_size" json:"deck_size"`
	Format      string        `bson:"format" json:"format"`
	Cards       []CardAmount  `bson:"cards" json:"cards"`
	CreatedAt   bson.DateTime `bson:"created_at" json:"created_at"`
	UpdatedAt   bson.DateTime `bson:"updated_at" json:"updated_at"`
}

type DecksList struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	OwnerID     bson.ObjectID `bson:"owner_id" json:"owner_id"`
	Name        string        `bson:"name" json:"name"`
	DeckSize    int           `bson:"deck_size" json:"deck_size"`
	Commander   Card          `bson:"commander" json:"commander"`
	Format      string        `bson:"format" json:"format"`
}

type DeckPayload struct {
	OwnerID     bson.ObjectID `json:"owner_id,omitempty" validate:"omitempty"`
	Name        string        `json:"name" validate:"required,min=3,max=50"`
	Description string        `json:"description" validate:"max=300"`
	DeckSize    int           `json:"deck_size" validate:"required"`
	Commander   *Card          `json:"commander,omitempty"`
	Partner     Card          `json:"partner,omitempty" validate:"omitempty"`
	Format      string        `json:"format" validate:"required,oneof=Commander Standard Modern Pioneer Legacy Vintage Brawl Historic Pauper"`
	Cards       []CardAmount  `json:"cards" validate:"required,dive,required"`
}

type User struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Username  string        `bson:"username" json:"username"`
	Email     string        `bson:"email" json:"email"`
	Password  string        `bson:"password" json:"password"`
	CreatedAt bson.DateTime `bson:"created_at" json:"created_at"`
	UpdatedAt bson.DateTime `bson:"updated_at" json:"updated_at"`
}

type UpdateUserPayload struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6,max=50"`
}

type LoginPayload struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required,min=6,max=50"`
}

type RegisterPayload struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6,max=50"`
}
