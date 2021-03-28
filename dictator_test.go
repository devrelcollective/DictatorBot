package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"gopkg.in/yaml.v2"
	"testing"
)

const ConfigString = `Dictator_version: 1
Camunda Host:
- name: camunda
  port: 8443
  protocol: https
  host: davidgs.com
Slack Listener:
- name: dictatorbot
  port: 9091
  protocol: https
  host: davidgs.com
Authorized senders:
- name: Mary Thengvall
  username: mary_grace
  is-on-call: false
- name: Jeremy Meiss
  username: jeremy
  is-on-call: false
- name: Jocelyn Mathews
  username: jocelyn
  is-on-call: true
  order: 2
- name: David G. Simmons
  username: davidgs
  is-on-call: true
  order: 1
- name: Quintessence Anx
  username: quintessence
  is-on-call: true
  order: 3
- name: Daniel Maher
  username: phrawzty
  is-on-call: true
  order: 4
Current On Call: jocelyn
On Call Index: 1
Slack Secret: shgenev5634635fhasdlh3q45
Total On Call: 4
Response Url:
Response Token:
AppID:
Channel ID: 
`

var dictators = [...]string{"mary_grace", "jeremy", "jocelyn", "davidgs", "quintessence", "phrawzty"}

const dictatorString = `*Currently Authorized Benevolent Dictators are:*
• Mary Thengvall (mary_grace)
• Jeremy Meiss (jeremy)
• Jocelyn Mathews (jocelyn)
• David G. Simmons (davidgs)
• Quintessence Anx (quintessence)
• Daniel Maher (phrawzty)
`

var rotation = [...]string{"davidgs", "jocelyn", "quintessence", "phrawzty"}

const rotationString = "davidgs-->jocelyn-->quintessence-->phrawzty"

func init_config() {
err := yaml.Unmarshal([]byte(ConfigString), &config)
	if err != nil {

	}
}
func TestGetDictators(t *testing.T) {
	init_config()
	result := getDictators()
	if len(result) == 0 {
		t.Errorf("getDictators Failed expected %d got %v ", 6, len(result))
	}
	if len(result) != len(dictators) {
		t.Errorf("getDictators Failed expected %d got %v ", 6, len(result))
	}
	for x := 0; x < len(result); x++ {
		if result[x] != dictators[x] {
			t.Errorf("getDictators Failed expected %s got %s ", dictators[x], result[x])
		}
	}
}

func TestSendDirect(t *testing.T) {
	input := "directmessage"
	if !SendDirect(input) {
		t.Errorf("sendDirect Failed expected %v got %v ", true, false)
	}
	input = "otherMessage"
	if SendDirect(input) {
		t.Errorf("sendDirect Failed expected %v got %v ", false, true)
	}
}

func TestGetDictatorString(t *testing.T) {
	init_config()
	result := getDictatorString()
	if result != dictatorString {
		t.Errorf("getDictatorString Failed expected\n %v \ngot\n %v ", dictatorString, result)
	}
}

func TestGetRotation(t *testing.T) {
	init_config()
		result := getRotation()

	if len(result) != len(rotation) {
		t.Errorf("getRotation Failed expected %v got %v ", len(rotation), len(result))
	}
	for x := 0; x < len(result); x++ {
		if rotation[x] != result[x] {
			t.Errorf("getRotation Failed expected %v got %v ", rotation[x], result[x])
		}
	}
}

func TestGetRotationString(t *testing.T) {
	init_config()
	input := rotationString
	result := getRotationString()
	if result != input {
		t.Errorf("getRotationString Failed expected %v got %v ", input, result)
	}
}

func TestGetOnCall(t *testing.T) {
	init_config()
	input := config.CurrentOnCall
	result := getOnCall()
	if input != result {
		t.Errorf("getOnCall Failed expected %v got %v ", input, result)
	}
}

func TestGetNextOnCall(t *testing.T) {
	init_config()
	input := rotation[config.OnCallIndex+1]
	result := getNextOnCall()
	if result != input {
		t.Errorf("getNextOnCall Failed expected %v got %v ", input, result)
	}

}

func TestGetOnCallIndex(t *testing.T) {
	init_config()
	for x := 0; x < len(rotation); x++ {
		result := getOnCallIndex(rotation[x])
		if result != x {
			t.Errorf("getOnCallIndex Failed expected %v got %v ", x, result)
		}
	}
}

func TestRotateOnCallIndex(t *testing.T) {
	init_config()
	for x := 0; x < len(rotation); x++ {
		result := rotateOnCallIndex(x)
		if result != rotation[x] {
			t.Errorf("rotateOnCallIndex Failed expected %v got %v ", rotation[x], result)
		}
	}
	result := rotateOnCallIndex(4)
	if result != rotation[0] {
		t.Errorf("rotateOnCallIndex Failed expected %v got %v ", rotation[0], result)
	}
}

func TestRotateOnCall(t *testing.T) {
	init_config()
	for x := 0; x < len(rotation); x++ {
		result := rotateOnCall(rotation[x])
		if result != rotation[x] {
			t.Errorf("rotateOnCallIndex Failed expected %v got %v ", rotation[x], result)
		}
	}
}

func TestCheckHeader(t *testing.T) {
	init_config()
	h := hmac.New(sha256.New, []byte(config.SlackSecret))
	h.Write([]byte(ConfigString))
	sha := hex.EncodeToString(h.Sum(nil))
	input := fmt.Sprintf("v0=%s", sha)
	result := checkHeader(input, ConfigString)
	if !result {
		t.Errorf("checkHeader Failed got %v", result)
	}
}

func TestIsValueInList(t *testing.T) {
	var test = make([]string, 4)
	for x := 0; x < len(rotation); x++ {
		test[x] = rotation[x]
	}
	for x := 0; x < len(rotation); x++ {
		result := isValueInList(rotation[x], test)
		if !result {
			t.Errorf("isValueInList Failed expected %v got %v ", true, result)
		}
	}
	result := isValueInList("@anais", test)
	if result {
		t.Errorf("isValueInList Failed expected %v got %v ", false, result)
	}
}