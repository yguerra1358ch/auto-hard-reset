package main

import (
	"os/exec"
	"strings"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"gobot.io/x/gobot/drivers/gpio"
	"fmt"
)

//Rig structure
type Rig struct {
	name string
	pin  *gpio.RelayDriver
	ip   string
	info string
	rigos   string
	lastCheckTime string
	lastCheckResult string
}

//CheckMachines - ping machines, if there is no responce >> hard-reset
func CheckMachines(r []Rig) {
	var rigsDown []int
	log.Notice("Checking machines: ")

	for i := 0; i < len(r); i++ {
		log.Notice("Ping/TCPPing machine: ", r[i].name, "ip: ", r[i].ip)
		//if !r[i].Ping() {
		if !r[i].TCPPing() {
			if r[i].rigos == "linux" {
				log.Notice("Restart here. Linux is probably frozen")
				r[i].Restarter()
				rigsDown = append(rigsDown,i)
			} else if r[i].rigos == "windows" {
				log.Notice("Turn On here. Windows is probably off")
				r[i].TurnOn()
				rigsDown = append(rigsDown,i)
			} else {
				log.Notice(fmt.Sprintf("Unknown OS Type: %s",r[i].rigos))
			}
		}
	}

	log.Notice("Checking machines DONE\n----------------------")
	if len(rigsDown) > 0 {
		log.Notice("Waiting 2 minutes for down machines to come up before recheck")
		time.Sleep(120 * time.Second)
		for i :=0; i <len(rigsDown); i++ {
			log.Notice(fmt.Sprintf("Checking %s - %s",r[rigsDown[i]].name,r[rigsDown[i]].ip))
			if !r[rigsDown[i]].TCPPing() {
				log.Notice(fmt.Sprintf("Rebooting %s IP: %s",r[rigsDown[i]].name,r[rigsDown[i]].ip))
				r[rigsDown[i]].Restarter()
			}
		}
	}
		
	log.Notice("Starting timer")
}

//Ping IP from Linux shell
func (r *Rig) Ping() bool {
	out, _ := exec.Command("ping", r.ip, "-c 3", "-i 3", "-w 10").Output()
	if strings.Contains(string(out), "100% packet loss") {
		log.Error("HOST NOT FOUND: ", r.name, r.ip)
		return false
	}

	log.Notice("HOST IS ONLINE: ", r.name)
	return true
}

func (r *Rig) TCPPing() bool {
	port := "9630"
	if r.rigos == "linux" {
		port = "22"
	}
	matchString := "len=46"
        
	out, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("sudo hping3 -c 1 -S -p %s %s",port,r.ip)).Output()

	if err != nil {
		log.Notice(fmt.Sprintf("Error: %s ",err))
	}
	if strings.Contains(string(out), matchString) {
        	log.Notice(fmt.Sprintf("Port: %s Passed",port))
        	return true
        }
       	log.Notice(fmt.Sprintf("Port: %s FAILED",port))
        return false
}



//ForceShutDown machine
func (r *Rig) ForceShutDown() {
	r.pin.Off()
	time.Sleep(5 * time.Second)
	r.pin.On()
}

//TurnOn machine
func (r *Rig) TurnOn() {
	r.pin.Off()
	time.Sleep(108 * time.Millisecond)
	r.pin.On()
}

//Restarter function logic
func (r *Rig) Restarter() {
	log.Warning("Restarting: ", r.name)

	if Config.RemoteNotify {
		if Config.Pushover {
			PushoverNotify(r)
		}

		if Config.TgBotActivate {
			//TODO: telegram notification
			tgbotapi.NewMessageToChannel(Config.TgAdminUserName, "TEST MSG")
		}
	}

	r.ForceShutDown()
	time.Sleep(5 * time.Second)
	r.TurnOn()

	log.Warning("Machine restarted: ", r.name)
}
