package main

import (
//	"crypto/tls"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
	"log"

	"github.com/yorick1989/cod_masterserver/cmd/gsmaster/server"
	"github.com/yorick1989/cod_masterserver/pkg/gsquery"
)

var map_preview_processed map[string]bool = make(map[string]bool)

func handleGameservers(chmsg *chan interface{}) {

	protocol_mapping := map[string]string{
		"cod1_1":   "cod1-1_1",
		"cod1_5":   "cod1-1_4",
		"cod1_6":   "cod1-1_5",
		"cod2_115": "cod2-1_0",
		"cod2_117": "cod2-1_2",
		"cod2_118": "cod2-1_3",
		"cod2_119": "cod2-1_3",
		"cod2_120": "cod2-1_4",
		"cod4_1":   "cod4-1_0",
		"cod4_21":  "cod4-1_7",
		"cod4_7":   "cod4-1_8",
		"coduo_21": "coduo-1_41",
		"coduo_22": "coduo-1_51",
	}

	game_mapping := map[string]string{
		"codmaster.activision.com:20510":   "cod1",
		"cod2master.activision.com:20710":  "cod2",
		"master.cod2x.me:20710":            "cod2",
		"cod4master.activision.com:20810":  "cod4",
		"coduomaster.activision.com:20610": "coduo",
		"[::]:20710":                       "cod2",
	}

	for msg := range *chmsg {
		switch msg.(type) {
		case map[string]*server.Gameserver:
			var mu sync.RWMutex
			var wg sync.WaitGroup

			template_gameservers := struct {
				TotalPlayers     int
				TotalGameservers int
				Gameservers      []map[string]interface{}
			}{
				TotalGameservers: len(msg.(map[string]*server.Gameserver)),
				Gameservers:      []map[string]interface{}{},
			}

			req := &http.Client{
				Timeout: 5 * time.Second,
			}
			//	Transport: &http.Transport{
			//		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			//	},

			for ipport, gs := range msg.(map[string]*server.Gameserver) {
				wg.Add(1)

				go func() {
					defer wg.Done()

					_, err := gs.GetStats()
					if err != nil {
						return
					}

					var hostname_colorized string

					data := strings.Split(gs.Sanitized.Stats.Hostname, "^")
					if len(data) > 1 {
						for f := range data {
							if len(data[f]) < 1 {
								continue
							}
							hn := string(data[f])
							switch string(hn[0]) {
							case "0":
								hostname_colorized += `<span style="color: #000000">` + hn[1:] + `</span>`
							case "1":
								hostname_colorized += `<span style="color: #cc0000">` + hn[1:] + `</span>`
							case "2":
								hostname_colorized += `<span style="color: #33cc00">` + hn[1:] + `</span>`
							case "3":
								hostname_colorized += `<span style="color: #ffcc00">` + hn[1:] + `</span>`
							case "4":
								hostname_colorized += `<span style="color: #3333ff">` + hn[1:] + `</span>`
							case "5":
								hostname_colorized += `<span style="color: #66cccc">` + hn[1:] + `</span>`
							case "6":
								hostname_colorized += `<span style="color: #993399">` + hn[1:] + `</span>`
							case "7":
								hostname_colorized += `<span style="color: #ffffff">` + hn[1:] + `</span>`
							case "8":
								hostname_colorized += `<span style="color: #cc0000">` + hn[1:] + `</span>`
							case "9":
								hostname_colorized += `<span style="color: #999999">` + hn[1:] + `</span>`
							default:
								hostname_colorized += string(data[f])
							}
						}
					} else {
						hostname_colorized = gs.Sanitized.Stats.Hostname
					}

					players_colorized := []gsquery.SanitizedGameserverplayer{}

					mu.Lock()
					template_gameservers.TotalPlayers += len(gs.Sanitized.Players)
					mu.Unlock()

					for _, v := range gs.Sanitized.Players {

						var player_name string

						data := strings.Split(v.Name, "^")
						if len(data) > 1 {
							for f := range data {
								if len(data[f]) < 1 {
									continue
								}
								pn := string(data[f])
								switch string(pn[0]) {
								case "0":
									player_name += `<span style="color: #000000">` + pn[1:] + `</span>`
								case "1":
									player_name += `<span style="color: #cc0000">` + pn[1:] + `</span>`
								case "2":
									player_name += `<span style="color: #33cc00">` + pn[1:] + `</span>`
								case "3":
									player_name += `<span style="color: #ffcc00">` + pn[1:] + `</span>`
								case "4":
									player_name += `<span style="color: #3333ff">` + pn[1:] + `</span>`
								case "5":
									player_name += `<span style="color: #66cccc">` + pn[1:] + `</span>`
								case "6":
									player_name += `<span style="color: #993399">` + pn[1:] + `</span>`
								case "7":
									player_name += `<span style="color: #ffffff">` + pn[1:] + `</span>`
								case "8":
									player_name += `<span style="color: #cc0000">` + pn[1:] + `</span>`
								case "9":
									player_name += `<span style="color: #999999">` + pn[1:] + `</span>`
								default:
									player_name += string(data[f])
								}
							}
						} else {
							player_name = v.Name
						}
						players_colorized = append(players_colorized, gsquery.SanitizedGameserverplayer{
							Id:    v.Id,
							Name:  player_name,
							Score: v.Score,
							Ping:  v.Ping,
						})
					}

					var version string

					if k, ok := protocol_mapping[game_mapping[gs.MasterServer]+"_"+gs.Stats["protocol"]]; ok {
						version = k
					} else {
						version = game_mapping[gs.MasterServer] + "_protocol_" + gs.Stats["protocol"]
					}

					var map_preview string

					if game_mapping[gs.MasterServer] != "cod1" && game_mapping[gs.MasterServer] != "coduo" {
						map_preview = fmt.Sprintf("https://cod.pm/mp_maps/%s/stock/%s.png", game_mapping[gs.MasterServer], gs.Sanitized.Stats.Map)
					} else {
						map_preview = fmt.Sprintf("https://cod.pm/mp_maps/%s/stock/%s.png", "cod1+coduo", gs.Sanitized.Stats.Map)
					}

					mu.Lock()
					if _, ok := map_preview_processed[map_preview]; ok {

					  if !map_preview_processed[map_preview] {
							map_preview = "https://cod.pm/mp_maps/unknown.png"
					  }

					} else {

						resp, err := req.Get(map_preview)

						if err != nil {
							map_preview = "https://cod.pm/mp_maps/unknown.png"
						} else {
							if resp.StatusCode == 200 {
								map_preview_processed[map_preview] = true
							} else {
								map_preview_processed[map_preview] = false
								map_preview = "https://cod.pm/mp_maps/unknown.png"
							}
							resp.Body.Close()
						}

					}

					template_gameservers.Gameservers = append(template_gameservers.Gameservers, map[string]interface{}{
						"ipport":             ipport,
						"hostname":           gs.Sanitized.Stats.Hostname,
						"hostname_colorized": hostname_colorized,
						"map":                gs.Sanitized.Stats.Map,
						"curclients":         gs.Sanitized.Stats.CurrentPlayers,
						"maxclients":         gs.Sanitized.Stats.MaximumPlayers,
						"protocol":           gs.Stats["protocol"],
						"game":               game_mapping[gs.MasterServer],
						"gametype":           gs.Stats["g_gametype"],
						"version":            version,
						"map_preview":        map_preview,
						"stats":              gs.Stats,
						"players":            gs.Sanitized.Players,
						"players_colorized":  players_colorized,
						"passwordprotected":  gs.Sanitized.Stats.PasswordProtected,
					})
					mu.Unlock()

				}()

			}

			wg.Wait()

			sort.Slice(template_gameservers.Gameservers, func(i, j int) bool {
				return template_gameservers.Gameservers[i]["curclients"].(int) > template_gameservers.Gameservers[j]["curclients"].(int)
			})

			template_gameservers.TotalGameservers = len(template_gameservers.Gameservers)

			// if len(msg.(map[string]*server.Gameserver)) != template_gameservers.TotalGameservers {
			// 	for ipport := range msg.(map[string]*server.Gameserver) {
			// 		found := false

			// 		for i := range template_gameservers.Gameservers {
			// 			if ipport == template_gameservers.Gameservers[i]["ipport"] {
			// 				found = true
			// 			}
			// 		}

			// 		if !found {
			// 			log.Println("Missing: ", ipport)
			// 		}
			// 	}

			// }

			templmu.Lock()
			templ = template_gameservers
			templmu.Unlock()

			log.Println("Server browser template updated.")

		default:
			log.Println("unknown message: ", msg)
		}

	}
}
