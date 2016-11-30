package irc

import (
	"database/sql"
)

// Insert a new channel record into the channels table. Also inserts a new
// user-channel relationship record in the user_channel table. Returns nil on
// success, error otherwise.
func CreateChannel(db *sql.DB, channelName string, creator int64) error {
	//TODO check input before executing
	// Create new channel record
	_, err := db.Exec("INSERT INTO channels (channel_name,creator) VALUES (?,?)", channelName, creator)
	if err != nil {
		return err
	}
	// Create new user-channel relationship
	_, err = db.Exec("INSERT INTO user_channel (user_id,channel_name) VALUES (?,?)", creator, channelName)
	if err != nil {
		return err
	}
	return nil
}

// Removes all user-channel relationship records from user_channel table before
// deleting the channel from the channels table. Returns nil on success, error
// otherwise.
func DestroyChannel(db *sql.DB, channelName string) error {
	_, err := db.Exec("DELETE user_channel FROM user_channel INNER JOIN channels ON channels.channel_name = user_channel.channel_name WHERE channels.channel_name = ?", channelName)
	if err != nil {
		return err
	}
	_, err = db.Exec("DELETE FROM channels WHERE channel_name=?", channelName)
	if err != nil {
		return err
	}
	return nil
}

func JoinChannel(db *sql.DB, channelName string, userid int64) error {
	return nil
}
