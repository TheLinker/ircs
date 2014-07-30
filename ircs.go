package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type Msg struct {
	user *User
	msg      string
}

type Channel struct {
	name   string
	users  UsersSet
	out    chan Msg
}

type ChannelsSet struct {
	sync.RWMutex
	s map[string]*Channel
}

func (set *ChannelsSet) Init() {
	set.s = make(map[string]*Channel)
}

func (set *ChannelsSet) Set(k string, c *Channel) {
	set.Lock()
	set.s[k] = c
	set.Unlock()
}

func (set *ChannelsSet) Get(k string) (c *Channel, ok bool) {
	set.RLock()
	defer set.RUnlock()
	c, ok = set.s[k]
	return
}

var Channels ChannelsSet

type User struct {
	conn        net.Conn
	nickname    string
	username    string
	realname    string
	hostname    string
	can_connect bool
	out         chan string
	channels    ChannelsSet
}

type UsersSet struct {
	sync.RWMutex
	s []*User
}

func (set *UsersSet) Init() {
	set.s = make([]*User, 0)
}

func (users *UsersSet) Remove(u *User) (ret *User) {
	found := false

	users.Lock()
	for i := range users.s {
		if users.s[i] == u {
			found = true
			ret = users.s[i]
			users.s[i] = users.s[len(users.s)-1]
			break
		}
	}
	if found {
		users.s = users.s[:len(users.s)-1]
	}
	users.Unlock()
	return
}

func (users *UsersSet) FindByNick(nickname string) *User {
	users.RLock()
	defer users.RUnlock()
	for _, u := range users.s {
		if u.nickname == nickname {
			return u
		}
	}
	return nil
}

func (users *UsersSet) Add(u *User) {
	users.Lock()
	users.s = append(users.s, u)
	users.Unlock()
}

var Users UsersSet

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
		u.conn.SetDeadline(time.Now().Add(time.Second * 30))
		parseCommand(msg, u)
	}
	log.Println("removing user: ", u.nickname)
	removeUser(u)
}

func removeUser(u *User) {
	// send messages to user channels
	Channels.RLock()
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
	Channels.RUnlock()

	// removes user from global channels' set
	Channels.RLock()
	for _, c := range Channels.s {
		c.users.Remove(u)
	}
	Channels.RUnlock()

	// removes user from global users set
	Users.Remove(u)

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
			msg = "PING :" + u.nickname
		}
		log.Println(u.nickname + "\t<- " + msg)
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
	Users.Init()
	Channels.Init()

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

		Users.Add(user)

		go sendtoClient(user)
		go listenClient(user)
	}
}
