package gsquery

import (
	"bytes"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Gameserverstats map[string]string

type Gameserverplayers []map[string]string

type SanitizedGameserverstats struct {
	Ip                string
	Port              int
	Hostname          string
	CurrentPlayers    int
	MaximumPlayers    int
	Map               string
	PasswordProtected bool
}

type SanitizedGameserverplayer struct {
	Id    int
	Name  string
	Score int
	Ping  int
}

type SanitizedGameserverplayers []SanitizedGameserverplayer

type Gameserver struct {
	Ipport    string
	Protocol  ProtocolType
	Stats     Gameserverstats
	Info      Gameserverstats
	Players   Gameserverplayers
	Sanitized struct {
		Stats   SanitizedGameserverstats
		Players SanitizedGameserverplayers
	}
	Udpconn net.Conn
	sync.RWMutex
}

type ProtocolType interface {
	GetStats() (*Gameserver, error)
}

var supportedProtocols = []interface{}{
	ProtocolCod2{},
}

type ProtocolCod2 struct {
}

func udpSetup(ipport string) (net.Conn, error) {

	raddr, err := net.ResolveUDPAddr("udp", ipport)
	if err != nil {
		return nil, fmt.Errorf("could not resolve the ip")
	}

	conn, err := net.DialUDP("udp", nil, raddr)

	if err != nil {
		return nil, fmt.Errorf("could not connect to the ip")
	}

	return conn, nil

}

func GetInfo(ipport string) (*Gameserver, error) {

	conn, err := udpSetup(ipport)

	if err != nil {
		return nil, err
	}

	for _, p := range supportedProtocols {
		switch protocol := p.(type) {
		case ProtocolCod2:
			gameserver := NewGameserver(ipport, &protocol, conn)

			if _, err := gameserver.GetInfo(); err == nil {
				return gameserver, nil
			}

			continue

		}
	}

	return nil, fmt.Errorf("no known protocol found")

}

func GetStats(ipport string) (*Gameserver, error) {

	conn, err := udpSetup(ipport)

	if err != nil {
		return nil, err
	}

	for _, p := range supportedProtocols {
		switch protocol := p.(type) {
		case ProtocolCod2:
			gameserver := NewGameserver(ipport, &protocol, conn)

			if _, err := gameserver.GetStats(); err == nil {
				return gameserver, nil
			}

			continue

		}
	}

	return nil, fmt.Errorf("no known protocol found")

}

func protocolRequestResponse(conn net.Conn, request []byte, response []byte) ([]byte, int, error) {

	if conn == nil {
		return []byte{}, 0, fmt.Errorf("no connection established")
	}

	// Fetch gameserver info
	conn.Write(request)

	buff := make([]byte, 1024)

	// Change back to 30
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	n, err := conn.Read(buff)

	if err != nil {
		return []byte{}, 0, err
	} else if n <= len(response) || !bytes.Equal(buff[:len(response)], response) {
		return []byte{}, 0, fmt.Errorf("wrong protocol")
	}

	return buff, n, nil

}

func (g *Gameserver) processStats(buff []byte, size int) error {

	stats := make(Gameserverstats)

	players := Gameserverplayers{}

	response := []byte("\xFF\xFF\xFF\xFFstatusResponse\n")

	for {

		gsdata := strings.Split(string(buff[len(response)+1:]), "\n")
		gsstats := strings.Split(gsdata[0], "\\")

		for i := 0; i+1 < len(gsstats); i += 2 {
			stats[strings.ToLower(gsstats[i])] = gsstats[i+1]
		}

		for _, gsplayerinfo := range gsdata[1:] {
			gsplayer := strings.Split(gsplayerinfo, " ")
			if len(gsplayer) == 3 {
				players = append(players, map[string]string{
					"name":  strings.Trim(gsplayer[2], `"`),
					"frags": gsplayer[0],
					"ping":  gsplayer[1],
				})
			}
		}
		if !bytes.Equal(buff[size-3:], []byte("EOT")[:]) {
			break
		}

	}

	sort.Slice(players, func(i, j int) bool {
		v, _ := strconv.Atoi(players[i]["frags"])
		v2, _ := strconv.Atoi(players[j]["frags"])
		return v > v2
	})

	g.Lock()

	g.Stats = stats

	g.Players = players

	g.Unlock()

	g.SanitizeStats()

	return nil

}

func (g *Gameserver) GetInfo() (*Gameserver, error) {

	for i := 0; i <= 3; i++ {

		g.RLock()
		if g.Udpconn == nil {
			defer g.RUnlock()
			return g, nil
		}
		buff, size, err := protocolRequestResponse(g.Udpconn, []byte("\xFF\xFF\xFF\xFFgetinfo"), []byte("\xFF\xFF\xFF\xFFinfoResponse\n"))
		g.RUnlock()

		if err != nil {
			if i < 3 {
				g.Lock()
				if conn, err := udpSetup(g.Ipport); err != nil {
					g.Udpconn = conn
				}
				g.Unlock()
				continue
			}
			return nil, err
		} else if err := g.processInfo(buff, size); err != nil {
			return nil, err
		} else {
			return g, nil
		}

	}

	return nil, fmt.Errorf("could not connect to the server")

}

func (g *Gameserver) processInfo(buff []byte, size int) error {

	info := make(Gameserverstats)

	response := []byte("\xFF\xFF\xFF\xFFinfoResponse\n")

	for {

		gsdata := strings.Split(string(buff[len(response)+1:]), "\n")
		gsstats := strings.Split(gsdata[0], "\\")

		for i := 0; i+1 < len(gsstats); i += 2 {
			info[strings.ToLower(gsstats[i])] = gsstats[i+1]
		}

		if !bytes.Equal(buff[size-3:], []byte("EOT")[:]) {
			break
		}

	}

	g.Lock()

	g.Info = info

	g.Unlock()

	return nil

}

func (p *ProtocolCod2) GetStats() (*Gameserver, error) {

	return nil, fmt.Errorf("could not connect to the server")

}

func (g *Gameserver) GetStats() (*Gameserver, error) {

	for i := 0; i <= 3; i++ {

		g.Lock()
		buff, size, err := protocolRequestResponse(g.Udpconn, []byte("\xFF\xFF\xFF\xFFgetstatus"), []byte("\xFF\xFF\xFF\xFFstatusResponse\n"))
		g.Unlock()

		if err != nil {
			if i < 3 {
				g.Lock()
				if conn, err := udpSetup(g.Ipport); err != nil {
					g.Udpconn = conn
				}
				g.Unlock()
				continue
			}
			return nil, err
		} else if err := g.processStats(buff, size); err != nil {
			return nil, err
		} else {
			return g, nil
		}

	}

	return nil, fmt.Errorf("could not connect to the server")

}

func (g *Gameserver) SanitizeStats() error {

	g.Lock()

	defer g.Unlock()

	g.Sanitized.Players = SanitizedGameserverplayers{}

	for _, v := range g.Players {

		data := SanitizedGameserverplayer{
			Name: v["name"],
		}

		if score, err := strconv.Atoi(v["frags"]); err == nil {
			data.Score = score
		}

		if ping, err := strconv.Atoi(v["ping"]); err == nil {
			data.Ping = ping
		}

		g.Sanitized.Players = append(g.Sanitized.Players, data)

	}

	ipport := strings.Split(g.Ipport, ":")
	g.Sanitized.Stats.Ip = ipport[0]
	g.Sanitized.Stats.Hostname = g.Stats["sv_hostname"]
	g.Sanitized.Stats.Map = g.Stats["mapname"]
	g.Sanitized.Stats.CurrentPlayers = int(len(g.Sanitized.Players))
	g.Sanitized.Stats.PasswordProtected = bool(g.Stats["pswrd"] == "1")

	if port, err := strconv.Atoi(ipport[1]); err == nil {
		g.Sanitized.Stats.Port = port
	}

	if maxclients, err := strconv.Atoi(g.Stats["sv_maxclients"]); err == nil {
		g.Sanitized.Stats.MaximumPlayers = maxclients
	}

	return nil

}

func NewGameserver(ipport string, protocol ProtocolType, conn net.Conn) *Gameserver {
	return &Gameserver{
		Ipport:   ipport,
		Protocol: protocol,
		Stats:    make(Gameserverstats),
		Info:     make(Gameserverstats),
		Players:  Gameserverplayers{},
		Sanitized: struct {
			Stats   SanitizedGameserverstats
			Players SanitizedGameserverplayers
		}{
			Stats:   SanitizedGameserverstats{},
			Players: SanitizedGameserverplayers{},
		},
		Udpconn: conn,
	}
}
