package deck

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Romma711/deck-builder/pkg/config"
	"github.com/Romma711/deck-builder/pkg/types"
	"github.com/Romma711/deck-builder/pkg/utils"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func CreateDeck(deckPayload types.DeckPayload, userID string) (any, error) {
	var err error

	if err = utils.StructValidator(deckPayload); err != nil {
		return nil, err
	}

	deckPayload.OwnerID, err = bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	deck, err := parseDeckByFormat(deckPayload)
	if err != nil {
		return nil, err
	}
	ctx, cancel := config.GetContext()
	defer cancel()
	collection := config.GetCollection("decks")
	_, err = collection.InsertOne(ctx, deck)
	if err != nil {
		return nil, err
	}

	return &deck, nil
}

func GetAllDecks() ([]types.DecksList, error) {
	ctx, cancel := config.GetContext()
	defer cancel()
	collection := config.GetCollection("decks")

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var decks []types.DecksList
	for cursor.Next(ctx) {
		var deck types.DecksList
		if err := cursor.Decode(&deck); err != nil {
			return nil, err
		}
		decks = append(decks, deck)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return decks, nil
}

func GetDecksByCommander(commander string) ([]types.DecksList, error) {
	ctx, cancel := config.GetContext()
	defer cancel()
	collection := config.GetCollection("decks")

	cursor, err := collection.Find(ctx, bson.M{"commander.name": commander})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var decks []types.DecksList
	for cursor.Next(ctx) {
		var deck types.DecksList
		if err := cursor.Decode(&deck); err != nil {
			return nil, err
		}
		decks = append(decks, deck)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return decks, nil
}

func GetDecksByFormat(format string) ([]types.DecksList, error) {
	log.Println("Buscando mazos por formato:", format)
	ctx, cancel := config.GetContext()
	defer cancel()
	collection := config.GetCollection("decks")
	cursor, err := collection.Find(ctx, bson.M{"format": format})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var decks []types.DecksList
	for cursor.Next(ctx) {
		var deck types.DecksList
		if err := cursor.Decode(&deck); err != nil {
			return nil, err
		}
		decks = append(decks, deck)
	}	

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return decks, nil
}

func GetDecksByOwnerID(ownerID string) ([]types.DecksList, error) {
	ctx, cancel := config.GetContext()
	defer cancel()
	collection := config.GetCollection("decks")

	ownerBson, err := utils.ParseObjectIDFromHex(ownerID)
	if err != nil {
		return nil, err
	}
	
	cursor, err := collection.Find(ctx, bson.M{"owner_id": ownerBson})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var decks []types.DecksList
	for cursor.Next(ctx) {
		var deck types.DecksList
		if err := cursor.Decode(&deck); err != nil {
			return nil, err
		}
		decks = append(decks, deck)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return decks, nil
}

func GetDeckByID(deckID string) (any, error) {
	ctx, cancel := config.GetContext()
	defer cancel()
	collection := config.GetCollection("decks")

	deckIDBson, err := utils.ParseObjectIDFromHex(deckID)
	if err != nil {
		return nil, err
	}

	var deck interface{}
	if err := collection.FindOne(ctx, bson.M{"_id": deckIDBson}).Decode(&deck); err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, errors.New("mazo no encontrado")
		}
		return nil, err
	}
	return &deck, nil
}

func UpdateDeck(deckID string, deckPayload types.DeckPayload) (*types.DeckCommander, error){
	ctx, cancel := config.GetContext()
	defer cancel()
	collection := config.GetCollection("decks")
	var err error
	if err = utils.StructValidator(deckPayload); err != nil {
		return nil, err
	}

	var update bson.M
	if deckPayload.Format != "commander" && deckPayload.Format != "brawl" {	
		update = bson.M{
			"name":        deckPayload.Name,
			"description": deckPayload.Description,
			"deck_size":   deckPayload.DeckSize,
			"format":     deckPayload.Format,
			"cards":      deckPayload.Cards,
			"updated_at": bson.NewDateTimeFromTime(time.Now()),
		}
	}
	if deckPayload.Format == "commander" || deckPayload.Format == "brawl" {
		if deckPayload.Commander == nil {
			return nil, errors.New("el campo 'commander' es obligatorio para el formato " + deckPayload.Format)
		}
			if err := utils.StructValidator(*deckPayload.Commander); err != nil {
			return nil, fmt.Errorf("validación del commander fallida: %v", err)
		}
		update = bson.M{
			"name":        deckPayload.Name,
			"description": deckPayload.Description,
			"deck_size":   deckPayload.DeckSize,
			"commander":   deckPayload.Commander,
			"partner":     deckPayload.Partner, // si no hay, queda vacío
			"format":      deckPayload.Format,
			"cards":       deckPayload.Cards,
			"updated_at":  bson.NewDateTimeFromTime(time.Now()),
		}
	}

	deckIDBson, err := utils.ParseObjectIDFromHex(deckID)
	if err != nil {
		return nil, err
	}

	_, err = collection.UpdateOne(ctx, bson.M{"_id": deckIDBson}, bson.M{"$set": update})
	if err != nil {
		return nil, err
	}

	var updatedDeck types.DeckCommander
	if err := collection.FindOne(ctx, bson.M{"_id": deckIDBson}).Decode(&updatedDeck); err != nil {
		return nil, err
	}

	return &updatedDeck, nil
}


func DeleteDeck(deck string) error {
	ctx, cancel := config.GetContext()
	defer cancel()
	collection := config.GetCollection("decks")

	deckIDBson, err := utils.ParseObjectIDFromHex(deck)
	if err != nil {
		return err
	}

	_, err = collection.DeleteOne(ctx, bson.M{"_id": deckIDBson})
	if err != nil {
		return err
	}

	return nil
}

func parseDeckByFormat(payload types.DeckPayload) (interface{}, error){
	now := bson.NewDateTimeFromTime(time.Now())

    format := strings.ToLower(payload.Format)

    switch format {
    case "commander", "brawl":
		if payload.Commander == nil {
			return nil, errors.New("el campo 'commander' es obligatorio para el formato " + payload.Format)
		}
			if err := utils.StructValidator(*payload.Commander); err != nil {
			return nil, fmt.Errorf("validación del commander fallida: %v", err)
		}

        deck := types.DeckCommander{
            ID:          bson.NewObjectID(),
            OwnerID:     payload.OwnerID,
            Name:        payload.Name,
            Description: payload.Description,
            DeckSize:    payload.DeckSize,
            Commander:   *payload.Commander,
            Partner:     payload.Partner, // si no hay, queda vacío
            Format:      payload.Format,
            Cards:       payload.Cards,
            CreatedAt:   now,
            UpdatedAt:   now,
        }
        return deck, nil

    case "standard", "modern", "pioneer", "legacy", "vintage", "historic", "pauper":
        deck := types.Deck60Size{
            ID:          bson.NewObjectID(),
            OwnerID:     payload.OwnerID,
            Name:        payload.Name,
            Description: payload.Description,
            DeckSize:    payload.DeckSize,
            Format:      payload.Format,
            Cards:       payload.Cards,
            CreatedAt:   now,
            UpdatedAt:   now,
        }
        return deck, nil

    default:
        return nil, errors.New("formato no reconocido: " + payload.Format)
    }
}