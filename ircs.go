package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

func parseCommand(message string, u *User) bool {
	var prefix, command, argv string

	if len(message) == 0 {
		return false
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
		if u.status >= handler.minConStatus {
			handler.handler(u, prefix, argv)
		} else {
			Replay(u.out, server.Hostname,
				"ERR_NOTREGISTERED", u.nickname)
		}
	} else {
		log.Println("Command not found: " + command)
	}
	return command == "QUIT"
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
		if !strings.Contains(msg, "PING") && !strings.Contains(msg, "PONG") {
			log.Println(u.nickname + "\t-> " + msg)
		}
		u.conn.SetDeadline(time.Now().Add(time.Second * 30))
		if parseCommand(msg, u) {
			break
		}
	}
	log.Println("Removing user: ", u.nickname)
	removeUser(u)
}

func removeUser(u *User) {
	server.channels.RLock()
	for _, c := range server.channels.s {
		c.users.Remove(u)
	}
	server.channels.RUnlock()

	// removes user from global users set
	server.users.Remove(u)

	// send messages to user channels
	u.channels.RLock()
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
	u.channels.RUnlock()

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
			if u.status == CONN_CONNECTED {
				msg = "PING :" + u.nickname
			}
		}
		if len(msg) != 0 {
			if !strings.Contains(msg, "PING") && !strings.Contains(msg, "PONG") {
				log.Println(u.nickname + "\t<- " + msg)
			}
			msg += "\r\n"
			_, err := u.conn.Write([]byte(msg))
			if err != nil {
				log.Println(err)
				break
			}
		}
	}
	pinger.Stop()
}

func main() {
	server.Created = time.Now().Format("2006/01/02 15:04:05")
	//llenar con configfile, pero bueh
	server.Hostname = "bayerl.com.ar"
	server.Name = "MyIRCServer"
	server.Version = "0.0.0.0.0.0.0.1"
	server.ListenAddr = ":2000"

	file, err := os.Open("conf.json")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&server)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	server.users.Init()
	server.channels.Init()

	l, err := net.Listen("tcp", server.ListenAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(fmt.Sprintf("Server %s listening on: %s", server.Name, server.ListenAddr))
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		conn.SetDeadline(time.Now().Add(time.Second * 30))

		user := new(User)
		user.Init()
		user.conn = conn

		log.Println("Connection from: " + conn.RemoteAddr().String())

		server.users.Add(user)

		go sendtoClient(user)
		go listenClient(user)
	}
}
