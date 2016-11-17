package main

import (
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

type Server struct {
	Password   string
	Hostname   string
	Name       string
	Version    string
	Created    string
	ListenAddr string
	channels   ChannelsSet
	users      UsersSet
}

var server Server

type Msg struct {
	user *User
	msg  string
}
