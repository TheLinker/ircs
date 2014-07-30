package main

import (
	"container/list"
	"fmt"
	"log"
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
		Replay(u.out, "bayerl.com.ar", "ERR_NONICKNAMEGIVEN", u.nickname)
		return
	}

	nickPattern := "^[a-zA-Z\\[\\]_^`{}|][a-zA-Z0-9\\[\\]_^`{}|-]{0,8}$"
	matched, _ := regexp.MatchString(nickPattern, args)
	if !matched {
		Replay(u.out, "bayerl.com.ar", "ERR_ERRONEUSNICKNAME", u.nickname, args)
		return
	}

	if FindUser(args) != nil {
		Replay(u.out, "bayerl.com.ar", "ERR_NICKNAMEINUSE", u.nickname, args)
		return
	}

	//Si llegamos hasta aca, el nickname es valido
	if len(u.nickname) == 0 {
		u.out <- fmt.Sprintf("NOTICE * :Welcome %s", args)
	} else {
		u.out <- fmt.Sprintf(":%s!%s@%s NICK %s", u.nickname, u.username, u.hostname, args)
	}

	//se lo mandamos a los canales del usuario
	u.channelsM.RLock()
	for _, c := range u.channels {
		c.out <- Msg{u.nickname, fmt.Sprintf(":%s!%s@%s NICK %s", u.nickname, u.username, u.hostname, args)}
	}
	u.channelsM.RUnlock()

	u.nickname = args
}

func UserHandler(u *User, prefix string, args string) {
	argv := strings.SplitN(args, " ", 4)
	if len(argv) < 4 {
		Replay(u.out, "bayerl.com.ar", "ERR_NEEDMOREPARAMS", u.nickname, "USER")
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

	Replay(u.out, "bayerl.com.ar", "RPL_WELCOME", u.nickname, u.nickname, u.username, u.hostname)
	Replay(u.out, "bayerl.com.ar", "RPL_YOURHOST", u.nickname, "MyIRCServer", "0.0.0.0.0.0.1")
	Replay(u.out, "bayerl.com.ar", "RPL_CREATED", u.nickname, "2014/07/26")
	Replay(u.out, "bayerl.com.ar", "RPL_MYINFO", u.nickname, "bayerl.com.ar", "0.0.0.0.0.0.1", "*", "*")
}

func PingHandler(c *User, prefix string, args string) {
	c.out <- fmt.Sprintf("PONG %s", args)
}

func JoinHandler(u *User, prefix string, args string) {
	argv := strings.Split(args, " ")
	if len(argv) == 0 {
		return
	}

	channelName := argv[0]

	channel, ok := u.channels[channelName]
	if ok {
		//ya esta en el canal
		return
	}

	channel, ok = Channels[channelName]
	if !ok {
		tmp := &Channel{name: channelName, out: make(chan Msg), users: list.New()}
		Channels[channelName] = tmp
		channel = tmp
		go sendtoChannel(channel)
	}
	channel.usersM.Lock()
	channel.users.PushBack(u)
	channel.usersM.Unlock()

	u.channelsM.Lock()
	u.channels[channel.name] = channel
	u.channelsM.Unlock()

	//ahora la respuesta
	//u.out <- fmt.Sprintf(":%s JOIN %s", u.nickname, channel.name)
	channel.out <- Msg{u.nickname, fmt.Sprintf(":%s!%s@%s JOIN %s", u.nickname, u.username, u.hostname, channel.name)}
	u.out <- fmt.Sprintf(":%s!%s@%s JOIN %s", u.nickname, u.username, u.hostname, channel.name)

	//motd
	Replay(u.out, "bayerl.com.ar", "RPL_TOPIC", u.nickname, channel.name, "Hola")

	//usuarios conectados
	SendUserList(u, "bayerl.com.ar", channel)

}

func PrivmsgHandler(c *User, prefix string, args string) {
	argv := strings.SplitN(args, " ", 2)
	if len(argv) != 2 {
		return
	}

	channelName := argv[0]
	channel, ok := Channels[channelName]
	if !ok {
		return
	}

	msg := strings.Trim(argv[1], ": ")

	channel.out <- Msg{c.nickname, fmt.Sprintf(":%s!%s@%s PRIVMSG %s :%s", c.nickname, c.username, c.hostname, channel.name, msg)}
}

func PongHandler(user *User, prefix string, args string) {
	return
}

func WhoHandler(user *User, prefix string, args string) {
	argv := strings.Split(args, " ")
	if len(argv) == 0 {
		return
	}

	//por ahora asumo que me esta pasando un canal
	channelName := argv[0]
	channel, ok := Channels[channelName]
	if !ok {
		return
	}

	channel.usersM.Lock()
	for u := channel.users.Front(); u != nil; u = u.Next() {
		tmp := u.Value.(*User)
		Replay(user.out, "bayerl.com.ar", "RPL_WHOREPLY", user.nickname, channel.name, tmp.username, tmp.hostname, "bayerl.com.ar", tmp.nickname, "H", "0", tmp.realname)
	}
	channel.usersM.Unlock()

	Replay(user.out, "bayerl.com.ar", "RPL_ENDOFWHO", user.nickname, channel.name)

}

func PartHandler(user *User, prefix string, args string) {
	argv := strings.Split(args, " :")

	if len(argv) != 2 {
		Replay(user.out, "bayerl.com.ar", "ERR_NEEDMOREPARAMS", user.nickname, "PART")
		return
	}

	channelsStr := strings.Split(strings.Trim(argv[0], " "), ",")

	for _, i := range channelsStr {
		//busco el canal
		channel, ok := Channels[i]
		if !ok {
			Replay(user.out, "bayerl.com.ar", "ERR_NOSUCHCHANNEL", user.nickname, i)
			break
		}

		//busco el canal en el usuario
		user.channelsM.RLock()
		_, ok = user.channels[i]
		user.channelsM.RUnlock()
		if !ok {
			Replay(user.out, "bayerl.com.ar", "ERR_NOTONCHANNEL", user.nickname, i)
			break
		}

		//elimino el usuario del canal y le mando un mensaje a todos en el canal
		RemoveUserFromChannel(channel, user)
		channel.out <- Msg{user.nickname,
			fmt.Sprintf(":%s!%s@%s PART %s :%s", user.nickname, user.username, user.hostname, channel.name, argv[1])}
		user.out <- fmt.Sprintf(":%s!%s@%s PART %s :%s", user.nickname, user.username, user.hostname, channel.name, argv[1])

		//elimino el canal del usuario
		user.channelsM.Lock()
		delete(user.channels, i)
		user.channelsM.Unlock()
	}

}

func QuitHandler(user *User, prefix string, args string) {
	argv := strings.Split(args, " :")
	var msg string
	if len(argv) == 0 {
		msg = ""
	} else {
		msg = argv[0]
	}

	//hay que enviar a cada canal el mensaje QUIT
	for _, i := range user.channels {
		i.out <- Msg{user.nickname,
			fmt.Sprintf(":%s!%s@%s QUIT %s :%s", user.nickname, user.username, user.hostname, msg)}
		user.out <- fmt.Sprintf(":%s!%s@%s ERROR :Closing Link: %s (Quit: %s)", user.nickname, user.username, user.hostname, user.hostname, msg)

		RemoveUserFromChannel(i, user)
	}

	//elimino el usuario de la lista global
	UsersLock.Lock()
	for u := Users.Front(); u != nil; u = u.Next() {
		if u.Value.(*User).nickname == user.nickname {
			Users.Remove(u)
			close(u.Value.(*User).out)
			close(u.Value.(*User).in)
			err := u.Value.(*User).conn.Close()
			if err != nil {
				log.Println(err)
			}

			break
		}
	}
	UsersLock.Unlock()

	return

}
