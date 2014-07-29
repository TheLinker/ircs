package main

import (
	"bufio"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type Msg struct {
	nickname string
	msg      string
}

type Channel struct {
	name   string
	usersM sync.Mutex
	users  []*User
	out    chan Msg
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
	in          chan string
}

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

		msg := strings.TrimRight(string(line), "\r\n")
		log.Println("<- " + msg)
		u.in <- msg
	}
}

func removeUser(u *User) {
	for _, c := range Channels {
		c.usersM.Lock()
		for i := range c.users {
			if c.users[i] == u {
				c.users[i] = nil
			}
		}
		c.usersM.Unlock()
	}

	close(u.out)
	close(u.in)
	err := u.conn.Close()
	if err != nil {
		log.Println(err)
	}
}

func processMessages(u *User) {
	timer := time.NewTimer(time.Second * 30)
Out:
	for {
		select {
		case msg := <-u.in:
			parseCommand(msg, u)
		case <-timer.C:
			removeUser(u)
			break Out
		}
		timer.Reset(time.Second * 30)
	}
}

func sendtoChannel(c *Channel) {
	for msg := range c.out {
		c.usersM.Lock()
		for _, u := range c.users {
			if u != nil && msg.nickname != u.nickname {
				u.out <- msg.msg
			}
		}
		c.usersM.Unlock()
	}
}

func sendtoClient(u *User) {
	pinger := time.NewTicker(time.Second * 10)
	for {
		var msg string
		select {
		case msg = <-u.out:
		case <-pinger.C:
			msg = "PING :" + u.nickname
		}
		log.Println("-> " + msg)
		msg += "\r\n"
		_, err := u.conn.Write([]byte(msg))
		if err != nil {
			log.Println(err)
			break
		}
	}
	pinger.Stop()
}

func main() {
	Channels = make(map[string]*Channel)

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
		user.conn = conn
		user.out = make(chan string)
		user.in = make(chan string)
		go sendtoClient(user)
		go listenClient(user)
		go processMessages(user)
	}
}
