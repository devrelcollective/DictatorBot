package main

import (
	// "context"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	camundaclientgo "github.com/citilinkru/camunda-client-go/v2"
	"github.com/citilinkru/camunda-client-go/v2/processor"
	// "github.com/robfig/cron/v3"
	"gopkg.in/yaml.v2"
)

// DictatorPayload is the incoming payload from Slack
type DictatorPayload struct {
	Token               string `json:"token"`
	TeamID              string `json:"team_id"`
	TeamDomain          string `json:"team_domain"`
	ChannelID           string `json:"channel_id"`
	ChannelName         string `json:"channel_name"`
	UserID              string `json:"user_id"`
	UserName            string `json:"user_name"`
	Command             string `json:"text"`
	APIAppID            string `json:"api_app_id"`
	IsEnterpriseInstall string `json:"is_enterprise_install"`
	ResponseURL         string `json:"response_url"`
	TriggerID           string `json:"trigger_id"`
}

var config DictatorConfig
// DictatorConfig is the entire configuration for the Bot
type DictatorConfig struct {
	DictatorVersion int `yaml:"Dictator_version"`
	CamundaHost     []struct {
		Name     string `yaml:"name"`
		Port     int    `yaml:"port"`
		Protocol string `yaml:"protocol"`
		Host     string `yaml:"host"`
	} `yaml:"Camunda Host"`
	SlackListener []struct {
		Name     string `yaml:"name"`
		Port     int    `yaml:"port"`
		Protocol string `yaml:"protocol"`
		Host     string `yaml:"host"`
	} `yaml:"Slack Listener"`
	AuthorizedSenders []struct {
		Name     string `yaml:"name"`
		Username string `yaml:"username"`
		IsOnCall bool   `yaml:"is-on-call"`
		Order    int    `yaml:"order,omitempty"`
	} `yaml:"Authorized senders"`
	CurrentOnCall string `yaml:"Current On Call"`
	OnCallIndex   int    `yaml:"On Call Index"`
	SlackSecret   string `yaml:"Slack Secret"`
	TotalOnCall   int    `yaml:"Total On Call"`
	ResponseURL   string `yaml:"Response Url"`
	ResponseToken string `yaml:"Response Token"`
	AppID         string `yaml:"AppID"`
	ChannelID     string `yaml:"Channel ID"`
}

// WriteDictators outputs the entire config file
func WriteDictator() {
	taters, err := yaml.Marshal(config)
	err = ioutil.WriteFile("./dictator.yaml", taters, 0644)
	if err != nil {
		panic(err)
	}
}

// init_dictators reads the config file and sets up the config struct
func init_dictator(){
	dat, err := ioutil.ReadFile("./dictator.yaml")
	if err != nil {
		log.Fatal("No startup file: ", err)
	}
	err = yaml.Unmarshal(dat, &config)
	if err != nil {
		log.Fatal(err)
	}
}

// these are all the dictators that can run this command
func getDictators() []string { // test written
	var taters []string = make([]string, len(config.AuthorizedSenders))
	for x := 0; x < len(config.AuthorizedSenders); x++ {
		taters[x] = config.AuthorizedSenders[x].Username
	}
	return taters
}

func SendDirect(msg_type string) bool { //test written
	if msg_type == "directmessage" {
		return true
	}
	return false
}

func getDictatorString() string { // test written
	var retValue strings.Builder
	retValue.WriteString("*Currently Authorized Benevolent Dictators are:*\n")
	for x := 0; x < len(config.AuthorizedSenders); x++ {
		retValue.WriteString("• " + config.AuthorizedSenders[x].Name + " (" + config.AuthorizedSenders[x].Username + ")\n")
	}
	return retValue.String()
}

// this is the rotation order
func getRotation() []string { // test written
	var taters []string = make([]string, config.TotalOnCall)
	for x := 0; x < len(config.AuthorizedSenders); x++ {
		if config.AuthorizedSenders[x].IsOnCall {
			taters[config.AuthorizedSenders[x].Order-1] = config.AuthorizedSenders[x].Username
		}
	}
	return taters
}

func getRotationString() string { // test written
	var retValue strings.Builder
	taters := getRotation()
	for x := 0; x < len(taters); x++ {
		retValue.WriteString(taters[x])
		retValue.WriteString("-->")
	}
	return strings.TrimRight(retValue.String(), "-->")
}

// this returns who is on-call now from the rotation
func getOnCall() string { // test written
	return getRotation()[config.OnCallIndex]
}

func getNextOnCall() string { // test written
	if config.OnCallIndex > len(getRotation()) {
		return getRotation()[0]
	}
	return getRotation()[config.OnCallIndex+1]
}

// this returns the index of who is on-call
func getOnCallIndex(newTater string) int { //Test Written
	oc := getRotation()
	for x := 0; x < len(oc); x++ {
		if oc[x] == newTater {
			return x
		}
	}
	return config.OnCallIndex
}

// rotates the on-call person index. Returns new on-call person
func rotateOnCallIndex(newTater int) string { // test written
	if newTater >= len(getRotation()) {
		config.OnCallIndex = 0
	} else {
		config.OnCallIndex = newTater
	}
	config.CurrentOnCall = getRotation()[config.OnCallIndex]
	return getRotation()[config.OnCallIndex]
}

// rotates the on-call person based on the new name
func rotateOnCall(newTater string) string { // test written
	ind := getOnCallIndex(newTater)
	config.OnCallIndex = ind
	return rotateOnCallIndex(getOnCallIndex(newTater))
}

func checkHeader(key string, data string) bool { // Test Written
	// Create a new HMAC by defining the hash type and the key (as byte array)
	h := hmac.New(sha256.New, []byte(config.SlackSecret))
	// Write Data to it
	h.Write([]byte(data))
	// Get result and encode as hexadecimal string
	sha := hex.EncodeToString(h.Sum(nil))
	comp := fmt.Sprintf("v0=%s", sha)
	return comp == key
}

func validateDictator(newVars map[string]camundaclientgo.Variable, contx *processor.Context) error {
	// fmt.Printf("Running task %s. WorkerId: %s. TopicName: %s\n", contx.Task.Id, contx.Task.WorkerId, contx.Task.TopicName)
	varb := contx.Task.Variables
	// cmd, err := url.QueryUnescape(fmt.Sprintf("%v", newVars["command"].Value))
	// if err != nil {
	// 	WriteDictator()
	// 	log.Fatal(err)
	// 	return err
	// }
	// fmt.Println("validate_dictator Command:", cmd)
	// fmt.Println("Sender: ", varb["sender"].Value)
	senderOk := isValueInList(fmt.Sprintf("%v", varb["sender"].Value), getDictators())
	if varb["sender"].Value == "dictatorbot" {
		senderOk = true
	}
	vars := make(map[string]camundaclientgo.Variable)
	//stat := camundaclientgo.Variable{Value: "true", Type: "boolean"}
	//com :=
	vars["senderOk"] = camundaclientgo.Variable{Value: senderOk, Type: "boolean"}
	vars["status"] = camundaclientgo.Variable{Value: "true", Type: "boolean"}
	if !senderOk {
		vars["message_type"] = camundaclientgo.Variable{Value: "failure", Type: "string"}
	} else {
		vars["message_type"] = camundaclientgo.Variable{Value: "success", Type: "string"}
	}
	err := contx.Complete(processor.QueryComplete{
		Variables: &vars,
	})
	if err != nil {
		WriteDictator()
		return err
		// fmt.Printf("Error set complete task %s: %s\n", ctx.Task.Id, err)
	}
	return nil
}

func startCamundaProcess(data DictatorPayload) {

	opts := camundaclientgo.ClientOptions{}
	opts.ApiPassword = "demo"
	opts.ApiUser = "demo"
	opts.EndpointUrl = config.CamundaHost[0].Protocol + "://" + config.CamundaHost[0].Host + ":" + fmt.Sprint(config.CamundaHost[0].Port) + "/engine-rest"

	// 	camundaclientgo.ClientOptions{
	// 	// this should use values from the config file
	// 	EndpointUrl: config.CamundaHost[0].Protocol + "://" + config.CamundaHost[0].Host + ":" + fmt.Sprint(config.CamundaHost[0].Port) + "/engine-rest", //"https://davidgs.com:8443/engine-rest",
	// 	ApiUser:     "demo",
	// 	ApiPassword: "demo",
	// 	Timeout:     time.Second * 10,
	// })
	messageName := "Query_dictator"
	processKey := "DictatorBot"
	variables := map[string]camundaclientgo.Variable{
		"command":      {Value: strings.TrimSpace(data.Command), Type: "string"},
		"sender":       {Value: data.UserName, Type: "string"},
		"token":        {Value: data.Token, Type: "string"},
		"channel_id":   {Value: data.ChannelID, Type: "string"},
		"channel_name": {Value: data.ChannelName, Type: "string"},
		"response_url": {Value: data.ResponseURL, Type: "string"},
		"user_id":      {Value: data.UserID, Type: "string"},
		"api_app_id":   {Value: data.APIAppID, Type: "string"},
	}
	client := camundaclientgo.NewClient(opts)
	reqMessage := camundaclientgo.ReqMessage{}
	reqMessage.MessageName = messageName
	reqMessage.BusinessKey = processKey
	reqMessage.ProcessVariables = &variables
	err := client.Message.SendMessage(&reqMessage)

	// _, err := client.ProcessDefinition.StartInstance(
	// 	camundaclientgo.QueryProcessDefinitionBy{Key: &processKey},
	// 	camundaclientgo.ReqStartInstance{Variables: &variables},
	// )
	if err != nil {
		log.Printf("Error starting process: %s\n", err)
		return
	}
}

// Handles all the incomming https requests
func dictator(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// log.Println("GET Method Not Supported")
		http.Error(w, "GET Method not supported", 400)
	} else {
		key := r.Header.Get("X-Slack-Signature")

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		timestamp := r.Header.Get("X-Slack-Request-Timestamp")
		step1 := strings.ReplaceAll(string(body), "&", "\", \"")
		step2 := strings.ReplaceAll(step1, "=", "\": \"")
		step1 = fmt.Sprintf("{\"%s\"}", step2)
		var t DictatorPayload
		err = json.Unmarshal([]byte(step1), &t)
		if err != nil {
			panic(err)
		}
		signedData := fmt.Sprintf("v0:%s:%s", timestamp, string(body))
		if err != nil {
			WriteDictator()
			log.Fatal(err)
		}
		if !checkHeader(key, signedData) {
			w.WriteHeader(400)
			return
		}
		// log.Println(t.Command)
		w.WriteHeader(200)
		startCamundaProcess(t)
	}
}

// is a value in the array?
func isValueInList(value string, list []string) bool { // Test Written
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

func RunEverySecond() {
	var data DictatorPayload = DictatorPayload{}
	data.Command = "update"
	data.UserID = "dictatorbot"
	data.UserName = "dictatorbot"
	data.ChannelID = config.ChannelID
	data.Token = config.ResponseToken
	data.ChannelID = config.ChannelID
	data.ResponseURL = config.ResponseURL
	data.APIAppID = config.AppID

	startCamundaProcess(data)
	// fmt.Println("Every minute")
}
func main() {

	init_dictator()
	fmt.Println("Starting up ... ")
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		WriteDictator()
		os.Exit(1)
	}()
	// cro := cron.New()
	// cro.AddFunc("0 12 * * MON", func() {
	// 	RunEverySecond()
	// })
	// cro.Start()

	client := camundaclientgo.NewClient(camundaclientgo.ClientOptions{
		EndpointUrl: config.CamundaHost[0].Protocol + "://" + config.CamundaHost[0].Host + ":" + fmt.Sprint(config.CamundaHost[0].Port) + "/engine-rest",
		// ApiUser:     "demo",
		// ApiPassword: "demo",
		// RESET to 10
		Timeout: time.Second * 10,
	})
	logger := func(err error) {
		fmt.Println(err.Error())
	}
	proc := processor.NewProcessor(client, &processor.ProcessorOptions{
		WorkerId:                  "dictatorBot",
		LockDuration:              time.Second * 5,
		MaxTasks:                  10,
		MaxParallelTaskPerHandler: 100,
		LongPollingTimeout:        5 * time.Second,
	}, logger)

	proc.AddHandler( // validate proper sender validate_dictator
		&[]camundaclientgo.QueryFetchAndLockTopic{
			{TopicName: "validate_dictator"},
		},
		func(ctx *processor.Context) error {
			return validateDictator(ctx.Task.Variables, ctx)
		},
	)

	proc.AddHandler( // get authorized users sender get_auth
		&[]camundaclientgo.QueryFetchAndLockTopic{
			{TopicName: "get_auth"},
		},
		func(ctx *processor.Context) error {
			//	fmt.Printf("Running task %s. WorkerId: %s. TopicName: %s\n", ctx.Task.Id, ctx.Task.WorkerId, ctx.Task.TopicName)

			vars := make(map[string]camundaclientgo.Variable)
			vars["message_type"] = camundaclientgo.Variable{Value: "success", Type: "string"}
			vars["message"] = camundaclientgo.Variable{Value: getDictatorString(), Type: "string"}
			err := ctx.Complete(processor.QueryComplete{
				Variables: &vars,
			})
			if err != nil {
				// fmt.Printf("Error set complete task %s: %s\n", ctx.Task.Id, err)
			}
			return nil
		},
	)

	proc.AddHandler( // see who is on call whos_oncall
		&[]camundaclientgo.QueryFetchAndLockTopic{
			{TopicName: "whos_oncall"},
		},
		func(ctx *processor.Context) error {
			// fmt.Printf("Running task %s. WorkerId: %s. TopicName: %s\n", ctx.Task.Id, ctx.Task.WorkerId, ctx.Task.TopicName)
			var err error
			vars := make(map[string]camundaclientgo.Variable)
			body := fmt.Sprintf("<@%s> is the person on-call this week.", getOnCall())
			vars["message_type"] = camundaclientgo.Variable{Value: "success", Type: "string"}
			vars["on-call"] = camundaclientgo.Variable{Value: getOnCall(), Type: "string"}
			vars["message"] = camundaclientgo.Variable{Value: body, Type: "string"}

			err = ctx.Complete(processor.QueryComplete{
				Variables: &vars,
			})

			if err != nil {
				// fmt.Printf("Error set complete task %s: %s\n", ctx.Task.Id, err)
			}

			return nil
		},
	)

	proc.AddHandler( // get the entire rotation scheme get_rotation
		&[]camundaclientgo.QueryFetchAndLockTopic{
			{TopicName: "get_rotation"},
		},
		func(ctx *processor.Context) error {
			// fmt.Printf("Running task %s. WorkerId: %s. TopicName: %s\n", ctx.Task.Id, ctx.Task.WorkerId, ctx.Task.TopicName)
			var err error
			msg := fmt.Sprintf("The on-call rotation schedule is:\n *%s* \nand *%s* is the on-call Dictator", getRotationString(), getOnCall())
			vars := make(map[string]camundaclientgo.Variable)

			vars["message"] = camundaclientgo.Variable{Value: msg, Type: "string"}
			vars["message_type"] = camundaclientgo.Variable{Value: "success", Type: "string"}
			err = ctx.Complete(processor.QueryComplete{
				Variables: &vars,
			})

			if err != nil {
				// fmt.Printf("Error set complete task %s: %s\n", ctx.Task.Id, err)
			}

			return nil
		},
	)

	proc.AddHandler( // get who is next in the rotation get_next
		&[]camundaclientgo.QueryFetchAndLockTopic{
			{TopicName: "get_next"},
		},
		func(ctx *processor.Context) error {
			// fmt.Printf("Running task %s. WorkerId: %s. TopicName: %s\n", ctx.Task.Id, ctx.Task.WorkerId, ctx.Task.TopicName)
			var err error
			vars := make(map[string]camundaclientgo.Variable)
			msg := fmt.Sprintf("The next person on-call is <@%s>", getNextOnCall())
			vars["message"] = camundaclientgo.Variable{Value: msg, Type: "string"}
			vars["message_type"] = camundaclientgo.Variable{Value: "success", Type: "string"}
			err = ctx.Complete(processor.QueryComplete{
				Variables: &vars,
			})
			if err != nil {
				// fmt.Printf("Error set complete task %s: %s\n", ctx.Task.Id, err)
			}
			return nil
		},
	)

	proc.AddHandler( // make sure that the updated dictator is allowed check_new_oncall
		&[]camundaclientgo.QueryFetchAndLockTopic{
			{TopicName: "check_new_oncall"},
		},
		func(ctx *processor.Context) error {
			// fmt.Printf("Running task %s. WorkerId: %s. TopicName: %s\n", ctx.Task.Id, ctx.Task.WorkerId, ctx.Task.TopicName)
			var err error
			varb := ctx.Task.Variables
			text := fmt.Sprintf("%v", varb["command"].Value)
			text, err = url.QueryUnescape(text)
			if err != nil {
				WriteDictator()
				log.Fatal(err)
			}

			newTater := strings.TrimLeft(text, "@")
			vars := make(map[string]camundaclientgo.Variable)
			if !isValueInList(newTater, getRotation()) {
				//if getOnCallIndex(config, newTater) < 0 {
				vars["onCallOK"] = camundaclientgo.Variable{Value: "false", Type: "boolean"}
				vars["message_type"] = camundaclientgo.Variable{Value: "failure", Type: "string"}
			} else {
				vars["onCallOK"] = camundaclientgo.Variable{Value: "true", Type: "boolean"}
				vars["message_type"] = camundaclientgo.Variable{Value: "success", Type: "string"}
			}
			if newTater == "update" {
				thisTater := getOnCallIndex(getOnCall())
				rotateOnCallIndex(thisTater+1)
				vars["onCallOK"] = camundaclientgo.Variable{Value: "true", Type: "boolean"}
				vars["message_type"] = camundaclientgo.Variable{Value: "success", Type: "string"}
			}
			err = ctx.Complete(processor.QueryComplete{
				Variables: &vars,
			})

			if err != nil {
				// fmt.Printf("Error set complete task %s: %s\n", ctx.Task.Id, err)
			}

			// fmt.Printf("Task %s completed\nTask Command: %s\nTask Result: %s", ctx.Task.Id, text, getOnCall())
			return nil
		},
	)

	proc.AddHandler( // set the new on-call dictator update_oncall
		&[]camundaclientgo.QueryFetchAndLockTopic{
			{TopicName: "update_oncall"},
		},
		func(ctx *processor.Context) error {
			// fmt.Printf("Running task %s. WorkerId: %s. TopicName: %s\n", ctx.Task.Id, ctx.Task.WorkerId, ctx.Task.TopicName)
			var err error
			varb := ctx.Task.Variables
			var timer string
			if varb["command"].Value == nil {
				timer = "timer"
			}
			if timer == "timer" {
				// fmt.Println("Timer event fired!")
				vars := make(map[string]camundaclientgo.Variable)
				lastTater := getOnCall()
				if config.OnCallIndex+1 >= len(getRotation()) {
					rotateOnCallIndex(0)
				} else {
					rotateOnCallIndex(config.OnCallIndex+1)
				}
				thisTater := getOnCall()
				msg := fmt.Sprintf("<@%s> has been relieved of duty and <@%s> is now the on-call person", lastTater, thisTater)
				vars["message"] = camundaclientgo.Variable{Value: msg, Type: "string"}
				vars["message_type"] = camundaclientgo.Variable{Value: "success", Type: "string"}
				vars["command"] = camundaclientgo.Variable{Value: "update", Type: "string"}
				vars["senderOk"] = camundaclientgo.Variable{Value: "true", Type: "boolean"}
				vars["onCallOK"] = camundaclientgo.Variable{Value: "true", Type: "boolean"}
				fmt.Println(msg)
				err = ctx.Complete(processor.QueryComplete{
					Variables: &vars,
				})
				if err != nil {
					// fmt.Printf("Error set complete task %s: %s\n", ctx.Task.Id, err)
				}
				// fmt.Printf("Task %s completed\nTask Command: %s\nTask Result: %s", ctx.Task.Id, "Timer Event", getOnCall())
				return nil
			} else {
				text := fmt.Sprintf("%v", varb["command"].Value)
							fmt.Printf("Get Auth: %v\n", varb["command"].Value)

				newTater, err := url.QueryUnescape(text)
				if err != nil {
					WriteDictator()
					log.Fatal(err)
				}
				newTater = strings.TrimLeft(newTater, "@")
				newTater = rotateOnCall(newTater)
				vars := make(map[string]camundaclientgo.Variable)
				lastTater := fmt.Sprintf("%v", varb["sender"].Value)
				msg := fmt.Sprintf("<@%s> has made a change and <@%s> is now the on-call person", lastTater, newTater)
				vars["message"] = camundaclientgo.Variable{Value: msg, Type: "string"}
				vars["message_type"] = camundaclientgo.Variable{Value: "success", Type: "string"}
				err = ctx.Complete(processor.QueryComplete{
					Variables: &vars,
				})
				if err != nil {
					// fmt.Printf("Error set complete task %s: %s\n", ctx.Task.Id, err)
				}
				// fmt.Printf("Task %s completed\nTask Command: %s\nTask Result: %s\n", ctx.Task.Id, text, getOnCall())
				return nil
			}
		},
	)

	proc.AddHandler( // format a message format_message
		&[]camundaclientgo.QueryFetchAndLockTopic{
			{TopicName: "format_message"},
		},
		func(ctx *processor.Context) error {
			// fmt.Printf("Running task %s. WorkerId: %s. TopicName: %s\n", ctx.Task.Id, ctx.Task.WorkerId, ctx.Task.TopicName)
			varb := ctx.Task.Variables
			var body string
			vars := make(map[string]camundaclientgo.Variable)
			if varb["senderOk"].Value != nil {
				if varb["senderOk"].Value == "false" {
					body = ":X: Nice try, but only Benevolent Dictators (Moderators) may use this bot."
					vars["message_type"] = camundaclientgo.Variable{Value: "failure", Type: "string"}
					vars["message"] = camundaclientgo.Variable{Value: body, Type: "string"}
				}
				if varb["onCallOK"].Value != nil {
					if varb["onCallOK"].Value == false {
						comm := fmt.Sprintf("%v", varb["command"].Value)
						comm, err := url.QueryUnescape(comm)
						if err != nil {
							WriteDictator()
							log.Fatal(err)
						}
						comm = strings.TrimLeft(comm, "@")
						body = fmt.Sprintf("You cannot nominate <@%s> because they are not a Benevolent Dictator!", comm)
						vars["message_type"] = camundaclientgo.Variable{Value: "failure", Type: "string"}
						vars["message"] = camundaclientgo.Variable{Value: body, Type: "string"}
					}
				}
			} else {
				body = fmt.Sprintf("%v", varb["message"])
				vars["message_type"] = camundaclientgo.Variable{Value: "success", Type: "string"}
			}
			err := ctx.Complete(processor.QueryComplete{
				Variables: &vars,
			})
			if err != nil {
				// fmt.Printf("Error set complete task %s: %s\n", ctx.Task.Id, err)
			}
			// fmt.Printf("Task %s completed\nTask Command: %s\nTask Result: %s", ctx.Task.Id, text, getOnCall())
			return nil
		},
	)

	proc.AddHandler( // send the message send_message
		&[]camundaclientgo.QueryFetchAndLockTopic{
			{TopicName: "send_message"},
		},
		func(ctx *processor.Context) error {
			//	fmt.Printf("Running task %s. WorkerId: %s. TopicName: %s\n", ctx.Task.Id, ctx.Task.WorkerId, ctx.Task.TopicName)
			varb := ctx.Task.Variables

			msg_type := fmt.Sprintf("%v", varb["message_type"].Value)
			var final_msg string
			if msg_type == "failure" {
				final_msg = fmt.Sprintf(":X: %v %s", varb["message"].Value, ":unamused:")
			} else {
				final_msg = fmt.Sprintf(":white_check_mark: %v ", varb["message"].Value)
			}
			var err error
			reply_url := fmt.Sprintf("%v", varb["response_url"].Value)
			response_type := fmt.Sprintf("%v", varb["channel_name"].Value)
			if varb["channel_name"].Value == nil {
				response_type = "channel"
			}
			var channel_id string
			if varb["channel_id"].Value == nil {
				channel_id = "C01TA9C0FJL" // "G0A7K9GPN"
			} else {
				channel_id = fmt.Sprintf("%v", varb["channel_id"].Value)
			}
			command := fmt.Sprintf("%v", varb["command"].Value)
			command, err = url.QueryUnescape(command)
			if err != nil {
				WriteDictator()
				log.Fatal(err)
			}
			if SendDirect(response_type) {
				reply_url, err = url.QueryUnescape(reply_url)
				if err != nil {
					WriteDictator()
					log.Fatal(err)
				}
				reqBody, err := json.Marshal(map[string]string{
					"response_type":    "message",
					"replace_original": "false",
					"text":             final_msg,
				})
				if err != nil {
					WriteDictator()
					log.Fatal(err)
				}
				resp, err := http.Post(reply_url, "application/json", bytes.NewBuffer(reqBody))
				if err != nil {
					WriteDictator()
					log.Fatal(err)
				}
				defer resp.Body.Close()
			} else {
				reply_url = config.ResponseURL + "?token=" + config.ResponseToken + "&channel=" + channel_id + "&text=" + url.QueryEscape(final_msg)
				resp, err := http.Get(reply_url)
				if err != nil {
					WriteDictator()
					log.Fatal(err)
				}
				defer resp.Body.Close()
			}
			vars := make(map[string]camundaclientgo.Variable)
			vars["complete"] = camundaclientgo.Variable{Value: "true", Type: "boolean"}
			err = ctx.Complete(processor.QueryComplete{
				Variables: &vars,
			})
			if err != nil {
				//	fmt.Printf("Error set complete task %s: %s\n", ctx.Task.Id, err)
			}
			return nil
		},
	)

	proc.AddHandler( // send the help/usage message get_help
		&[]camundaclientgo.QueryFetchAndLockTopic{
			{TopicName: "get_help"},
		},
		func(ctx *processor.Context) error {
			//	fmt.Printf("Running task %s. WorkerId: %s. TopicName: %s\n", ctx.Task.Id, ctx.Task.WorkerId, ctx.Task.TopicName)
			var err error
			vars := make(map[string]camundaclientgo.Variable)
			body := ` *Available commands are:*
			1) _help_ or _?_
			2) _rotate_ or _rotation_ to get the full rotation schedule
			3) _who_ to see who the current on-call person is
			4) _next_ to see who the next on-call person will be
			5) _@username_ to place someone on-call
			6) _auth_ or _authorized_ to see who is authorized to use the DictatorBot
			7) _update_ to place the next person in the rotation on-call`
			vars["message_type"] = camundaclientgo.Variable{Value: "success", Type: "string"}
			vars["message"] = camundaclientgo.Variable{Value: body, Type: "string"}
			vars["onCallOK"] = camundaclientgo.Variable{Value: "true", Type: "boolean"}

			err = ctx.Complete(processor.QueryComplete{
				Variables: &vars,
			})

			if err != nil {
				fmt.Printf("Error set complete task %s: %s\n", ctx.Task.Id, err)
			}

			//fmt.Printf("Task %s completed\nTask Command: %s\nTask Result: %s", ctx.Task.Id, text, getOnCall())
			return nil
		},
	)
	http.HandleFunc("/dictator", dictator)

	if config.SlackListener[0].Protocol == "https" {
		err := http.ListenAndServeTLS(":"+fmt.Sprint(config.SlackListener[0].Port), "/home/davidgs/.node-red/combined", "/home/davidgs/.node-red/combined", nil) // set listen port
		if err != nil {
			log.Fatal("ListenAndServeTLS: ", err)
		}
	} else {
		err := http.ListenAndServe(":"+fmt.Sprint(config.SlackListener[0].Port), nil) // set listen port
		if err != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	}
}
