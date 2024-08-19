package deck

import (
	"database/sql"
	"deck-builder-back/pkg/types"
	"fmt"
)

type StoreDB struct{
	db *sql.DB
}

func NewStore(cfg *sql.DB) *StoreDB{
	return &StoreDB{db: cfg}
}

func (s *StoreDB) CreateUser(deck types.Deck) error{
	_, err:= s.db.Exec("INSERT INTO decks (name, format, deck_size, created_by, deck_list) VALUES(?,?,?,?,?);",deck.Name,deck.Format,deck.DeckSize,deck.CreatedBy,deck.DeckList)
	if err != nil{
		return err
	}
	return nil
}

func (s *StoreDB) GetDeckByName(name string) (*types.Deck, error){
	rows, err := s.db.Query("SELECT * FROM decks WHERE name = ?",name)
	if err != nil{
		return nil, err
	}
	deck := new(types.Deck)
	for rows.Next(){
		deck,err = scanRowsIntoDeck(rows)
		if err != nil{
			return nil, err
		}
	}
	if deck.ID==0{
		return nil,fmt.Errorf("deck not found")
	}
	return deck,nil
}

func(s *StoreDB) GetDecksByUser(id int)([]*types.Deck,error){
	rows, err := s.db.Query("SELECT * FROM decks WHERE created_by = ?",id)
	if err != nil{
		return nil, err
	}
	decks := []*types.Deck{}
	deck := new(types.Deck)
	for rows.Next(){
		deck,err = scanRowsIntoDeck(rows)
		decks = append(decks,deck)
		if err != nil{
			return nil, err
		}
	}
	if decks == nil{
		return nil, fmt.Errorf("User has no decks")
	}
	return decks,nil
}



func scanRowsIntoDeck(rows *sql.Rows) (*types.Deck, error){
	deck := new(types.Deck)

	err := rows.Scan(
		&deck.ID,
		&deck.Name,
		&deck.Format,
		&deck.DeckSize,
		&deck.CreatedBy,
		&deck.DeckList,
	)
	if err != nil {
		return nil, err
	}
	return deck, nil
}