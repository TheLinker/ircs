package main

import (
	"sync"
	"net"
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

type User struct {
	conn        net.Conn
	nickname    string
	username    string
	realname    string
	hostname    string
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
	users.Lock()
	for i := range users.s {
		if users.s[i] == u {
			ret = users.s[i]
			users.s[i] = users.s[len(users.s)-1]
			users.s = users.s[:len(users.s)-1]
			break
		}
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
