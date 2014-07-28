package main

import (
	"container/list"
	"fmt"
	"log"
	"net"
	"strings"
)

var Mess_handlers = map[string]func(c *User, prefix string, args string){
	"NICK":    Nick_handler,
	"USER":    User_handler,
	"PING":    Ping_handler,
	"JOIN":    Join_handler,
	"PRIVMSG": Privmsg_handler,
}

func Nick_handler(c *User, prefix string, args string) {
	//por ahora lo aceptamos sin mas comprobacion
	c.nickname = args
	log.Print(c.nickname)
	c.out <- fmt.Sprintf("NOTICE * :Welcome %s", c.nickname)
}

func User_handler(c *User, prefix string, args string) {
	argv := strings.SplitN(args, " ", 4)
	if len(argv) != 4 {
		return
	}

	var hostname string

	c.username = argv[0]
	c.realname = strings.Trim(argv[3], " :")

	log.Print(c.conn.RemoteAddr().String())
	host, err := net.LookupAddr(c.conn.RemoteAddr().String())
	if err != nil {
		log.Print(err)
		hostname = "localhost"
	} else {
		hostname = host[0]
	}
	c.hostname = hostname

	c.out <- fmt.Sprintf("001 %s :Welcome to the Internet Relay Network %s!%s@%s", c.nickname, c.nickname, c.username, c.hostname)
	c.out <- fmt.Sprintf("002 %s :Your host is MyIRCServer, running version 0.0.0.0.0.1", c.nickname)
	c.out <- fmt.Sprintf("003 %s :This server was created 2014/07/26", c.nickname)
	c.out <- fmt.Sprintf("004 %s :localhost 0.0.0.0.0.1 * *", c.nickname)
}

func Ping_handler(c *User, prefix string, args string) {
	c.out <- fmt.Sprintf("PONG %s", args)
}

func Join_handler(c *User, prefix string, args string) {
	argv := strings.Split(args, " ")
	if len(argv) == 0 {
		return
	}

	canal := argv[0]
	channel, ok := Channels[canal]

	if !ok {
		tmp := &Channel{name: canal, out: make(chan Msg), users: list.New()}
		Channels[canal] = tmp
		channel = tmp

		go sendtoChannel(channel)
	}

	channel.users.PushBack(c)

	//ahora la respuesta
	c.out <- fmt.Sprintf("JOIN %s", channel.name)

	//motd
	c.out <- fmt.Sprintf("332 %s %s :%s", c.nickname, channel.name, "Hola")

	//usuarios conectados
	var users string

	for e := channel.users.Front(); e != nil; e = e.Next() {
		users += " " + e.Value.(*User).nickname
	}

	c.out <- fmt.Sprintf("353 %s = %s :%s", c.nickname, channel.name, strings.TrimLeft(users, " "))
	c.out <- fmt.Sprintf("366 %s %s :End of /NAMES list", c.nickname, channel.name)

	channel.out <- Msg{c.nickname, fmt.Sprintf(":%s!%s@%s JOIN %s", c.nickname, c.username, c.hostname, channel.name)}
}

func Privmsg_handler(c *User, prefix string, args string) {
	argv := strings.SplitN(args, " ", 2)
	if len(argv) != 2 {
		return
	}

	canal := argv[0]
	channel, ok := Channels[canal]
	if !ok {
		return
	}

	msg := strings.Trim(argv[1], ": ")

	channel.out <- Msg{c.nickname, fmt.Sprintf(":%s!%s@%s PRIVMSG %s :%s", c.nickname, c.username, c.hostname, channel.name, msg)}
}
