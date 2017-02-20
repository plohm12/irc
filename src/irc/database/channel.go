package database

import (
	"database/sql"
	"irc/message"
)

var (
	s_GetCreator         *sql.Stmt
	s_GetChanUser        *sql.Stmt
	s_GetChanUsers       *sql.Stmt
	s_GetChanTopic       *sql.Stmt
	s_CreateChan         *sql.Stmt
	s_CreateChanUser     *sql.Stmt
	s_DeleteChan         *sql.Stmt
	s_DeleteChanUser     *sql.Stmt
	s_DeleteAllChanUsers *sql.Stmt
)

// Insert a new channel record into the channels table. Also inserts a new
// user-channel relationship record in the user_channel table.
func CreateChannel(channelName message.Param, creator Id) {
	var err error
	//TODO check input before executing
	_, err = s_CreateChan.Exec(channelName, creator)
	if err != nil {
		panic(err)
	}
	_, err = s_CreateChanUser.Exec(creator, channelName)
	if err != nil {
		panic(err)
	}
}

// Removes all user-channel relationship records from user_channel table before
// deleting the channel from the channels table.
func DestroyChannel(channelName message.Param) {
	s_DeleteAllChanUsers.Exec(channelName)
	_, err := s_DeleteChan.Exec(channelName)
	if err != nil {
		panic(err)
	}
}

func JoinChannel(channelName message.Param, userid Id) string {
	var topic string
	var err error
	err = s_GetChanTopic.QueryRow(channelName).Scan(&topic)
	if err == sql.ErrNoRows {
		CreateChannel(channelName, userid)
	} else if err != nil {
		panic(err)
	} else {
		// Add user relationship
		_, err = s_CreateChanUser.Exec(userid, channelName)
		if err != nil {
			panic(err)
		}
	}
	return topic
}

func PartChannel(channelName message.Param, userid Id) {
	var err error
	_, err = s_DeleteChanUser.Exec(channelName, userid)
	if err != nil {
		panic(err)
	}
	rows, err := s_GetChanUsers.Query(channelName)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	if !rows.Next() {
		// No more users in the channel, remove it
		DestroyChannel(channelName)
	}
}

func GetChannelCreator(channel message.Param) (creator Id, ok bool) {
	ok = true
	err := s_GetCreator.QueryRow(channel).Scan(&creator)
	if err == sql.ErrNoRows {
		ok = false
	} else if err != nil {
		panic(err)
	}
	return
}

// Check if channel exists with a dummy query
func Check(channel message.Param) bool {
	_, ok := GetChannelCreator(channel)
	return ok
}

func (id Id) IsMemberOf(channel message.Param) bool {
	var dummy Id
	err := s_GetChanUser.QueryRow(channel, id).Scan(&dummy)
	if err == sql.ErrNoRows {
		return false
	} else if err != nil {
		panic(err)
	}
	return true
}

func GetChannelUsers(channel message.Param) (users []Id) {
	rows, err := s_GetChanUsers.Query(channel)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id Id
		rows.Scan(&id)
		users = append(users, id)
	}
	return
}

func prepareChannelStatements() {
	var err error
	s_GetCreator, err = db.Prepare("SELECT creator FROM " + TABLE_CHANNELS + " WHERE channel_name=?")
	if err != nil {
		panic(err)
	}
	s_GetChanUser, err = db.Prepare("SELECT user_id FROM " + TABLE_USER_CHANNEL + " WHERE channel_name=? AND user_id=?")
	if err != nil {
		panic(err)
	}
	s_GetChanUsers, err = db.Prepare("SELECT user_id FROM " + TABLE_USER_CHANNEL + " WHERE channel_name=?")
	if err != nil {
		panic(err)
	}
	s_GetChanTopic, err = db.Prepare("SELECT topic FROM " + TABLE_CHANNELS + " WHERE channel_name=?")
	if err != nil {
		panic(err)
	}
	s_CreateChan, err = db.Prepare("INSERT INTO " + TABLE_CHANNELS + " (channel_name,creator) VALUES (?,?)")
	if err != nil {
		panic(err)
	}
	s_CreateChanUser, err = db.Prepare("INSERT INTO " + TABLE_USER_CHANNEL + " (user_id,channel_name) VALUES (?,?)")
	if err != nil {
		panic(err)
	}
	s_DeleteChan, err = db.Prepare("DELETE FROM " + TABLE_CHANNELS + " WHERE channel_name=?")
	if err != nil {
		panic(err)
	}
	s_DeleteChanUser, err = db.Prepare("DELETE FROM " + TABLE_USER_CHANNEL + " WHERE channel_name=? AND user_id=?")
	if err != nil {
		panic(err)
	}
	s_DeleteAllChanUsers, err = db.Prepare("DELETE FROM " + TABLE_USER_CHANNEL + " WHERE channel_name=?")
	if err != nil {
		panic(err)
	}
}

func closeChannelStatements() {
	s_GetCreator.Close()
	s_GetChanUser.Close()
	s_GetChanUsers.Close()
	s_GetChanTopic.Close()
	s_CreateChan.Close()
	s_CreateChanUser.Close()
	s_DeleteChan.Close()
	s_DeleteChanUser.Close()
	s_DeleteAllChanUsers.Close()
}
