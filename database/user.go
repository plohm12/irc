package database

func CreateUser() int64 {
	dbResult, err := s_NewUser.Exec()
	if err != nil {
		panic(err)
	}
	id, err := dbResult.LastInsertId()
	if err != nil {
		panic(err)
	}
	return id
}

func DeleteUser(id int64) error {
	if err != nil {
		return err
	}
	_, err = s_DeleteUser.Exec(id)
	if err != nil {
		return err
	}
	return nil
}
