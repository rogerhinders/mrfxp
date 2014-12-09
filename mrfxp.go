package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"fmt"
	"github.com/rvah/mrfxp/config"
	_ "github.com/rvah/mrfxp/fxp"
	"io/ioutil"
	"net/http"
)

const RES_PATH = "res/"

type SiteListMsg struct {
	Event string
	Sites []config.SiteConfig
}

type SettingsDataMsg struct {
	Message string
	//	Sections []config.Sections
}

type FetchMsg struct {
	Message string
}

type SocketMsg struct {
	Event     string
	Reference string
	Data      interface{}
}

func fetchHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var t FetchMsg

	err := decoder.Decode(&t)

	if err != nil {
		fmt.Println(err)
		return
	}

	encoder := json.NewEncoder(w)

	switch t.Message {
	case "SitesData":
	case "SettingsData":
		encoder.Encode(SettingsDataMsg{Message: "lala"})
	}
}

func indexHandler(w http.ResponseWriter, req *http.Request) {
	r, err := loadResource("index.html")

	if err != nil {
		fmt.Fprint(w, "error loading resource!")
		return
	}
	fmt.Fprint(w, r)
}

func jsHandler(w http.ResponseWriter, req *http.Request) {
	r, err := loadResource("mrfxp.js")

	if err != nil {
		fmt.Fprint(w, "error loading resource!")
		return
	}
	fmt.Fprint(w, r)
}

func socketHandler(ws *websocket.Conn) {
	fmt.Println("client called :)")

	//send initial data to setup UI
	conf := new(config.Config)
	err := conf.Init()

	if err != nil {
		fmt.Println(err)
		return
	}
	/*else {
		sites, _ := conf.GetSites()
		siteMsg := new(SiteListMsg)
		siteMsg.Event = "init-sitelist"
		siteMsg.Sites = sites
		err := websocket.JSON.Send(ws, siteMsg)

		if err != nil {
			fmt.Println(err)
		}
	}*/

	//var toSend string
	for {
		/*
			fmt.Print("Enter msg: ")
			fmt.Scanf("%s", &toSend)
			err := websocket.Message.Send(ws, fmt.Sprintf("Server says: %s", toSend))

			if err != nil {
				fmt.Println("error sending msg to client :(")
			}
		*/

		var msg SocketMsg

		err = websocket.JSON.Receive(ws, &msg)
		//err = websocket.Message.Receive(ws, &msg)

		if err != nil {
			fmt.Println(err)
			return
		}
		dmap := msg.Data.(map[string]interface{})
		switch msg.Event {
		case "AddSection":
			conf.AddSection(dmap["Name"].(string))
			sections, _ := conf.GetSections()
			msg.Event = "SetSections"
			msg.Reference = msg.Reference
			msg.Data = sections

			err = websocket.JSON.Send(ws, msg)
			if err != nil {
				fmt.Println(err)
			}
		case "GetSections":
			sections, _ := conf.GetSections()
			msg.Event = "SetSections"
			msg.Reference = msg.Reference
			msg.Data = sections

			err = websocket.JSON.Send(ws, msg)
			if err != nil {
				fmt.Println(err)
			}
		case "GetSites":
			sites, _ := conf.GetSites()
			msg.Event = "SetSites"
			msg.Reference = msg.Reference
			msg.Data = sites

			//remove password for security
			for i := 0; i < len(sites); i++ {
				sites[i].Password = "***"
			}

			//send it
			err = websocket.JSON.Send(ws, msg)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func main() {
	config.RegDBDriver()

	http.Handle("/mrfxp", http.HandlerFunc(indexHandler))
	http.Handle("/mrfxp/mrfxp.js", http.HandlerFunc(jsHandler))
	http.Handle("/mrfxp/ws", websocket.Handler(socketHandler))

	err := http.ListenAndServe(":8888", nil)

	if err != nil {
		fmt.Println(err)
		return
	}
}

func loadResource(name string) (string, error) {
	f, err := ioutil.ReadFile(fmt.Sprintf("%s%s", RES_PATH, name))

	if err != nil {
		return "", err
	}

	return string(f), nil
}
