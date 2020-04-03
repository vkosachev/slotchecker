package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type DayList []struct {
	Date  string `json:"date"`
	Slots []struct {
		ID              string  `json:"id"`
		EndOrderingTime float64 `json:"end_ordering_time"`
		TimeRange       string  `json:"time_range"`
		Price           int     `json:"price"`
		Currency        string  `json:"currency"`
		IsOpen          bool    `json:"is_open"`
		Date            string  `json:"date"`
	} `json:"items"`
}

type sendMessageReqBody struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

type Subscriber struct {
	Id int
	User int64
	Channel int64
}


// Function checks slot availability and notifies bot if new slots appear
func CheckSlots() (bool, int, error){
	url := os.Getenv("URL")

	var finalUrl strings.Builder

	finalUrl.WriteString(url)
	finalUrl.WriteString(fmt.Sprintf("%g", 46.9800804))
	finalUrl.WriteString(",")
	finalUrl.WriteString(fmt.Sprintf("%g", 28.8575673))


	resp, err := http.Get(finalUrl.String())
	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()

	var days DayList

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	err = json.Unmarshal(body, &days)

	if err != nil{
		log.Fatalln(err)
	}

	var openSlot bool
	var numSlots = 0
	for _, v := range days {
		for _, slot:= range v.Slots {
			if slot.IsOpen {
				openSlot=true
				numSlots++
			}
		}
	}

	return openSlot, numSlots, err


}

func notifySubscribers(chatIDS [] int64, message string) error {


	for _, chat:= range chatIDS {
		// Create the request body struct
		reqBody := &sendMessageReqBody{
			ChatID: chat,
			Text:   message,
		}

		// Create the JSON body from the struct
		reqBytes, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}

		// Send a post request with your token
		var postUrl strings.Builder
		postUrl.WriteString("https://api.telegram.org/bot")
		postUrl.WriteString(os.Getenv("BOT_TOKEN"))
		postUrl.WriteString("/sendMessage")

		res, err := http.Post(postUrl.String(), "application/json", bytes.NewBuffer(reqBytes))
		if err != nil {
			return err
		}
		if res.StatusCode != http.StatusOK {
			return errors.New("unexpected status" + res.Status)
		}
	}
	return errors.New("unexpected behaviour")
}

func getSubscribers() ( []int64, error) {

	var channels []int64
	db, err := sql.Open("sqlite3", "./subscribers.db")
	if err != nil {
		log.Println(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT channel FROM subscribers")
	if err != nil {
		log.Println(err)
	}
	defer rows.Close()

	for rows.Next() {
		var subscr Subscriber
		err = rows.Scan(&subscr.Channel)
		if err != nil {
			log.Println(err)
		}

		channels = append(channels, subscr.Channel)

	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return channels, err

}

func getSuccessMessage(numslots int) string {
	var message strings.Builder
	message.WriteString("WE HAVE OPEN SLOT(S)[")
	message.WriteString(fmt.Sprintf("%s", numslots))
	message.WriteString("] ! Act quickly!")

	return message.String()
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	pollInterval, err := strconv.ParseInt(os.Getenv("POLL_TIME"),10, 32)
	if err != nil {
		pollInterval = 300
	}


	timerCh := time.Tick(time.Duration(pollInterval) * time.Second)

	for range timerCh {
		slotExists, numSlots, err := CheckSlots()
		if err != nil {
			log.Println(err)
		}
		if slotExists != false {
			chats, err := getSubscribers()
			if err != nil {
				log.Println(err)
			}

			err = notifySubscribers(chats, getSuccessMessage(numSlots))
			if err != nil {
				log.Println(err)
			}
		}
	}

	timer2 := time.Tick(time.Duration(60) * time.Minute)

	for range timer2 {
		chats, err := getSubscribers()
		if err != nil {
			log.Println(err)
		}

		err = notifySubscribers(chats, "Still No slots ")
		if err != nil {
			log.Println(err)
		}
	}


}