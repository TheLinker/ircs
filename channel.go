package main

import (
    "strings"
    "sync"
)

type Channel struct {
    name  string
    topic string
    topicuser string
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

func (set *ChannelsSet) Remove(u *Channel) (ret *Channel) {
    set.Lock()
    delete(set.s, strings.ToUpperSpecial(IRCCase, u.name))
    set.Unlock()
    return
}
