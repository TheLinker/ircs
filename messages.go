package main

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

var CommandHandlers = map[string]func(c *User, prefix string, args string){
	"NICK":    NickHandler,
	"USER":    UserHandler,
	"PING":    PingHandler,
	"PONG":    PongHandler,
	"JOIN":    JoinHandler,
	"PRIVMSG": PrivmsgHandler,
	"WHO":     WhoHandler,
	"PART":    PartHandler,
	"QUIT":    QuitHandler,
}

func NickHandler(u *User, prefix string, args string) {
	if len(args) == 0 {
		Replay(u.out, "bayerl.com.ar",
			"ERR_NONICKNAMEGIVEN",u.nickname)
		return
	}

	nickPattern := "^[a-zA-Z\\[\\]_^`{}|][a-zA-Z0-9\\[\\]_^`{}|-]{0,8}$"
	matched, _ := regexp.MatchString(nickPattern, args)
	if !matched {
		Replay(u.out, "bayerl.com.ar",
			"ERR_ERRONEUSNICKNAME", u.nickname, args)
		return
	}

	if Users.FindByNick(args) != nil {
		Replay(u.out, "bayerl.com.ar",
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
		c.out <- Msg{
			u,
			fmt.Sprintf(":%s!%s@%s NICK %s", u.nickname,
				u.username, u.hostname, args),
		}
	}
	u.channels.RUnlock()

	u.nickname = args
}

func UserHandler(u *User, prefix string, args string) {
	argv := strings.SplitN(args, " ", 4)
	if len(argv) < 4 {
		Replay(u.out, "bayerl.com.ar",
			"ERR_NEEDMOREPARAMS", u.nickname, "USER")
		return
	}

	u.username = argv[0]
	u.realname = strings.Trim(argv[3], " :")

	host, err := net.LookupAddr(u.conn.RemoteAddr().String())
	if err != nil {
		u.hostname = u.conn.RemoteAddr().String()
	} else {
		u.hostname = host[0]
	}

	Replay(u.out, "bayerl.com.ar", "RPL_WELCOME",
		u.nickname, u.nickname, u.username, u.hostname)
	Replay(u.out, "bayerl.com.ar", "RPL_YOURHOST",
		u.nickname, "MyIRCServer", "0.0.0.0.0.0.1")
	Replay(u.out, "bayerl.com.ar", "RPL_CREATED", u.nickname, "2014/07/26")
	Replay(u.out, "bayerl.com.ar", "RPL_MYINFO",
		u.nickname, "bayerl.com.ar", "0.0.0.0.0.0.1", "*", "*")
}

func PingHandler(c *User, prefix string, args string) {
	c.out <- fmt.Sprintf("PONG %s", args)
}

func JoinHandler(u *User, prefix string, args string) {
	argv := strings.Split(args, " ")
	if len(argv) == 0 {
		return
	}

	cName := argv[0]

	c, ok := u.channels.s[cName]
	if ok {
		return //ya esta en el canal
	}

	c, ok = Channels.s[cName]
	if !ok {
		c = &Channel{
			name: cName,
			out: make(chan Msg),
		}
		c.users.Init()
		Channels.Add(cName, c)
		go sendtoChannel(c)
	}
	c.users.Add(u)
	u.channels.Add(cName, c)

	//ahora la respuesta
	//u.out <- fmt.Sprintf(":%s JOIN %s", u.nickname, channel.name)
	c.out <- Msg{
		u,
		fmt.Sprintf(":%s!%s@%s JOIN %s", u.nickname, u.username,
			u.hostname, c.name),
	}
	u.out <- fmt.Sprintf(":%s!%s@%s JOIN %s", u.nickname, u.username,
		u.hostname, c.name)

	//motd
	Replay(u.out, "bayerl.com.ar", "RPL_TOPIC", u.nickname, c.name, "Hola")

	//usuarios conectados
	SendUserList(u, "bayerl.com.ar", c)

}

func PrivmsgHandler(u *User, prefix string, args string) {
	argv := strings.SplitN(args, " ", 2)
	if len(argv) != 2 {
		return
	}

	cName := argv[0]
	c, ok := Channels.s[cName]
	if !ok {
		return
	}

	msg := strings.Trim(argv[1], ": ")

	c.out <- Msg{
		u,
		fmt.Sprintf(":%s!%s@%s PRIVMSG %s :%s", u.nickname,
		u.username, u.hostname, c.name, msg),
	}
}

func PongHandler(user *User, prefix string, args string) {
	return
}

func WhoHandler(u *User, prefix string, args string) {
	argv := strings.Split(args, " ")
	if len(argv) == 0 {
		return
	}

	//por ahora asumo que me esta pasando un canal
	cName := argv[0]
	c, ok := Channels.s[cName]
	if !ok {
		return
	}

	c.users.Lock()
	for _,u := range c.users.s {
		Replay(u.out, "bayerl.com.ar", "RPL_WHOREPLY", u.nickname,
			c.name, u.username, u.hostname, "bayerl.com.ar",
			u.nickname, "H", "0", u.realname)
	}
	c.users.Unlock()

	Replay(u.out, "bayerl.com.ar", "RPL_ENDOFWHO", u.nickname, c.name)

}

func PartHandler(u *User, prefix string, args string) {
	argv := strings.Split(args, " :")
	if len(argv) != 2 {
		Replay(u.out, "bayerl.com.ar", "ERR_NEEDMOREPARAMS",
			u.nickname, "PART")
		return
	}

	channelsStr := strings.Split(strings.Trim(argv[0], " "), ",")
	for _, str := range channelsStr {
		//busco el canal
		c, ok := Channels.s[str]
		if !ok {
			Replay(u.out, "bayerl.com.ar", "ERR_NOSUCHCHANNEL",
				u.nickname, str)
			break
		}

		//busco el canal en el usuario
		u.channels.RLock()
		_, ok = u.channels.s[str]
		u.channels.RUnlock()
		if !ok {
			Replay(u.out, "bayerl.com.ar", "ERR_NOTONCHANNEL",
				u.nickname, str)
			break
		}

		//elimino el usuario del canal y le mando un mensaje
		//a todos en el canal
		c.users.Remove(u)
		c.out <- Msg{
			u,
			fmt.Sprintf(":%s!%s@%s PART %s :%s", u.nickname,
				u.username, u.hostname, c.name, argv[1]),
		}
		u.out <- fmt.Sprintf(":%s!%s@%s PART %s :%s", u.nickname,
			u.username, u.hostname, c.name, argv[1])

		//elimino el canal del usuario
		u.channels.Lock()
		delete(u.channels.s, str)
		u.channels.Unlock()
	}

}

func QuitHandler(u *User, prefix string, args string) {
	argv := strings.Split(args, " :")
	var msg string
	if len(argv) > 0 {
		msg = argv[0]
	}

	//send to each user's channel the QUIT msg and remove user from channel
	u.channels.RLock()
	for _, c := range u.channels.s {
		c.out <- Msg{
			u,
			fmt.Sprintf(":%s!%s@%s QUIT %s :%s", u.nickname,
				u.username, u.hostname, msg),
		}
		u.out <- fmt.Sprintf(":%s!%s@%s ERROR :Closing Link: %s (Quit: %s)",
			u.nickname, u.username, u.hostname, u.hostname, msg)

		// removes user from channel
		c.users.Remove(u)
	}
	u.channels.RUnlock()

	//removes user from global Users
	Users.Remove(u)

	return
}
