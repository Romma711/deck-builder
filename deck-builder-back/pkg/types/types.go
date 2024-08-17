package types


type User struct {
	ID		  int	 `json:"id"`
	Alias     string `json:"alias"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	ManyDecks string `json:"manydecks"`
	
}

type UserStore interface{
	GetUserByEmail(email string) (*User, error)
	GetUserById(id int)(*User, error)
	CreateUser(user User) error
}

type ToRegister struct{
	Alias     string `json:"alias" validate:"required"`
	Email     string `json:"email" validate:"required"`
	Password  string `json:"password" validate:"required"`
}

type ToLogin struct{
	Email     string `json:"email" validate:"required"`
	Password  string `json:"password" validate:"required"`
}