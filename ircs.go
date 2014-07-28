package main

import (
	"bufio"
	"container/list"
	"log"
	"net"
	"strings"
)

type Channel struct {
	name  string
	users *list.List
	out   chan string
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
	inp         chan string
}

var Users *list.List

func interpretar_comando(message string, c *User) {
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

	handler, ok := Mess_handlers[command]

	if ok {
		handler(c, prefix, argv)
	}

}

func listenClient(c *User) {
	r := bufio.NewReader(c.conn)
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			log.Print(err)
			break
		}

		message := strings.TrimRight(string(line), "\r\n")
		log.Print("<- " + message)
		interpretar_comando(message, c)
	}
}

func sendtoChannel(c *Channel) {
	for {
		//message := <-c.out
		user, message := <-c.out, <-c.out

		for e := c.users.Front(); e != nil; e = e.Next() {
			if user != e.Value.(*User).nickname {
				e.Value.(*User).out <- message
			}
		}
	}
}

func sendtoClient(c *User) {
	for {
		message := <-c.out
		message += "\r\n"
		log.Print("-> " + message)
		c.conn.Write([]byte(message))
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
		tmp.inp = make(chan string)
		tmp.out = make(chan string)
		Users.PushBack(tmp)
		go sendtoClient(tmp)
		go listenClient(tmp)
	}
}
