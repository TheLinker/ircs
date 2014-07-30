package main

import (
	"bufio"
	"container/list"
	"fmt"
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
	users  *list.List
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
	channelsM   sync.RWMutex
	channels    map[string]*Channel
}

var UsersLock sync.RWMutex
var Users *list.List

func RemoveUserFromChannel(channel *Channel, user *User) {
	channel.usersM.Lock()
	for u := channel.users.Front(); u != nil; u = u.Next() {
		if u.Value.(*User).nickname == user.nickname {
			channel.users.Remove(u)
			break
		}
	}
	channel.usersM.Unlock()
}

func FindUser(nickname string) *User {
	UsersLock.RLock()
	defer UsersLock.RUnlock()
	for c := Users.Front(); c != nil; c = c.Next() {
		if c.Value.(*User).nickname == nickname {
			return c.Value.(*User)
		}
	}
	return nil
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

	//	log.Printf("%q %q %q\n", prefix, command, argv)

	handler, ok := CommandHandlers[command]
	if ok {
		handler(u, prefix, argv)
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
		log.Println(u.nickname + "\t-> " + msg)
		u.in <- msg
	}
}

func removeUser(u *User) {
	//TODO: removes user from global list
	for _, c := range Channels {
		c.usersM.Lock()
		var found *list.Element
		for e := c.users.Front(); e != nil; e = e.Next() {
			if e.Value.(*User) == u {
				found = e
			}
		}
		if found != nil {
			for _, i := range u.channels {
				i.out <- Msg{u.nickname,
					fmt.Sprintf(":%s!%s@%s QUIT %s :%s", u.nickname, u.username, u.hostname, "Timeout")}
				u.out <- fmt.Sprintf(":%s!%s@%s ERROR :Closing Link: %s (Quit: %s)", u.nickname, u.username, u.hostname, u.hostname, "Timeout")

				c.users.Remove(found)
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
		for u := c.users.Front(); u != nil; u = u.Next() {
			if msg.nickname != u.Value.(*User).nickname {
				u.Value.(*User).out <- msg.msg
			}
		}
		c.usersM.Unlock()
	}
}

func sendtoClient(u *User) {
	pinger := time.NewTimer(time.Second * 10)
	for {
		var msg string
		select {
		case msg = <-u.out:
		case <-pinger.C:
			msg = "PING :TheLinker"
		}
		pinger.Reset(time.Second * 10)
		log.Println(u.nickname + "\t<- " + msg)
		msg += "\r\n"
		_, err := u.conn.Write([]byte(msg))
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
		tmp.in = make(chan string)
		tmp.channels = make(map[string]*Channel)

		UsersLock.Lock()
		Users.PushBack(tmp)
		UsersLock.Unlock()

		go sendtoClient(tmp)
		go listenClient(tmp)
		go processMessages(tmp)
	}
}
