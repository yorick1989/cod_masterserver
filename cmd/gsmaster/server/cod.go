package server

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yorick1989/cod_masterserver/pkg/gsquery"
)

type Gameserver struct {
	LastCheck    int
	MasterServer string
	sync.Mutex
	gsquery.Gameserver
}

type UpdateHandlerFunc func(map[string]*Gameserver)

type CodMaster struct {
	ip             net.IP
	port           int
	maxbufsize     int
	conn           *net.UDPConn
	currcheck      int64
	gscheck        time.Duration
	gsexpired      int64
	gsauth         map[string]string
	querytimeout   time.Duration
	guid_hash_key  string
	customrequests map[string]func(*net.UDPConn, net.Addr, string) error
	gameservers    map[string]*Gameserver
	masterServers  map[string][]int
	updatehandlers []UpdateHandlerFunc
	sync.RWMutex
}

func NewCodMaster(ip string, port int, maxbufsize int, gscheck time.Duration, querytimeout time.Duration) *CodMaster {

	addr, err := net.ResolveIPAddr("ip", ip)
	if err != nil {
		panic(err)
	}

	return &CodMaster{
		ip:             addr.IP,
		port:           port,
		maxbufsize:     maxbufsize,
		currcheck:      time.Now().Unix(),
		gscheck:        gscheck,
		gsexpired:      1800,
		gsauth:         make(map[string]string),
		querytimeout:   querytimeout,
		guid_hash_key:  "",
		customrequests: make(map[string]func(*net.UDPConn, net.Addr, string) error),
		gameservers:    make(map[string]*Gameserver),
		masterServers:  make(map[string][]int),
		updatehandlers: []UpdateHandlerFunc{},
	}

}

func (c *CodMaster) StartListener() error {

	c.RLock()

	addr := &net.UDPAddr{
		IP:   c.ip,
		Port: c.port,
		Zone: "",
	}

	log.Printf("Master server: start listener on '%s:%d'.\n", c.ip, c.port)

	if conn, err := net.ListenUDP("udp", addr); err != nil {

		c.RUnlock()
		return err
	} else {
		c.conn = conn
	}

	defer c.conn.Close()

	c.RUnlock()

	go func(gscheck time.Duration) {

		var masterservers map[string][]int = make(map[string][]int)

		log.Printf("Started the periodically check run (which runs every '%d' seconds).\n", gscheck)

		for {

			// Fetch the master server(s).
			c.RLock()
			for k, v := range c.masterServers {
				masterservers[k] = v
			}
			c.RUnlock()

			for msaddr, msprotocols := range masterservers {

				for _, msprotocol := range msprotocols {
					c.fetchMasterServer(msaddr, msprotocol)
				}

			}

			c.RLock()
			l := len(c.gameservers)
			c.RUnlock()

			if l > 0 {

				c.currcheck = time.Now().Unix()

				c.checkGameservers()

			}

			c.triggerUpdateHandlerFuncs()

			time.Sleep(gscheck)

		}

	}(c.gscheck)

	for {
		buf := make([]byte, c.maxbufsize)
		n, addr, err := c.conn.ReadFrom(buf)
		if err != nil {
			continue
		}

		reqdata := regexp.MustCompile(`[\s\n]`).Split(string(buf[4:n]), 2)

		if !bytes.Equal(buf[:4], []byte("\xFF\xFF\xFF\xFF")) || len(reqdata) < 2 {
			log.Printf("Master server: wrong data received from '%s': %q\n", addr, buf[:n])
			continue
		}

		reqtype, reqmsg := reqdata[0], reqdata[1]

		c.currcheck = time.Now().Unix()

		switch reqtype {
		case "heartbeat":
			reqmsg = strings.Trim(reqmsg, "\n")
			if reqmsg == "flatline" {
				ipport := addr.String()
				c.delGameserver(ipport)

				c.triggerUpdateHandlerFuncs()
			} else if slices.Contains([]string{"COD-2"}, reqmsg) {
				ipport := addr.String()

				gs, err := c.addGameserver(ipport)
				if err != nil {
					break
				}
				gs.MasterServer = c.conn.LocalAddr().String()

				c.triggerUpdateHandlerFuncs()
			} else {
				log.Printf("Master server: wrong data received from '%s': %q\n", addr, buf[:n])
			}
		case "infoResponse", "statusResponse":
			continue
			/*if reqtype == "statusResponse" {
				c.challengeGameserver(addr)
			}*/
		case "getservers":
			c.getGameservers(addr, reqmsg)
		case "getKeyAuthorize":
			reqdata := regexp.MustCompile(`[\s\n]`).Split(reqmsg, -1)

			if len(reqdata) >= 2 {
				ipport := strings.Split(addr.String(), ":")
				c.Lock()
				c.gsauth[ipport[0]] = reqdata[1]
				c.Unlock()
				log.Printf("Auth server: authorization request incoming for the gameserver '%s'\n", addr)
			}
		case "getIpAuthorize":
			c.authGameserver(addr, reqmsg)
		default:
			if f, ok := c.customrequests[reqtype]; ok {
				log.Printf("Master server: custom request triggered: '%s'\n", reqtype)
				f(c.conn, addr, reqmsg)
			} else {
				log.Printf("Master server: wrong data received from '%s': %q\n", addr, buf[:n])
			}
		}

	}

}

func (c *CodMaster) ClientIpPort(addr net.Addr) (ip string, port int) {

	ipport := strings.Split(addr.String(), ":")

	ip = ipport[0]

	port, _ = strconv.Atoi(ipport[1])

	return

}

func (c *CodMaster) triggerUpdateHandlerFuncs() {

	var gameservers map[string]*Gameserver = make(map[string]*Gameserver)

	c.RLock()
	gameservers = c.gameservers
	c.RUnlock()

	for _, f := range c.updatehandlers {
		go f(gameservers)
	}

}

func (c *CodMaster) AddUpdateHandlerFunc(f UpdateHandlerFunc) {

	c.Lock()
	c.updatehandlers = append(c.updatehandlers, f)
	c.Unlock()

	log.Printf("Master server: update handler func added: %v\n", f)

}

func (c *CodMaster) AddCustomRequest(request string, f func(*net.UDPConn, net.Addr, string) error) {

	c.Lock()
	c.customrequests[request] = f
	c.Unlock()

	log.Printf("Master server: custom handler added: %s\n", request)

}

func (c *CodMaster) AddMasterServer(ipport string, protocols []int) error {

	c.Lock()
	c.masterServers[ipport] = protocols
	c.Unlock()

	return nil
}

func (c *CodMaster) fetchMasterServer(ipport string, protocol int) error {

	var wg sync.WaitGroup

	var gscount int = 0
	var gsfetched int = 0
	var gsfailcount int = 0

	data := []byte(fmt.Sprintf("\xFF\xFF\xFF\xFFgetservers\n%d", protocol))

	raddr, err := net.ResolveUDPAddr("udp", ipport)
	if err != nil {
		return fmt.Errorf("could not resolve the ip")
	}

	conn, err := net.DialUDP("udp", nil, raddr)

	if err != nil {
		c.RLock()
		defer c.RUnlock()
		return fmt.Errorf("could not connect to the remote server '%s': %s", ipport, err)
	}

	defer conn.Close()

	for {
		conn.Write(data)

		c.RLock()
		buf := make([]byte, c.maxbufsize)

		conn.SetReadDeadline(time.Now().Add(c.querytimeout))
		c.RUnlock()

		n, err := conn.Read(buf)
		if err != nil {
			return fmt.Errorf("could not read from the remote server '%s': %s", conn.RemoteAddr(), err)
		}

		reqdata := regexp.MustCompile(`[\s\n]`).Split(string(buf[4:n]), 2)

		if !bytes.Equal(buf[:4], []byte("\xFF\xFF\xFF\xFF")) || len(reqdata) < 2 {
			return fmt.Errorf("wrong data received from '%s': %q", conn.RemoteAddr(), buf[:n])
		}

		reqtype, reqmsg := reqdata[0], reqdata[1]

		if reqtype == "getserversResponse" && bytes.Equal([]byte(reqmsg[:1]), []byte("\x00")) {

			for _, gsdata := range strings.Split(reqmsg, "\\") {
				if gsdata == "EOF" {

					wg.Wait()

					if gsfetched > 0 {
						log.Printf("Master server: fetched '%d' of the total '%d' ('%d' failed) servers with the protocol '%d' from the master server '%s'.\n", gsfetched, gscount, gsfailcount, protocol, ipport)
					}

					return nil

				} else if len(gsdata) != 6 {
					continue
				}

				gsipport := fmt.Sprintf("%d.%d.%d.%d:%d", byte(gsdata[0]), byte(gsdata[1]), byte(gsdata[2]), byte(gsdata[3]), binary.BigEndian.Uint16([]byte(gsdata[4:])))

				wg.Add(1)

				go func(ipport string, gsipport string) {
					defer wg.Done()

					c.RLock()
					_, ok := c.gameservers[gsipport]
					c.RUnlock()

					if ok {
						c.Lock()
						gscount++
						c.Unlock()
						return
					}
					gs, err := c.addGameserver(gsipport)

					if err != nil {
						c.Lock()
						gsfailcount++
						c.Unlock()
						return
					}
					gs.MasterServer = ipport

					c.Lock()
					gscount++
					gsfetched++
					c.Unlock()

				}(ipport, gsipport)

			}

		}

	}

}

func (c *CodMaster) addGameserver(ipport string) (*Gameserver, error) {

	c.Lock()
	gs, ok := c.gameservers[ipport]
	c.Unlock()

	if ok {
		return gs, nil
	}

	raddr, err := net.ResolveUDPAddr("udp", ipport)
	if err != nil {
		return nil, fmt.Errorf("could not resolve the ip")
	}

	conn, err := net.DialUDP("udp", nil, raddr)

	if err != nil {
		return nil, fmt.Errorf("could not connect to the ip")
	}

	gs = &Gameserver{}

	gs.Ipport = ipport

	gs.Protocol = &gsquery.ProtocolCod2{}

	gs.Udpconn = conn

	gs.GetInfo()

	gs.LastCheck = int(c.currcheck)

	if len(gs.Info) == 0 {
		if gs.Udpconn != nil {
			gs.Udpconn.Close()
		}
		return nil, fmt.Errorf("could not receive the gameserver info")
	}

	c.Lock()
	c.gameservers[ipport] = gs
	c.Unlock()

	log.Printf("Master server: added the gameserver '%s'.\n", ipport)

	return gs, nil

}

func (c *CodMaster) delGameserver(ipport string) {

	c.RLock()
	_, ok := c.gameservers[ipport]
	c.RUnlock()

	if ok {
		c.Lock()
		lc := c.gameservers[ipport].LastCheck
		delete(c.gameservers, ipport)
		c.Unlock()
		log.Printf("Master server: deleted the gameserver '%s'. Last check: %d\n", ipport, lc)
	}

}

func (c *CodMaster) getGameservers(addr net.Addr, reqmsg string) {

	ipport := addr.String()

	reqdata := regexp.MustCompile(`[\s\n]`).Split(reqmsg, 2)

	if len(reqdata) <= 0 {
		return
	}

	protocol := reqdata[0]

	per_packet_count := 0
	max_per_packet := 20

	header := []byte("\xFF\xFF\xFF\xFFgetserversResponse\n\x00")
	delimiter := []byte("\\")
	data := append(header[:], delimiter[:]...)

	var gameservers map[string]*Gameserver = make(map[string]*Gameserver)

	c.RLock()
	for k, v := range c.gameservers {
		gameservers[k] = v
	}
	c.RUnlock()

	for gsaddr, gs := range gameservers {

		gs.Lock()
		prot, ok := gs.Info["protocol"]
		gs.Unlock()

		if !ok || prot != protocol {
			continue
		}

		ipaddr := regexp.MustCompile(`[.:]`).Split(gsaddr, -1)

		if len(ipaddr) != 5 {
			continue
		}

		var gsip [6]byte

		if octet, err := strconv.Atoi(ipaddr[0]); err == nil {
			gsip[0] = byte(octet)
		} else {
			continue
		}

		if octet, err := strconv.Atoi(ipaddr[1]); err == nil {
			gsip[1] = byte(octet)
		} else {
			continue
		}

		if octet, err := strconv.Atoi(ipaddr[2]); err == nil {
			gsip[2] = byte(octet)
		} else {
			continue
		}

		if octet, err := strconv.Atoi(ipaddr[3]); err == nil {
			gsip[3] = byte(octet)
		} else {
			continue
		}

		gsport, err := strconv.ParseUint(ipaddr[4], 10, 16)
		if err != nil {
			continue
		}

		binary.BigEndian.PutUint16(gsip[4:6], uint16(gsport))

		// Append the gsip byte slice to the data byte slice.
		data = append(data[:], gsip[:]...)

		// Append the delimiter byte to the data byte slice.
		data = append(data[:], delimiter[:]...)

		per_packet_count += 1

		if per_packet_count == max_per_packet {

			data = append(data[:], []byte("EOT")[:]...)

			if _, err := c.conn.WriteTo(data, addr); err != nil {
				return
			}

			per_packet_count = 0
			data = append(header[:], delimiter[:]...)

		}

	}

	data = append(data[:], []byte("EOF")[:]...)

	if _, err := c.conn.WriteTo(data, addr); err != nil {
		return
	}

	log.Printf("Master server: sent the list of gameservers to '%s'\n", ipport)

}

func (c *CodMaster) checkGameservers() {

	var wg sync.WaitGroup

	var gameservers map[string]*Gameserver = make(map[string]*Gameserver)

	c.Lock()
	for k, v := range c.gameservers {
		gameservers[k] = v
	}
	c.Unlock()

	for gsaddr, gsdata := range gameservers {

		wg.Add(1)

		go func(ipport string, gs *Gameserver) {
			defer wg.Done()

			gs.Lock()

			if gs.LastCheck == 0 {
				c.delGameserver(ipport)
				return
			} else if (c.currcheck-int64(gs.LastCheck)) >= c.gsexpired || gs.Info["protocol"] == "0" {

				if _, err := gs.GetInfo(); err != nil {
					c.delGameserver(ipport)
					return
				}

			} else if (c.currcheck - int64(gs.LastCheck)) >= int64(c.gscheck.Seconds()) {
				gs.GetInfo()
			}

			gs.LastCheck = int(c.currcheck)

			gs.Unlock()

		}(gsaddr, gsdata)

	}

	wg.Wait()

	log.Printf("Master server: checked all gameservers.\n")

}

func (c *CodMaster) authGameserver(addr net.Addr, reqmsg string) {

	ipport := addr.String()

	guid := "0"
	pbguid := "0"

	reqdata := regexp.MustCompile(`[\s\n]`).Split(reqmsg, -1)

	if len(reqdata) < 3 {
		return
	}

	authkey := reqdata[0]
	authip := reqdata[1]

	if val, ok := c.gsauth[authip]; ok && len(c.guid_hash_key) > 0 {

		h := md5.New()
		io.WriteString(h, val)
		io.WriteString(h, c.guid_hash_key)

		pbguid = hex.EncodeToString(h.Sum(nil))

		bi := big.NewInt(0)

		bi.SetString(pbguid, 16)

		guid = bi.String()

		if len(guid) > 6 {
			guid = guid[len(guid)-6:]
		}

	}

	data := []byte(fmt.Sprintf("\xFF\xFF\xFF\xFFipAuthorize %s accept KEY_IS_GOOD %v %v", authkey, guid, pbguid))
	if _, err := c.conn.WriteTo(data, addr); err != nil {
		return
	}

	log.Printf("Auth server: authorized the gameserver '%s'\n", ipport)

}
