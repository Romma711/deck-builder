package types

type User struct {
	ID        int    `json:"id"`
	Alias     string `json:"alias"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	ManyDecks string `json:"manydecks"`
}

type UserStore interface {
	GetUserByEmail(email string) (*User, error)
	GetUserById(id int) (*User, error)
	CreateUser(user User) error
}

type ToRegister struct {
	Alias    string `json:"alias" validate:"required"`
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type ToLogin struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type Deck struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Format    int    `json:"format"`
	DeckSize  int    `json:"deck_size"`
	CreatedBy int    `json:"created_by"`
	DeckList  string `json:"deck_list"`
}

type NewDeck struct {
	Name      string `json:"name"`
	Format    int    `json:"format"`
	DeckSize  int    `json:"deck_size"`
	CreatedBy int    `json:"created_by"`
	DeckList  string `json:"deck_list"`
}

type DeckStore interface {
	CreateDeck(deck NewDeck)error
	GetDeckByName(name string) (*Deck, error)
	GetDecksByUser(id int)([]*Deck, error)
}
