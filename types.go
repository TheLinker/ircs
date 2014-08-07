package main

import (
	"net"
	"strings"
	"sync"
	"unicode"
)

type ConnStatus int

const (
	CONN_ESTABLISHED = iota
	CONN_PASS_OK
	CONN_NICK_OK
	CONN_CONNECTED
)

var IRCCase = unicode.SpecialCase{
	unicode.CaseRange{0x5b, 0x5b, [unicode.MaxCase]rune{0, 0x7b - 0x5b, 0}},           //[ U -> { L
	unicode.CaseRange{0x5c, 0x5c, [unicode.MaxCase]rune{0, 0x7c - 0x5c, 0}},           //\ U -> | L
	unicode.CaseRange{0x5d, 0x5d, [unicode.MaxCase]rune{0, 0x7d - 0x5d, 0}},           //] U -> } L
	unicode.CaseRange{0x5e, 0x5e, [unicode.MaxCase]rune{0x7e - 0x5e, 0, 0x7e - 0x5e}}, //^ L -> ~ U
	unicode.CaseRange{0x7b, 0x7b, [unicode.MaxCase]rune{0x5b - 0x7b, 0, 0x5b - 0x7b}}, //{ L -> [ U
	unicode.CaseRange{0x7c, 0x7c, [unicode.MaxCase]rune{0x5c - 0x7c, 0, 0x5c - 0x7c}}, //| L -> \ U
	unicode.CaseRange{0x7d, 0x7d, [unicode.MaxCase]rune{0x5d - 0x7d, 0, 0x5d - 0x7d}}, //} L -> ] U
	unicode.CaseRange{0x7e, 0x7e, [unicode.MaxCase]rune{0, 0x5e - 0x7e, 0}},           //~ U -> ^ L
}

type Msg struct {
	user *User
	msg  string
}

type Channel struct {
	name  string
	topic string
	users UsersSet
	out   chan Msg
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
	set.s[strings.ToUpperSpecial(IRCCase, k)] = c
	set.Unlock()
}

func (set *ChannelsSet) Get(k string) (c *Channel, ok bool) {
	set.RLock()
	defer set.RUnlock()
	c, ok = set.s[strings.ToUpperSpecial(IRCCase, k)]
	return
}

type User struct {
	conn     net.Conn
	nickname string
	username string
	realname string
	hostname string
	out      chan string
	channels ChannelsSet
	status   ConnStatus
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
		if strings.ToUpperSpecial(IRCCase, u.nickname) ==
			strings.ToUpperSpecial(IRCCase, nickname) {
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

type Server struct {
	password string
	hostname string
	name     string
	version  string
	created  string
}

var server Server
