package main

import (
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
)

type MsgHandler func(c *User, prefix string, args string)

type HandlerType struct {
	handler      MsgHandler
	minConStatus ConnStatus
}

var CommandHandlers = map[string]HandlerType{
	"PASS":    {PassHandler, CONN_ESTABLISHED},
	"NICK":    {NickHandler, CONN_PASS_OK},
	"USER":    {UserHandler, CONN_NICK_OK},
	"PING":    {PingHandler, CONN_CONNECTED},
	"PONG":    {PongHandler, CONN_CONNECTED},
	"JOIN":    {JoinHandler, CONN_CONNECTED},
	"PRIVMSG": {PrivmsgHandler, CONN_CONNECTED},
	"WHO":     {WhoHandler, CONN_CONNECTED},
	"PART":    {PartHandler, CONN_CONNECTED},
	"QUIT":    {QuitHandler, CONN_CONNECTED},
	"TOPIC":   {TopicHandler, CONN_CONNECTED},
}

func PassHandler(u *User, prefix string, args string) {
	if u.status > CONN_ESTABLISHED {
		Replay(u.out, server.Hostname,
			"ERR_ALREADYREGISTRED", u.nickname)
	}

	if len(args) == 0 {
		Replay(u.out, server.Hostname,
			"ERR_NEEDMOREPARAMS", u.nickname)
	}

	if server.Password != args {
		Replay(u.out, server.Hostname,
			"ERR_PASSWDMISMATCH", u.nickname)
	}

	//pass es correcto se ve
	u.status = CONN_PASS_OK
}

func NickHandler(u *User, prefix string, args string) {
	if len(args) == 0 {
		Replay(u.out, server.Hostname,
			"ERR_NONICKNAMEGIVEN", u.nickname)
		return
	}

	nickPattern := "^[a-zA-Z\\[\\]_^`{}|][a-zA-Z0-9\\[\\]_^`{}|-]{0,15}$"
	matched, _ := regexp.MatchString(nickPattern, args)
	if !matched {
		Replay(u.out, server.Hostname,
			"ERR_ERRONEUSNICKNAME", u.nickname, args)
		return
	}

	if server.users.FindByNick(args) != nil {
		Replay(u.out, server.Hostname,
			"ERR_NICKNAMEINUSE", u.nickname, args)
		return
	}

	//Si llegamos hasta aca, el nickname es valido
	if len(u.nickname) == 0 {
		u.out <- fmt.Sprintf("NOTICE * :Welcome %s", args)
	} else {
		u.out <- fmt.Sprintf(":%s!%s@%s NICK %s", u.nickname,
			u.username, u.hostname, args)
	}

	//se lo mandamos a los canales del usuario
	u.channels.RLock()
	for _, c := range u.channels.s {
		c.out <- Msg{u,
			fmt.Sprintf(":%s!%s@%s NICK %s", u.nickname,
				u.username, u.hostname, args),
		}
	}
	u.channels.RUnlock()

	u.nickname = args
	u.status = CONN_NICK_OK
}

func UserHandler(u *User, prefix string, args string) {
	if u.status == CONN_CONNECTED {
		Replay(u.out, server.Hostname,
			"ERR_ALREADYREGISTRED", u.nickname)
		return
	}
	argv := strings.SplitN(args, " ", 4)
	if len(argv) < 4 {
		Replay(u.out, server.Hostname,
			"ERR_NEEDMOREPARAMS", u.nickname, "USER")
		return
	}

	u.username = argv[0]
	u.realname = strings.Trim(argv[3], " :")

	addr := strings.SplitN(u.conn.RemoteAddr().String(), ":", 2)[0]
	host, err := net.LookupAddr(addr)
	if err != nil {
		u.hostname = addr
	} else {
		u.hostname = host[0]
	}

	Replay(u.out, server.Hostname, "RPL_WELCOME",
		u.nickname, u.nickname, u.username, u.hostname)
	Replay(u.out, server.Hostname, "RPL_YOURHOST",
		u.nickname, server.Name, server.Version)
	Replay(u.out, server.Hostname, "RPL_CREATED", u.nickname, server.Created)
	Replay(u.out, server.Hostname, "RPL_MYINFO",
		u.nickname, server.Hostname, server.Version, "*", "*")

	u.status = CONN_CONNECTED
}

func PingHandler(u *User, prefix string, args string) {
	u.out <- fmt.Sprintf("PONG %s", args)
}

func JoinHandler(u *User, prefix string, args string) {
	argv := strings.Split(args, " ")
	if len(argv) == 0 {
		return
	}

	cName := argv[0]

	chanPattern := "^[\\[&#+'][^ \x07,:]{0,49}$"
	matched, _ := regexp.MatchString(chanPattern, argv[0])
	if !matched {
		Replay(u.out, server.Hostname,
			"ERR_ILLEGALCHANNAME", u.nickname, argv[0])
		return
	}

	c, ok := u.channels.Get(cName)
	if ok {
		return //ya esta en el canal
	}

	c, ok = server.channels.Get(cName)
	if !ok {
		c = &Channel{
			name:  cName,
			topic: "",
			out:   make(chan Msg, 100),
		}
		c.users.Init()
		server.channels.Set(cName, c)
		go sendtoChannel(c)
	}
	c.users.Add(u)
	u.channels.Set(cName, c)

	//ahora la respuesta
	c.out <- Msg{u,
		fmt.Sprintf(":%s!%s@%s JOIN %s", u.nickname, u.username,
			u.hostname, c.name),
	}
	u.out <- fmt.Sprintf(":%s!%s@%s JOIN %s", u.nickname, u.username,
		u.hostname, c.name)

	//motd
	if len(c.topic) != 0 {
		Replay(u.out, server.Hostname, "RPL_TOPIC", u.nickname, c.name, c.topic)
	}

	//usuarios conectados
	SendUserList(u, server.Hostname, c)
}

func userMessage(u *User, nick, msg string) {
	target := server.users.FindByNick(nick)
	if target != nil {
		//solo se le puede mandar mensajes si completo el registro
		if target.status == CONN_CONNECTED {
			target.out <- fmt.Sprintf(":%s!%s@%s PRIVMSG %s :%s",
				u.nickname, u.username, u.hostname,
				nick, msg)
		}
	}
}

func PrivmsgHandler(u *User, prefix string, args string) {
	argv := strings.SplitN(args, " ", 2)
	if len(argv) != 2 {
		return
	}

	msg := strings.Trim(argv[1], ": ")
	name := argv[0]
	c, ok := server.channels.Get(name)
	if !ok {
		// it's not a channel, could be a user
		userMessage(u, name, msg)
		return
	}

	c.out <- Msg{u, fmt.Sprintf(":%s!%s@%s PRIVMSG %s :%s", u.nickname, u.username, u.hostname, c.name, msg)}
}

func PongHandler(user *User, prefix string, args string) {
	return
}

func WhoHandler(u *User, prefix string, args string) {
	//TODO: implement mask
	argv := strings.Split(args, " ")
	if len(argv) == 0 {
		return
	}
	mask := argv[0]

	u.channels.RLock()
	for _, c := range u.channels.s {
		c.users.RLock()
		for _, v := range c.users.s {
			Replay(u.out, server.Hostname, "RPL_WHOREPLY",
				v.nickname, c.name, v.username, v.hostname,
				server.Hostname, v.nickname, "H", "0", v.realname)
		}
		c.users.RUnlock()
	}
	u.channels.RUnlock()
	Replay(u.out, server.Hostname, "RPL_ENDOFWHO", u.nickname, mask)
}

func PartHandler(u *User, prefix string, args string) {
	log.Println(args)
	argv := strings.Split(args, " :")
	partMsg := ""
	if len(argv) == 0 {
		Replay(u.out, server.Hostname, "ERR_NEEDMOREPARAMS",
			u.nickname, "PART")
		return
	} else if len(argv) > 1 {
		partMsg = ":" + argv[1]
	}

	channelsStr := strings.Split(strings.Trim(argv[0], " "), ",")
	for _, str := range channelsStr {
		//busco el canal
		c, ok := server.channels.Get(str)
		if !ok {
			Replay(u.out, server.Hostname, "ERR_NOSUCHCHANNEL",
				u.nickname, str)
			break
		}

		//busco el canal en el usuario
		_, ok = u.channels.Get(str)
		if !ok {
			Replay(u.out, server.Hostname, "ERR_NOTONCHANNEL",
				u.nickname, str)
			break
		}

		//elimino el usuario del canal y le mando un mensaje
		//a todos en el canal
		c.users.Remove(u)
		c.out <- Msg{
			u,
			fmt.Sprintf(":%s!%s@%s PART %s %s", u.nickname,
				u.username, u.hostname, c.name, partMsg),
		}

		u.out <- fmt.Sprintf(":%s!%s@%s PART %s %s", u.nickname,
			u.username, u.hostname, c.name, partMsg)

		//elimino el canal del usuario
		u.channels.Remove(c)

		if len(c.users.s) == 0 {
			server.channels.Remove(c)
		}
	}
}

func QuitHandler(u *User, prefix string, args string) {
	argv := strings.Split(args, " :")
	var msg string
	if len(argv) > 0 {
		msg = argv[0]
	}

	//send to each user's channel the QUIT msg
	u.channels.RLock()
	for _, c := range u.channels.s {
		c.out <- Msg{u,
			fmt.Sprintf(":%s!%s@%s QUIT %s :%s", u.nickname,
				u.username, u.hostname, u.nickname, msg),
		}
		u.out <- fmt.Sprintf(":%s!%s@%s ERROR :Closing Link: %s (Quit: %s)",
			u.nickname, u.username, u.hostname, u.hostname, msg)
	}
	u.channels.RUnlock()
	return
}

func TopicHandler(u *User, prefix string, args string) {
	argv := strings.Split(args, " ")

	if len(argv) == 0 {
		Replay(u.out, server.Hostname, "ERR_NEEDMOREPARAMS", u.nickname, "TOPIC")
		return
	}

	str := strings.Trim(argv[0], " ")
	c, ok := server.channels.Get(str)
	if !ok {
		Replay(u.out, server.Hostname, "ERR_NOSUCHCHANNEL", u.nickname, str)
		return
	}

	_, ok = u.channels.Get(str)
	if !ok {
		Replay(u.out, server.Hostname, "ERR_NOTONCHANNEL", u.nickname, str)
		return
	}

	if len(argv) == 1 {
		if len(c.topic) != 0 {
			Replay(u.out, server.Hostname, "RPL_TOPIC", c.topicuser, c.name, c.topic)
		} else {
			Replay(u.out, server.Hostname, "RPL_NOTOPIC", c.name)
		}

		return
	} else if len(argv) == 2 && argv[1] == ":" {
		c.topic = ""
		c.topicuser = u.nickname
		Replay(u.out, server.Hostname, "RPL_NOTOPIC", c.name)
	} else {
		topic := strings.Split(args, ":")
		c.topic = topic[1]
		c.topicuser = u.nickname
		Replay(u.out, server.Hostname, "RPL_TOPIC", c.name, c.topic)
	}

	c.out <- Msg{
		u,
		fmt.Sprintf(":%s!%s@%s TOPIC %s :%s", u.nickname,
			u.username, u.hostname, c.name, c.topic),
	}

	return
}
