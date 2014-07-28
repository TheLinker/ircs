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
	log.Print(u.nickname)
	u.out <- fmt.Sprintf("NOTICE * :Welcome %s", u.nickname)
}

func UserHandler(u *User, prefix string, args string) {
	argv := strings.SplitN(args, " ", 4)
	if len(argv) != 4 {
		return
	}

	u.username = argv[0]
	u.realname = strings.Trim(argv[3], " :")

	log.Print(u.conn.RemoteAddr().String())
	host, err := net.LookupAddr(u.conn.RemoteAddr().String())
	if err != nil {
		log.Print(err)
		u.hostname = "localhost"
	} else {
		u.hostname = host[0]
	}

	u.out <- fmt.Sprintf("001 %s :Welcome to the Internet Relay Network %s!%s@%s", u.nickname, u.nickname, u.username, u.hostname)
	u.out <- fmt.Sprintf("002 %s :Your host is MyIRCServer, running version 0.0.0.0.0.1", u.nickname)
	u.out <- fmt.Sprintf("003 %s :This server was created 2014/07/26", u.nickname)
	u.out <- fmt.Sprintf("004 %s :localhost 0.0.0.0.0.1 * *", u.nickname)
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
	channel.users.PushBack(u)

	//ahora la respuesta
	u.out <- fmt.Sprintf("JOIN %s", channel.name)

	//motd
	u.out <- fmt.Sprintf("332 %s %s :%s", u.nickname, channel.name, "Hola")

	//usuarios conectados
	var nicks string
	for u := channel.users.Front(); u != nil; u = u.Next() {
		nicks += " " + u.Value.(*User).nickname
	}

	u.out <- fmt.Sprintf("353 %s = %s :%s", u.nickname, channel.name, strings.TrimLeft(nicks, " "))
	u.out <- fmt.Sprintf("366 %s %s :End of /NAMES list", u.nickname, channel.name)

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
