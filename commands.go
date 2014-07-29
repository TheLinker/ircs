package main

import (
	"fmt"
	"strings"
)

type response struct {
	text string
	argc int
}

// casos especiales
//        "RPL_USERHOST":          {"302 :*1%s *( ' ' %s )",},
//        "RPL_ISON":              {"303 :*1%s *( ' ' %s )",},
//        "RPL_WHOISCHANNELS":     {"319 %s :*( ( '@' / '+' ) %s ' ' )",},
//        "RPL_NAMREPLY":          {"353 ( '=' / '*' / '@' ) %s :[ '@' / '+' ] %s *( ' ' [ '@' / '+' ] %s )",},
var Responses = map[string]response{
	"RPL_WELCOME":           {"001 %s :Welcome to the Internet Relay Network %s!%s@%s", 4},
	"RPL_YOURHOST":          {"002 %s :Your host is %s, running version %s", 3},
	"RPL_CREATED":           {"003 %s :This server was created %s", 2},
	"RPL_MYINFO":            {"004 %s %s %s %s %s", 5},
	"RPL_BOUNCE":            {"005 %s :Try server %s, port %s", 3},
	"RPL_AWAY":              {"301 %s :%s", 2},
	"RPL_UNAWAY":            {"305 :You are no longer marked as being away", 0},
	"RPL_NOWAWAY":           {"306 :You have been marked as being away", 0},
	"RPL_WHOISUSER":         {"311 %s %s %s * :%s", 4},
	"RPL_WHOISSERVER":       {"312 %s %s :%s", 3},
	"RPL_WHOISOPERATOR":     {"313 %s :is an IRC operator", 1},
	"RPL_WHOISIDLE":         {"317 %s %s :seconds idle", 2},
	"RPL_ENDOFWHOIS":        {"318 %s :End of WHOIS list", 1},
	"RPL_WHOWASUSER":        {"314 %s %s %s * :%s", 4},
	"RPL_ENDOFWHOWAS":       {"369 %s :End of WHOWAS", 1},
	"RPL_LIST":              {"322 %s %s :%s", 3},
	"RPL_LISTEND":           {"323 :End of LIST", 0},
	"RPL_UNIQOPIS":          {"325 %s %s", 2},
	"RPL_CHANNELMODEIS":     {"324 %s %s %s", 3},
	"RPL_NOTOPIC":           {"331 %s :No topic is set", 1},
	"RPL_TOPIC":             {"332 %s %s :%s", 3},
	"RPL_INVITING":          {"341 %s %s", 2},
	"RPL_SUMMONING":         {"342 %s :Summoning user to IRC", 1},
	"RPL_INVITELIST":        {"346 %s %s", 2},
	"RPL_ENDOFINVITELIST":   {"347 %s :End of channel invite list", 1},
	"RPL_EXCEPTLIST":        {"348 %s %s", 2},
	"RPL_ENDOFEXCEPTLIST":   {"349 %s :End of channel exception list", 1},
	"RPL_VERSION":           {"351 %s.%s %s :%s", 4},
	"RPL_WHOREPLY":          {"352 %s %s %s %s %s ( 'H' / 'G' > ['*'] [ ( '@' / '+' ) ] :%s %s", 7},
	"RPL_ENDOFWHO":          {"315 %s :End of WHO list", 1},
	"RPL_ENDOFNAMES":        {"366 %s :End of NAMES list", 1},
	"RPL_LINKS":             {"364 %s %s :%s %s", 4},
	"RPL_ENDOFLINKS":        {"365 %s :End of LINKS list", 1},
	"RPL_BANLIST":           {"367 %s %s", 2},
	"RPL_ENDOFBANLIST":      {"368 %s :End of channel ban list", 1},
	"RPL_INFO":              {"371 :%s", 1},
	"RPL_ENDOFINFO":         {"374 :End of INFO list", 0},
	"RPL_MOTDSTART":         {"375 :- %s Message of the day -", 1},
	"RPL_MOTD":              {"372 :- %s", 1},
	"RPL_ENDOFMOTD":         {"376 :End of MOTD command", 0},
	"RPL_YOUREOPER":         {"381 :You are now an IRC operator", 0},
	"RPL_REHASHING":         {"382 %s :Rehashing", 1},
	"RPL_YOURESERVICE":      {"383 You are service %s", 1},
	"RPL_TIME":              {"391 %s :%s", 2},
	"RPL_USERSSTART":        {"392 :UserID Terminal  Host", 0},
	"RPL_USERS":             {"393 :%s %s %s", 3},
	"RPL_ENDOFUSERS":        {"394 :End of users", 0},
	"RPL_NOUSERS":           {"395 :Nobody logged in", 0},
	"RPL_TRACELINK":         {"200 Link %s %s %s V%s %s %s %s", 7},
	"RPL_TRACECONNECTING":   {"201 Try. %s %s", 2},
	"RPL_TRACEHANDSHAKE":    {"202 H.S. %s %s", 2},
	"RPL_TRACEUNKNOWN":      {"203 ???? %s [%s]", 2},
	"RPL_TRACEOPERATOR":     {"204 Oper %s %s", 2},
	"RPL_TRACEUSER":         {"205 User %s %s", 2},
	"RPL_TRACESERVER":       {"206 Serv %s %sS %sC %s %s@%s V%s", 7},
	"RPL_TRACESERVICE":      {"207 Service %s %s %s %s", 4},
	"RPL_TRACENEWTYPE":      {"208 %s 0 %s", 2},
	"RPL_TRACECLASS":        {"209 Class %s %s", 2},
	"RPL_TRACELOG":          {"261 File %s %s", 2},
	"RPL_TRACEEND":          {"262 %s %s :End of TRACE", 2},
	"RPL_STATSLINKINFO":     {"211 %s %s %s %s %s %s %s", 7},
	"RPL_STATSCOMMANDS":     {"212 %s %s %s %s", 4},
	"RPL_ENDOFSTATS":        {"219 %s :End of STATS report", 1},
	"RPL_STATSUPTIME":       {"242 :Server Up %d days %d:%02d:%02d", 4},
	"RPL_STATSOLINE":        {"243 O %s * %s", 2},
	"RPL_UMODEIS":           {"221 %s", 1},
	"RPL_SERVLIST":          {"234 %s %s %s %s %s %s", 6},
	"RPL_SERVLISTEND":       {"235 %s %s :End of service listing", 2},
	"RPL_LUSERCLIENT":       {"251 :There are %s users and %s services on %s servers", 3},
	"RPL_LUSEROP":           {"252 %s :operator(s) online", 1},
	"RPL_LUSERUNKNOWN":      {"253 %s :unknown connection(s)", 1},
	"RPL_LUSERCHANNELS":     {"254 %s :channels formed", 1},
	"RPL_LUSERME":           {"255 :I have %s clients and %s servers", 2},
	"RPL_ADMINME":           {"256 %s :Administrative info", 1},
	"RPL_ADMINLOC1":         {"257 :%s", 1},
	"RPL_ADMINLOC2":         {"258 :%s", 1},
	"RPL_ADMINEMAIL":        {"259 :%s", 1},
	"RPL_TRYAGAIN":          {"263 %s :Please wait a while and try again.", 1},
	"ERR_NOSUCHNICK":        {"401 %s :No such nick/channel", 1},
	"ERR_NOSUCHSERVER":      {"402 %s :No such server", 1},
	"ERR_NOSUCHCHANNEL":     {"403 %s :No such channel", 1},
	"ERR_CANNOTSENDTOCHAN":  {"404 %s :Cannot send to channel", 1},
	"ERR_TOOMANYCHANNELS":   {"405 %s :You have joined too many channels", 1},
	"ERR_WASNOSUCHNICK":     {"406 %s :There was no such nickname", 1},
	"ERR_TOOMANYTARGETS":    {"407 %s :%s recipients. %s", 3},
	"ERR_NOSUCHSERVICE":     {"408 %s :No such service", 1},
	"ERR_NOORIGIN":          {"409 :No origin specified", 0},
	"ERR_NORECIPIENT":       {"411 :No recipient given (%s)", 1},
	"ERR_NOTEXTTOSEND":      {"412 :No text to send", 0},
	"ERR_NOTOPLEVEL":        {"413 %s :No toplevel domain specified", 1},
	"ERR_WILDTOPLEVEL":      {"414 %s :Wildcard in toplevel domain", 1},
	"ERR_BADMASK":           {"415 %s :Bad Server/host mask", 1},
	"ERR_UNKNOWNCOMMAND":    {"421 %s :Unknown command", 1},
	"ERR_NOMOTD":            {"422 :MOTD File is missing", 0},
	"ERR_NOADMININFO":       {"423 %s :No administrative info available", 1},
	"ERR_FILEERROR":         {"424 :File error doing %s on %s", 2},
	"ERR_NONICKNAMEGIVEN":   {"431 :No nickname given", 0},
	"ERR_ERRONEUSNICKNAME":  {"432 %s :Erroneous nickname", 1},
	"ERR_NICKNAMEINUSE":     {"433 %s :Nickname is already in use", 1},
	"ERR_NICKCOLLISION":     {"436 %s :Nickname collision KILL from %s@%s", 3},
	"ERR_UNAVAILRESOURCE":   {"437 %s :Nick/channel is temporarily unavailable", 1},
	"ERR_USERNOTINCHANNEL":  {"441 %s %s :They aren't on that channel", 2},
	"ERR_NOTONCHANNEL":      {"442 %s :You're not on that channel", 1},
	"ERR_USERONCHANNEL":     {"443 %s %s :is already on channel", 2},
	"ERR_NOLOGIN":           {"444 %s :User not logged in", 1},
	"ERR_SUMMONDISABLED":    {"445 :SUMMON has been disabled", 0},
	"ERR_USERSDISABLED":     {"446 :USERS has been disabled", 0},
	"ERR_NOTREGISTERED":     {"451 :You have not registered", 0},
	"ERR_NEEDMOREPARAMS":    {"461 %s :Not enough parameters", 1},
	"ERR_ALREADYREGISTRED":  {"462 :Unauthorized command (already registered)", 0},
	"ERR_NOPERMFORHOST":     {"463 :Your host isn't among the privileged", 0},
	"ERR_PASSWDMISMATCH":    {"464 :Password incorrect", 0},
	"ERR_YOUREBANNEDCREEP":  {"465 :You are banned from this server", 0},
	"ERR_YOUWILLBEBANNED":   {"466", 0},
	"ERR_KEYSET":            {"467 %s :Channel key already set", 1},
	"ERR_CHANNELISFULL":     {"471 %s :Cannot join channel (+l)", 1},
	"ERR_UNKNOWNMODE":       {"472 %s :is unknown mode char to me for %s", 2},
	"ERR_INVITEONLYCHAN":    {"473 %s :Cannot join channel (+i)", 1},
	"ERR_BANNEDFROMCHAN":    {"474 %s :Cannot join channel (+b)", 1},
	"ERR_BADCHANNELKEY":     {"475 %s :Cannot join channel (+k)", 1},
	"ERR_BADCHANMASK":       {"476 %s :Bad Channel Mask", 1},
	"ERR_NOCHANMODES":       {"477 %s :Channel doesn't support modes", 1},
	"ERR_BANLISTFULL":       {"478 %s %s :Channel list is full", 2},
	"ERR_NOPRIVILEGES":      {"481 :Permission Denied- You're not an IRC operator", 0},
	"ERR_CHANOPRIVSNEEDED":  {"482 %s :You're not channel operator", 1},
	"ERR_CANTKILLSERVER":    {"483 :You can't kill a server!", 0},
	"ERR_RESTRICTED":        {"484 :Your connection is restricted!", 0},
	"ERR_UNIQOPPRIVSNEEDED": {"485 :You're not the original channel operator", 0},
	"ERR_NOOPERHOST":        {"491 :No O-lines for your host", 0},
	"ERR_UMODEUNKNOWNFLAG":  {"501 :Unknown MODE flag", 0},
	"ERR_USERSDONTMATCH":    {"502 :Cannot change mode for other users", 0},
}

func Replay(out chan string, prefix string, message string, argv ...interface{}) {
	var r string
	resp, ok := Responses[message]
	if !ok {
		return
	}

	if len(argv) != resp.argc {
		return
	}

	if len(prefix) != 0 {
		r = ":" + strings.Trim(prefix, " :") + " "
	}

	r += resp.text
	r = fmt.Sprintf(r, argv...)

	out <- r
}

func SendUserList(u *User, prefix string, channel *Channel) {
	//usuarios conectados
	var nicks string
	for _, u := range channel.users {
		if u != nil {
			nicks += " " + u.nickname
		}
	}

	u.out <- fmt.Sprintf(":%s 353 %s = %s :%s", prefix, u.nickname, channel.name, strings.TrimLeft(nicks, " "))
	u.out <- fmt.Sprintf(":%s 366 %s %s :End of /NAMES list", prefix, u.nickname, channel.name)
}
