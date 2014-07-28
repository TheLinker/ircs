package main

import (
	"bufio"
	"container/list"
	"log"
	"net"
	"strings"
)

type Msg struct {
	nickname string
	msg      string
}

type Channel struct {
	name  string
	users *list.List
	out   chan Msg
}

var Channels map[string]*Channel

type User struct {
	conn        net.Conn
	nickname    string
	username    string
	realname    string
	hostname    string
	can_connect bool
	out         chan string
}

var Users *list.List

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

	log.Printf("%q %q %q\n", prefix, command, argv)

	handler, ok := CommandHandlers[command]
	if ok {
		handler(u, prefix, argv)
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

		message := strings.TrimRight(string(line), "\r\n")
		log.Println("<- " + message)
		parseCommand(message, u)
	}
}

func sendtoChannel(c *Channel) {
	for msg := range c.out {
		for u := c.users.Front(); u != nil; u = u.Next() {
			if msg.nickname != u.Value.(*User).nickname {
				u.Value.(*User).out <- msg.msg
			}
		}
	}
}

func sendtoClient(u *User) {
	for message := range u.out {
		message += "\r\n"
		log.Println("-> " + message)
		_, err := u.conn.Write([]byte(message))
		if err != nil {
			log.Println(err)
			break
		}
	}
}

func main() {
	Users = list.New()
	Channels = make(map[string]*Channel)

	// Listen on TCP port 2000 on all interfaces.
	l, err := net.Listen("tcp", ":2000")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	for {
		// Wait for a connection.
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		// Handle the connection in a new goroutine.
		// The loop then returns to accepting, so that
		// multiple connections may be served concurrently.
		tmp := new(User)
		tmp.conn = conn
		tmp.out = make(chan string)
		Users.PushBack(tmp)
		go sendtoClient(tmp)
		go listenClient(tmp)
	}
}
