package main

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"log"
	"net/http"

	cli "github.com/urfave/cli/v2"

	"golang.org/x/crypto/ssh"
	goproxy "gopkg.in/elazarl/goproxy.v1"
)

func main() {

	wdDir := filepath.Dir(os.Args[0])

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "host",
				Aliases: []string{"h"},
			},
			&cli.StringFlag{
				Name:    "port",
				Aliases: []string{"p"},
			},
			&cli.StringFlag{
				Name:    "user",
				Aliases: []string{"u"},
			},
		},
		HideHelpCommand: true,
		HideHelp:        true,
		Action: func(c *cli.Context) error {
			s := fmt.Sprintf("host %s, port %s, user %s, parameter %s ",
				c.String("host"),
				c.String("port"),
				c.String("user"),
				c.Args().Get(0),
			)
			ioutil.WriteFile(wdDir+"\\secureshell2moba.log", []byte(s), fs.ModePerm)
			config := &ssh.ClientConfig{
				User:            c.String("user"),
				Auth:            []ssh.AuthMethod{ssh.Password("SSH_PASSWORD_HERE")},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			}
			conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", c.String("host"), c.String("port")), config)
			if err != nil {
				log.Fatal(err)
			}
			defer conn.Close()
			go func() {
				logger := log.New(os.Stderr, "http2socks: ", log.LstdFlags|log.Lshortfile)
				prxy := goproxy.NewProxyHttpServer()

				r := &RemoteDialer{Conn: conn}

				prxy.Tr = &http.Transport{Dial: r.Dial}
				fmt.Println("Listen")
				logger.Fatal(http.ListenAndServe("127.0.0.1:3080", prxy))
			}()

			session, _ := conn.NewSession()
			session.Stdin = os.Stdin
			session.Stdout = os.Stdout
			modes := ssh.TerminalModes{
				ssh.ECHO: 0, // supress echo

			}
			// run terminal session
			if err := session.RequestPty("xterm", 50, 80, modes); err != nil {
				log.Fatal(err)
			}
			// start remote shell
			if err := session.Shell(); err != nil {
				log.Fatal(err)
			}
			ch := make(chan os.Signal)
			signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
			<-ch
			return nil
		},
	}
	app.Run(os.Args)
}

type RemoteDialer struct {
	Conn *ssh.Client
}

func (r *RemoteDialer) Dial(network, addr string) (c net.Conn, err error) {
	fmt.Printf("network: %s, addr: %s", network, addr)
	return r.Conn.Dial(network, addr)
}
