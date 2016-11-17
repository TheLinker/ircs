package main

import (
    "net"
    "strings"
    "sync"
)

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

func (user *User) Init() {
    user.out = make(chan string, 20)
    user.channels.Init()

    if len(server.Password) == 0 {
        user.status = CONN_PASS_OK
    } else {
        user.status = CONN_ESTABLISHED
    }
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
