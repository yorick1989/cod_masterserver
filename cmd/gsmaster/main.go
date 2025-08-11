package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/yorick1989/cod_masterserver/cmd/gsmaster/server"
)

var (
	templmu sync.RWMutex
	templ   struct {
		TotalPlayers     int
		TotalGameservers int
		Gameservers      []map[string]interface{}
	}
)

func main() {

	var web_ip = flag.String("webip", "0.0.0.0", "The ip where the webserver listens on")
	var web_port = flag.Int("webport", 8080, "The port where the webserver listens on")

	var codmaster_ip = flag.String("codmasterip", "0.0.0.0", "Cod master server ip")
	var codmaster_port = flag.Int("codmasterport", 20710, "Cod master server port")

	var codauth_enable = flag.Bool("codauth", false, "Enable the Cod auth server")
	var codauth_ip = flag.String("codauthip", "0.0.0.0", "Cod auth server ip")
	var codauth_port = flag.Int("codauthport", 20700, "Cod auth server port")

	var gscheck = flag.Duration("gscheck", 30*time.Second, "Amount of seconds between the checks of the gameservers.")
	var querytimeout = flag.Duration("querytimeout", 30*time.Second, "UDP Query timeout (for both COD master and COD update servers)")

	flag.Parse()

	// COD master server

	chmsg := make(chan interface{})

	go handleGameservers(&chmsg)

	codmaster := server.NewCodMaster(
		*codmaster_ip,
		*codmaster_port,
		1024,
		*gscheck,
		*querytimeout,
	)

	codmaster.AddMasterServer("codmaster.activision.com:20510", []int{0})   // auth: 20500
	codmaster.AddMasterServer("cod2master.activision.com:20710", []int{0})  // auth: 20700
	codmaster.AddMasterServer("master.cod2x.me:20710", []int{0})            // auth: 20700
	codmaster.AddMasterServer("cod4master.activision.com:20810", []int{0})  // auth: 20800
	codmaster.AddMasterServer("coduomaster.activision.com:20610", []int{0}) // auth: 20600

	codmaster.AddUpdateHandlerFunc(func(gameservers map[string]*server.Gameserver) {
		chmsg <- gameservers
	})

	go func() {
		for {
			if err := codmaster.StartListener(); err != nil {
				log.Fatal(err)
			}
		}
	}()

	// COD auth server

	if *codauth_enable {
		codauth := server.NewCodMaster(
			*codauth_ip,
			*codauth_port,
			1024,
			*gscheck,
			*querytimeout,
		)

		go func() {
			for {
				if err := codauth.StartListener(); err != nil {
					log.Fatal(err)
				}
			}
		}()

	}

	// HTTP/WS server

	srv := NewServer()

	router := http.NewServeMux()
	router.HandleFunc("/", handleRoot)
	router.HandleFunc("/serverlist", handleServerlist)
	//router.Handle("/ws", websocket.Handler(srv.handleWS))

	httpserver := http.Server{
		Addr:    fmt.Sprintf("%s:%d", *web_ip, *web_port),
		Handler: router,
	}

	log.Println("Starting server on", httpserver.Addr)

	go srv.writeLoop()

	err := httpserver.ListenAndServe()

	if err != nil {
		fmt.Printf("could not start the http server\n", err)
		os.Exit(1)
	}

}
