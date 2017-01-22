package irc

const BUFFER_SIZE int = 512
const CHAN_BUF_SIZE int = 10 // buffer size for Go channels
const DEFAULT_USER string = "guest"
const CRLF string = "\r\n"

/* Connection information */
const NETWORK string = "tcp"
const HOST_IP string = "127.0.0.1"
const HOST_PORT string = "6667"
const HOST_ADDRESS string = HOST_IP + ":" + HOST_PORT
const SERVER_PREFIX string = ":" + HOST_IP

/* IRC Reply codes per RFC 2812 */
const RPL_NOP string = "000" //nop
const RPL_WELCOME string = "001"
const RPL_YOURHOST string = "002"
const RPL_CREATED string = "003"
const RPL_MYINFO string = "004"
const RPL_BOUNCE string = "005"

const RPL_TRACELINK string = "200"
const RPL_TRACECONNECTING string = "201"
const RPL_TRACEHANDSHAKE string = "202"
const RPL_TRACEUNKNOWN string = "203"
const RPL_TRACEOPERATOR string = "204"
const RPL_TRACEUSER string = "205"
const RPL_TRACESERVER string = "206"
const RPL_TRACESERVICE string = "207"
const RPL_TRACENEWTYPE string = "208"
const RPL_TRACECLASS string = "209"
const RPL_TRACERECONNECT string = "210"
const RPL_STATSLINKINFO string = "211"
const RPL_STATSCOMMANDS string = "212"
const RPL_ENDOFSTATS string = "219"
const RPL_UMODEIS string = "221"
const RPL_SERVLIST string = "234"
const RPL_SERVLISTEND string = "235"
const RPL_STATSUPTIME string = "242"
const RPL_STATSOLINE string = "243"
const RPL_LUSERCLIENT string = "251"
const RPL_LUSEROP string = "252"
const RPL_LUSERUNKNOWN string = "253"
const RPL_LUSERCHANNELS string = "254"
const RPL_LUSERME string = "255"
const RPL_ADMINME string = "256"
const RPL_ADMINLOC1 string = "257"
const RPL_ADMINLOC2 string = "258"
const RPL_ADMINEMAIL string = "259"
const RPL_TRACELOG string = "261"
const RPL_TRACEEND string = "262"
const RPL_TRYAGAIN string = "263"

const RPL_AWAY string = "301"
const RPL_USERHOST string = "302"
const RPL_ISON string = "303"
const RPL_UNAWAY string = "305"
const RPL_NOWAWAY string = "306"
const RPL_WHOISUSER string = "311"
const RPL_WHOISSERVER string = "312"
const RPL_WHOISOPERATOR string = "313"
const RPL_WHOWASUSER string = "314"
const RPL_ENDOFWHO string = "315"
const RPL_WHOISIDLE string = "317"
const RPL_ENDOFWHOIS string = "318"
const RPL_WHOISCHANNELS string = "319"
const RPL_LISTSTART string = "321"
const RPL_LIST string = "322"
const RPL_LISTEND string = "323"
const RPL_CHANNELMODEIS string = "324"
const RPL_UNIQOPIS string = "325"
const RPL_NOTOPIC string = "331"
const RPL_TOPIC string = "332"
const RPL_INVITING string = "341"
const RPL_SUMMONING string = "342"
const RPL_INVITELIST string = "346"
const RPL_ENDOFINVITELIST string = "347"
const RPL_EXCEPTLIST string = "348"
const RPL_ENDOFEXCEPTLIST string = "349"
const RPL_VERSION string = "351"
const RPL_WHOREPLY string = "352"
const RPL_NAMREPLY string = "353"
const RPL_LINKS string = "364"
const RPL_ENDOFLINKS string = "365"
const RPL_ENDOFNAMES string = "366"
const RPL_BANLIST string = "367"
const RPL_ENDOFBANLIST string = "368"
const RPL_ENDOFWHOWAS string = "369"
const RPL_INFO string = "371"
const RPL_MOTD string = "372"
const RPL_ENDOFINFO string = "374"
const RPL_MOTDSTART string = "375"
const RPL_ENDOFMOTD string = "376"
const RPL_YOUREOPER string = "381"
const RPL_REHASHING string = "382"
const RPL_YOURESERVICE string = "383"
const RPL_TIME string = "391"
const RPL_USERSSTART string = "392"
const RPL_USERS string = "393"
const RPL_ENDOFUSERS string = "394"
const RPL_NOUSERS string = "395"

/* IRC Error codes per RFC 2812 */
const ERR_CONNCLOSED string = "400" //connection closed
const ERR_NOSUCHNICK string = "401"
const ERR_NOSUCHSERVER string = "402"
const ERR_NOSUCHCHANNEL string = "403"
const ERR_CANNOTSENDTOCHAN string = "404"
const ERR_TOOMANYCHANNELS string = "405"
const ERR_WASNOSUCHNICK string = "406"
const ERR_TOOMANYTARGETS string = "407"
const ERR_NOSUCHSERVICE string = "408"
const ERR_NOORIGIN string = "409"
const ERR_NORECIPIENT string = "411"
const ERR_NOTEXTTOSEND string = "412"
const ERR_NOTOPLEVEL string = "413"
const ERR_WILDTOPLEVEL string = "414"
const ERR_BADMASK string = "415"
const ERR_UNKNOWNCOMMAND string = "421"
const ERR_NOMOTD string = "422"
const ERR_NOADMININFO string = "423"
const ERR_FILEERROR string = "424"
const ERR_NONICKNAMEGIVEN string = "431"
const ERR_ERRONEUSNICKNAME string = "432"
const ERR_NICKNAMEINUSE string = "433"
const ERR_NICKCOLLISION string = "436"
const ERR_UNAVAILRESOURCE string = "437"
const ERR_USERNOTINCHANNEL string = "441"
const ERR_NOTONCHANNEL string = "442"
const ERR_USERONCHANNEL string = "443"
const ERR_NOLOGIN string = "444"
const ERR_SUMMONDISABLED string = "445"
const ERR_USERDISABLED string = "446"
const ERR_NOTREGISTERED string = "451"
const ERR_NEEDMOREPARAMS string = "461"
const ERR_ALREADYREGISTRED string = "462"
const ERR_NOPERMFORHOST string = "463"
const ERR_PASSWDMISMATCH string = "464"
const ERR_YOUREBANNEDCREEP string = "465"
const ERR_YOUWILLBEBANNED string = "466"
const ERR_KEYSET string = "467"
const ERR_CHANNELISFULL string = "471"
const ERR_UNKNOWNMODE string = "472"
const ERR_INVITEONLYCHAN string = "473"
const ERR_BANNEDFROMCHAN string = "474"
const ERR_BADCHANNELKEY string = "475"
const ERR_BADCHANMASK string = "476"
const ERR_NOCHANMODES string = "477"
const ERR_BANLISTFULL string = "478"
const ERR_NOPRIVILEGES string = "481"
const ERR_CHANOPRIVSNEEDED string = "482"
const ERR_CANTKILLSERVER string = "483"
const ERR_RESTRICTED string = "484"
const ERR_UNIQOPPRIVSNEEDED string = "485"
const ERR_NOOPERHOST string = "491"

const ERR_GENERAL string = "500" // general error reply
const ERR_UMODEUNKNOWNFLAG string = "501"
const ERR_USERSDONTMATCH string = "502"
