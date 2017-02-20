package database

import (
	"database/sql"
	"irc/message"
)

var (
	s_NewUser         *sql.Stmt
	s_DeleteUser      *sql.Stmt
	s_GetPass         *sql.Stmt
	s_SetPass         *sql.Stmt
	s_GetNick         *sql.Stmt
	s_SetNick         *sql.Stmt
	s_GetNickUser     *sql.Stmt
	s_GetUserReal     *sql.Stmt
	s_SetUserModeReal *sql.Stmt
	s_GetIdByNick     *sql.Stmt
)

func CreateUser() Id {
	dbResult, err := s_NewUser.Exec()
	if err != nil {
		panic(err)
	}
	id, err := dbResult.LastInsertId()
	if err != nil {
		panic(err)
	}
	return Id(id)
}

func DeleteUser(id Id) {
	_, err := s_DeleteUser.Exec(id)
	if err != nil {
		panic(err)
	}
}

func (id Id) GetPassword() (password string, ok bool) {
	ok = true
	err := s_GetPass.QueryRow(id).Scan(&password)
	if err == sql.ErrNoRows {
		ok = false
	} else if err != nil {
		panic(err)
	}
	return
}

func (id Id) SetPassword(password message.Param) {
	pw := password.ToString()
	_, err := s_SetPass.Exec(pw, id)
	if err != nil {
		panic(err)
	}
}

func (id Id) GetNickname() (nickname string, ok bool) {
	ok = true
	err := s_GetNick.QueryRow(id).Scan(&nickname)
	if err == sql.ErrNoRows {
		ok = false
	} else if err != nil {
		panic(err)
	}
	return
}

func (id Id) SetNickname(nickname message.Param) {
	nn := nickname.ToString()
	_, err := s_SetNick.Exec(nn, id)
	if err != nil {
		panic(err)
	}
}

func (id Id) GetNicknameUsername() (nickname, username string, ok bool) {
	ok = true
	err := s_GetNickUser.QueryRow(id).Scan(&nickname, &username)
	if err == sql.ErrNoRows {
		ok = false
	} else if err != nil {
		panic(err)
	}
	return
}

func (id Id) GetUsernameRealname() (username, realname string, ok bool) {
	ok = true
	err := s_GetUserReal.QueryRow(id).Scan(&username, &realname)
	if err == sql.ErrNoRows {
		ok = false
	} else if err != nil {
		panic(err)
	}
	return
}

func (id Id) SetUsernameModeRealname(username, realname message.Param, mode int) {
	un := username.ToString()
	rn := realname.ToString()
	_, err := s_SetUserModeReal.Exec(un, mode, rn, id)
	if err != nil {
		panic(err)
	}
}

func GetIdByNickname(nickname message.Param) (id Id, ok bool) {
	ok = true
	nn := nickname.ToString()
	err := s_GetIdByNick.QueryRow(nn).Scan(&id)
	if err == sql.ErrNoRows {
		ok = false
	} else if err != nil {
		panic(err)
	}
	return
}

func prepareUserStatements() {
	var err error
	s_NewUser, err = db.Prepare("INSERT INTO " + TABLE_USERS + " () VALUES();")
	if err != nil {
		panic(err)
	}
	s_DeleteUser, err = db.Prepare("DELETE FROM " + TABLE_USERS + " WHERE id=?")
	if err != nil {
		panic(err)
	}
	s_GetPass, err = db.Prepare("SELECT password FROM " + TABLE_USERS + " WHERE id=?")
	if err != nil {
		panic(err)
	}
	s_SetPass, err = db.Prepare("UPDATE " + TABLE_USERS + " SET password=? WHERE id=?")
	if err != nil {
		panic(err)
	}
	s_GetNick, err = db.Prepare("SELECT nickname FROM " + TABLE_USERS + " WHERE id=?")
	if err != nil {
		panic(err)
	}
	s_SetNick, err = db.Prepare("UPDATE " + TABLE_USERS + " SET nickname=? WHERE id=?")
	if err != nil {
		panic(err)
	}
	s_GetNickUser, err = db.Prepare("SELECT nickname,username FROM " + TABLE_USERS + " WHERE id=?")
	if err != nil {
		panic(err)
	}
	s_GetUserReal, err = db.Prepare("SELECT username,realname FROM " + TABLE_USERS + " WHERE id=?")
	if err != nil {
		panic(err)
	}
	s_SetUserModeReal, err = db.Prepare("UPDATE " + TABLE_USERS + " SET username=?,mode=?,realname=? WHERE id=?")
	if err != nil {
		panic(err)
	}
	s_GetIdByNick, err = db.Prepare("SELECT id FROM " + TABLE_USERS + " WHERE nickname=?")
	if err != nil {
		panic(err)
	}
}

func closeUserStatements() {
	s_NewUser.Close()
	s_DeleteUser.Close()
	s_GetPass.Close()
	s_SetPass.Close()
	s_GetNick.Close()
	s_SetNick.Close()
	s_GetNickUser.Close()
	s_GetUserReal.Close()
	s_SetUserModeReal.Close()
	s_GetIdByNick.Close()
}
