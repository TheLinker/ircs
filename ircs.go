package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

var Users UsersSet
var Channels ChannelsSet

func parseCommand(message string, u *User) {
	var prefix, command, argv string

	if len(message) == 0 {
		return
	}
	if message[0] == ':' {
		//estan mandando prefix
		tmp := strings.SplitN(message, " ", 2)
		prefix = strings.TrimLeft(tmp[0], ":")
		message = tmp[1]
	}

	//obtenemos el comando
	tmp := strings.SplitN(message, " ", 2)
	command = tmp[0]
	if len(tmp) > 1 {
		argv = strings.Trim(tmp[1], " ")
	}

	handler, ok := CommandHandlers[command]
	if ok {
		if u.status >= handler.minConStatus {
			handler.handler(u, prefix, argv)
		} else {
			Replay(u.out, server.hostname,
				"ERR_NOTREGISTERED", u.nickname)
		}
	} else {
		log.Println("Command not found: " + command)
	}
}

func listenClient(u *User) {
	r := bufio.NewReader(u.conn)
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			log.Println(err)
			break
		}

		msg := strings.TrimRight(string(line), "\r\n")
		if !strings.Contains(msg, "PING") && !strings.Contains(msg, "PONG") {
			log.Println(u.nickname + "\t-> " + msg)
		}
		u.conn.SetDeadline(time.Now().Add(time.Second * 30))
		parseCommand(msg, u)
	}
	log.Println("Removing user: ", u.nickname)
	removeUser(u)
}

func removeUser(u *User) {
	// removes user from global channels' set
	Channels.RLock()
	for _, c := range Channels.s {
		c.users.Remove(u)
	}
	Channels.RUnlock()

	// removes user from global users set
	Users.Remove(u)

	// send messages to user channels
	u.channels.RLock()
	for _, c := range u.channels.s {
		c.out <- Msg{
			u,
			fmt.Sprintf(":%s!%s@%s QUIT %s :%s",
				u.nickname, u.username,
				u.hostname, c.name, "Timeout"),
		}
		u.out <- fmt.Sprintf(":%s!%s@%s ERROR :Closing Link: %s (Quit: %s)",
			u.nickname, u.username, u.hostname,
			u.hostname, "Timeout")
	}
	u.channels.RUnlock()

	err := u.conn.Close()
	if err != nil {
		log.Println(err)
	}
	close(u.out)
}

func sendtoChannel(c *Channel) {
	for msg := range c.out {
		c.users.RLock()
		for _, u := range c.users.s {
			if msg.user != u {
				select {
				case u.out <- msg.msg:
				default:
				}
			}
		}
		c.users.RUnlock()
	}
}

func sendtoClient(u *User) {
	pinger := time.NewTicker(time.Second * 10)
	for {
		var msg string
		select {
		case msg = <-u.out:
		case <-pinger.C:
			if u.status == CONN_CONNECTED {
				msg = "PING :" + u.nickname
			}
		}
		if len(msg) != 0 {
			if !strings.Contains(msg, "PING") && !strings.Contains(msg, "PONG") {
				log.Println(u.nickname + "\t<- " + msg)
			}
			msg += "\r\n"
			_, err := u.conn.Write([]byte(msg))
			if err != nil {
				log.Println(err)
				break
			}
		}
	}
	pinger.Stop()
}

func main() {
	Users.Init()
	Channels.Init()

	server.created = time.Now().Format("2006/01/02 15:04:05")
	//llenar con configfile, pero bueh
	server.hostname = "bayerl.com.ar"
	server.name = "MyIRCServer"
	server.version = "0.0.0.0.0.0.0.1"

	// Listen on TCP port 2000 on all interfaces.
	l, err := net.Listen("tcp", ":2000")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		user := new(User)
		conn.SetDeadline(time.Now().Add(time.Second * 30))
		user.conn = conn
		user.out = make(chan string, 20)
		user.channels.Init()
		if len(server.password) == 0 {
			user.status = CONN_PASS_OK
		} else {
			user.status = CONN_ESTABLISHED
		}
		log.Println("Connection from: " + conn.RemoteAddr().String())

		Users.Add(user)

		go sendtoClient(user)
		go listenClient(user)
	}
}
