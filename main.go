package main

import (
	"time"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/gpio"
	"gobot.io/x/gobot/platforms/raspi"
	
	"fmt"
	"net/http"
	"bytes"
)

var miningRigs = make([]Rig, 0)

func serveWebContent(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	formLen := len(r.Form)
	if formLen > 0 {
		log.Notice("WebServer: Processing POST")
		targetRigName := r.PostFormValue("rigName")
		var targetRig Rig
        	for _, rig := range miningRigs {
			if rig.name == targetRigName {
				targetRig = rig
			}
        	}

		if r.PostFormValue("softboot") != "" {
			targetRig.TurnOn()	
		} else {
			//hard boot
			targetRig.Restarter()
		}
		fmt.Fprintf(w, "<html><meta http-equiv='refresh' content='5; url=/' />Rebooting rig. Going back to <a href='/'>Main Page</a> in 10 seconds</html>")
	} else {
		log.Notice("Web Server: Sent Default Page")
		var htmlBuf bytes.Buffer
		htmlBuf.WriteString("<html><table border='1'><tr><th>Name</th><th>OS</th><th>Last Check Time</th><th>Last Check Result</th>")
		for _, curRig := range miningRigs {
			htmlBuf.WriteString(fmt.Sprintf("<form method='post'><tr><td>%s </td><td>%s </td><td>%s </td><td>%s </td><input type='hidden' name='rigName' value='%s'><td><input type='submit' name='softboot' value='Soft Reboot'></td><td><input type='submit' name='hardboot' value='Hard Reboot'></td></form></tr>",curRig.name,curRig.rigos,"Last Check ","Check Result",curRig.name))
		}
		htmlBuf.WriteString("</table></html>")

		fmt.Fprintf(w,htmlBuf.String())
	}
}

func initWebServer(webPort int) {
	http.HandleFunc("/", serveWebContent)
	log.Notice("Starting Web Listener on port %d",webPort)
	err := http.ListenAndServe(fmt.Sprintf(":%d",webPort), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func main() {
	r := raspi.NewAdaptor()

	//Read configuration file
	Config = ReadConfig()
	
	webPort := 9090
	if Config.WebPort != 0 {
		webPort = Config.WebPort
	}

	//parse machines to []Rig struct
	for _, m := range Config.Miners {
		log.Notice("minerConfig:", m)
		miningRigs = append(miningRigs, Rig{m.Name, gpio.NewRelayDriver(r, m.Pin), m.IP, m.Info,m.OS,"",""})
	}

	log.Notice("Configured rigs: ", len(miningRigs))

	//Logging machines in two outputs - console & external file
	if Config.Log {
		go LogMachines()
	}

	if Config.TgBotActivate {
		go TelegramBot(miningRigs)
	}
	//Gobot work func
	work := func() {
		log.Notice("HELLO! I WILL KEEP YOUR MINING RIGS ONLINE!")

		//Check machines on startup without waiting the timer. Use with caution. After a power failure, RPI could be ready faster than your machines and start restarting them without need.
		if Config.StartupCheck {
			CheckMachines(miningRigs)
		}

		timer := time.Duration(Config.WaitSeconds) * time.Second
		log.Notice("Starting timer: ", timer)

		//Check the machines periodically
		gobot.Every(timer, func() {
			CheckMachines(miningRigs)
		})
	}

	robot := gobot.NewRobot("auto-hard-reset", r, work)
	for _, rig := range miningRigs {
		robot.AddDevice(rig.pin)
	}

	go initWebServer(webPort)
	robot.Start()
}
