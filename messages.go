package main

import (
	"container/list"
	"fmt"
	"log"
	"net"
	"strings"
)

var CommandHandlers = map[string]func(c *User, prefix string, args string){
	"NICK":    NickHandler,
	"USER":    UserHandler,
	"PING":    PingHandler,
	"JOIN":    JoinHandler,
	"PRIVMSG": PrivmsgHandler,
}

func NickHandler(u *User, prefix string, args string) {
	//por ahora lo aceptamos sin mas comprobacion
	u.nickname = args
	log.Println(u.nickname)
	u.out <- fmt.Sprintf("NOTICE * :Welcome %s", u.nickname)
}

func UserHandler(u *User, prefix string, args string) {
	argv := strings.SplitN(args, " ", 4)
	if len(argv) != 4 {
		return
	}

	u.username = argv[0]
	u.realname = strings.Trim(argv[3], " :")

	log.Println(u.conn.RemoteAddr().String())
	host, err := net.LookupAddr(u.conn.RemoteAddr().String())
	if err != nil {
		log.Println(err)
		u.hostname = "localhost"
	} else {
		u.hostname = host[0]
	}

    Replay(u.out, "bayerl.com.ar", "RPL_WELCOME", u.nickname, u.nickname, u.username, u.hostname)
    Replay(u.out, "bayerl.com.ar", "RPL_YOURHOST", u.nickname, u.nickname, "MyIRCServer")
    Replay(u.out, "bayerl.com.ar", "RPL_CREATED", u.nickname, "2014/07/26")
    Replay(u.out, "bayerl.com.ar", "RPL_MYINFO", u.nickname ,"bayerl.com.ar", "0.0.0.0.0.0.1", "*", "*")
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
	channel, ok := Channels[channelName]
	if !ok {
		tmp := &Channel{name: channelName, out: make(chan Msg), users: list.New()}
		Channels[channelName] = tmp
		channel = tmp
		go sendtoChannel(channel)
	}
	channel.usersM.Lock()
	channel.users.PushBack(u)
	channel.usersM.Unlock()

	//ahora la respuesta
    u.out <- fmt.Sprintf(":%s JOIN %s", u.nickname, channel.name)

	//motd
    Replay(u.out, "bayerl.com.ar", "RPL_TOPIC", u.nickname, channel.name, "Hola")

	//usuarios conectados
    SendUserList(u, "bayerl.com.ar", channel)

	channel.out <- Msg{u.nickname, fmt.Sprintf(":%s!%s@%s JOIN %s", u.nickname, u.username, u.hostname, channel.name)}
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
