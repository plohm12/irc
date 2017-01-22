package database

import (
	"database/sql"
	"errors"
	"fmt"
	"irc"
)

// Insert a new channel record into the channels table. Also inserts a new
// user-channel relationship record in the user_channel table. Returns nil on
// success, error otherwise.
func CreateChannel(channelName string, creator int64) error {
	//TODO check input before executing
	// Create new channel record
	_, err := db.Exec("INSERT INTO "+TABLE_CHANNELS+" (channel_name,creator) VALUES (?,?)", channelName, creator)
	if err != nil {
		return errors.New(string(irc.SERVER_PREFIX + "  " + irc.ERR_GENERAL + " :" + err.Error() + irc.CRLF))
	}
	// Create new user-channel relationship
	_, err = db.Exec("INSERT INTO "+TABLE_USER_CHANNEL+" (user_id,channel_name) VALUES (?,?)", creator, channelName)
	if err != nil {
		return errors.New(string(irc.SERVER_PREFIX + "  " + irc.ERR_GENERAL + " :" + err.Error() + irc.CRLF))
	}
	return nil
}

// Removes all user-channel relationship records from user_channel table before
// deleting the channel from the channels table. Returns nil on success, error
// otherwise.
func DestroyChannel(channelName string) error {
	_, err := db.Exec("DELETE FROM "+TABLE_USER_CHANNEL+" WHERE channel_name=?", channelName)
	if err != nil {
		// will return error if PartChannel() removed last user
		//return errors.New(string(irc.SERVER_PREFIX + "  " + irc.ERR_GENERAL + " :" + err.Error() + irc.CRLF))
	}
	_, err = db.Exec("DELETE FROM "+TABLE_CHANNELS+" WHERE channel_name=?", channelName)
	if err != nil {
		return errors.New(string(irc.SERVER_PREFIX + "  " + irc.ERR_GENERAL + " :" + err.Error() + irc.CRLF))
	}
	return nil
}

func JoinChannel(channelName string, userid int64) (string, error) {
	var topic string
	err := db.QueryRow("SELECT topic FROM "+TABLE_CHANNELS+" WHERE channel_name=?", channelName).Scan(&topic)
	if err == sql.ErrNoRows {
		// Create the channel
		fmt.Println("New channel", channelName)
		err = CreateChannel(channelName, userid)
		if err != nil {
			return "", errors.New(string(irc.SERVER_PREFIX + "  " + irc.ERR_GENERAL + " :" + err.Error() + irc.CRLF))
		}
	} else if err != nil {
		return "", errors.New(string(irc.SERVER_PREFIX + "  " + irc.ERR_GENERAL + " :" + err.Error() + irc.CRLF))
	} else {
		// Add user relationship
		fmt.Println("User", userid, "joining", channelName)
		_, err = db.Exec("INSERT INTO "+TABLE_USER_CHANNEL+" (user_id,channel_name) VALUES (?,?)", userid, channelName)
		if err != nil {
			return "", errors.New(string(irc.SERVER_PREFIX + "  " + irc.ERR_GENERAL + " :" + err.Error() + irc.CRLF))
		}
	}
	return topic, nil
}

func PartChannel(channelName string, userid int64) error {
	//TODO query if user is a member of channel
	_, err := db.Exec("DELETE FROM "+TABLE_USER_CHANNEL+" WHERE channel_name=? AND user_id=?", channelName, userid)
	if err != nil {
		return errors.New(string(irc.SERVER_PREFIX + "  " + irc.ERR_GENERAL + " :" + err.Error() + irc.CRLF))
	}
	rows, err := db.Query("SELECT user_id FROM "+TABLE_USER_CHANNEL+" WHERE channel_name=?", channelName)
	if err != nil {
		return errors.New(string(irc.SERVER_PREFIX + "  " + irc.ERR_GENERAL + " :" + err.Error() + irc.CRLF))
	}
	defer rows.Close()
	if !rows.Next() {
		// No more users in the channel, remove it
		return DestroyChannel(channelName)
	}

	return nil
}