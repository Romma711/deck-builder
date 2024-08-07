package model

import (
	"database/sql"
	"deck-builder-back/pkg/types"
)

type StoreDB struct{
	db *sql.DB
}

func NewStore(cfg *sql.DB) *StoreDB{
	return &StoreDB{db: cfg}
}

func (s *StoreDB) CreateUser(user types.User) error{
	_, err:= s.db.Query("INSERT INTO users (alias, email, password) VALUES(?,?,?)",user.Alias, user.Email, user.Password)
	if err != nil{
		return err
	}
	return nil
}

func (s *StoreDB) GetUserByEmail(email string) (*types.User, error){
	 rows, err := s.db.Query("SELECT * FROM users WHERE email = ?",email)
	 if err != nil{
		return nil, err
	 }
	 user := new(types.User)
	 for rows.Next(){
		user, err = scanRowsIntoUser(rows)
		if err != nil{
			return nil, err
		}
	 }
	 if user.ID == 0{
		return nil,err
	 }

	 return user,nil
	
}

func (s *StoreDB) GetUserById(id int)(*types.User, error){
	rows, err := s.db.Query("SELECT * FROM users WHERE id = ?",id)
	if err != nil{
		return nil,err
	}
	user :=new(types.User)
	for rows.Next(){
		user, err = scanRowsIntoUser(rows)
		if err != nil{
			return nil,err
		}
	}
	if user.ID == 0{
		return nil,err
	}
	return user,nil
}

func (s *StoreDB) UpdateUser(user *types.User) error {
	_,err:= s.db.Query("UPDATE users SET alias=?, email=?, password=?, manydeck=? WHERE id=?",user.Alias, user.Email, user.Password, user.ManyDecks, user.ID)
	if err != nil{
		return err
	}
	return nil
}

func scanRowsIntoUser(rows *sql.Rows) (*types.User, error){
	user := new(types.User)

	err:= rows.Scan(
		&user.ID,
		&user.Alias,
		&user.Email,
		&user.Password,
		&user.ManyDecks,
	)
	if err != nil{
		return nil, err
	}
	return user, nil
}