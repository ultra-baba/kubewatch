/*
Copyright 2016 Skippbox, Ltd.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package flock

import (
	"fmt"
	"log"
	"os"

	"github.com/bitnami-labs/kubewatch/config"
	"net/http"
	"encoding/json"
	"time"
	kbEvent "github.com/bitnami-labs/kubewatch/pkg/event"
	"bytes"
)

var flockColors = map[string]string{
	"Normal":  "#00FF00",
	"Warning": "#FFFF00",
	"Danger":  "#FF0000",
}

var flockErrMsg = `
%s

You need to set Flock url for Flock notify,
using "--url/-u" or using environment variables:

export KW_FLOCK_URL=flock_url

Command line flags will override environment variables

`

// Flock handler implements handler.Handler interface,
// Notify event to Flock channel
type Flock struct {
	Url string
}

type FlockMessage struct {
	Notification   string `json:"notification"`
	Text        string `json:"text"`
	Attachements []FlockMessageAttachement `json:"attachments"`
}

type FlockMessageAttachement struct {
	Title       string `json:"title"`
	Color       string `json:"color"`
    Views       FlockMessageAttachementViews `json:"views"`
}

type FlockMessageAttachementViews struct {
    Flockml       string `json:"flockml"`
}

// Init prepares Flock configuration
func (f *Flock) Init(c *config.Config) error {
	url := c.Handler.Flock.Url

	if url == "" {
		url = os.Getenv("KW_FLOCK_URL")
	}

	f.Url = url

	return checkMissingFlockVars(f)
}

func (f *Flock) ObjectCreated(obj interface{}) {
	notifyFlock(f, obj, "created")
}

func (f *Flock) ObjectDeleted(obj interface{}) {
	notifyFlock(f, obj, "deleted")
}

func (f *Flock) ObjectUpdated(oldObj, newObj interface{}) {
	notifyFlock(f, newObj, "updated")
}

func notifyFlock(f *Flock, obj interface{}, action string) {
	e := kbEvent.New(obj, action)

	flockMessage := prepareFlockMessage(e, f)

	err := postMessage(f.Url, flockMessage)
	if err != nil {
		log.Printf("%s\n", err)
		return
	}

	log.Printf("Message successfully sent to channel %s at %s", f.Url, time.Now())
}

func checkMissingFlockVars(s *Flock) error {
	if s.Url == "" {
		return fmt.Errorf(flockErrMsg, "Missing Flock url")
	}

	return nil
}

func prepareFlockMessage(e kbEvent.Event, f *Flock) *FlockMessage {
	msg := fmt.Sprintf(
		"<flockml> A %s: %s in namespace %s has been %s </flockml>",
		e.Kind,
		e.Name,
		e.Namespace,
		e.Reason,
	)

	title := fmt.Sprintf(
		"%s: %s has been %s",
		e.Namespace,
		e.Kind,
		e.Reason,
	)

	log.Println(e)

	return &FlockMessage{
		Text: "Kubewatch Alert",
		Notification: "Kubewatch Alert",
		Attachements: []FlockMessageAttachement{
			{
				Title: title,
				Color: flockColors[e.Status],
                Views: FlockMessageAttachementViews{         
            		Flockml: msg,
        		},
			},
		},
	}
}

func postMessage(url string, flockMessage *FlockMessage) (error){
	message, err := json.Marshal(flockMessage)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(message))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	_, err = client.Do(req)
	if err != nil {
		return err
	}

	return nil
}
