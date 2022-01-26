package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/heroku/drain"
)

// setup env vars
var (
	app            string = ""
	port           string = ""
	firebaseURL    string = ""
	alertEmail     string = ""
	alertPassword  string = ""
	alertReceiver  string = ""
	healthCheckURL string = ""
)

// fetch env vars
func init() {
	app = os.Getenv("LOG_APP")
	port = os.Getenv("PORT")
	firebaseURL = os.Getenv("FIREBASE_URL")
	alertEmail = os.Getenv("EMAIL")
	alertPassword = os.Getenv("PASSWORD")
	alertReceiver = os.Getenv("ME")
	healthCheckURL = os.Getenv("HEALTH")
}

// sendToFirebase lets you store the log in firebase
func sendToFirebase(data string) {

	_, err := http.Post(firebaseURL+"/"+app+".json", "application/json", bytes.NewBuffer([]byte(data)))
	if err != nil {
		log.Print(err)
	}
}

// receiveLogs runs as a goroutine and works through log that comes in
func receiveLogs(d *drain.Drain, useJson bool) {
	for line := range d.Logs() {
		handleLog(line, useJson)
	}
}

// handleLog takes a log entry and fires the goroutine to send it in our store
func handleLog(line *drain.LogLine, useJson bool) {
	go sendToFirebase(line.Data)
}

func handleError(err error) {
	if err != nil {
		log.Fatalf(err.Error())
	}
}

func encodeRFC2047(String string) string {
	addr := mail.Address{Address: String}
	return strings.Trim(addr.String(), "<>@")
}

// getEmailHeaders will construct and send all email headers necessary
func getEmailHeaders(from mail.Address, to mail.Address, title string) map[string]string {

	header := make(map[string]string)
	header["Return-Path"] = from.String()
	header["From"] = from.String()
	header["To"] = to.String()
	header["Subject"] = encodeRFC2047(title)
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/plain; charset=\"utf-8\""
	header["Content-Transfer-Encoding"] = "base64"
	return header
}

// fireAlert will send an alert email
func fireAlert() {

	smtpHost := "smtp.gmail.com"
	smtpPort := "587" // for TLS support

	auth := smtp.PlainAuth(
		"",
		alertEmail,
		alertPassword,
		smtpHost,
	)

	from := mail.Address{Name: "kkrh", Address: alertEmail}
	to := mail.Address{Name: "", Address: alertReceiver}
	body := "Man down! Man down!"
	title := "kkrh check"

	header := getEmailHeaders(from, to, title)
	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + base64.StdEncoding.EncodeToString([]byte(body))

	err := smtp.SendMail(
		smtpHost+":"+smtpPort,
		auth,
		from.Address,
		[]string{to.Address},
		[]byte(message),
	)
	if err != nil {
		log.Print(err)
	}
}

// doHealthCheck will keep hitting the health endpoint every 25 minutes
func doHealthCheck(done <-chan bool, url string) {

	ticker := time.NewTicker(25 * time.Minute)
	for {
		select {
		case <-done:
			log.Print("Fin.")
			break
		case <-ticker.C:
			resp, err := http.Get(url)
			if err != nil {
				log.Println(err)
			}
			if resp != nil && resp.StatusCode != 200 {
				log.Print("firing alert")
				fireAlert()
			}
		}
	}
}

func main() {

	bucket := drain.NewDrain()
	http.HandleFunc("/"+app+"/logs", bucket.LogsHandler)

	done := make(chan bool)
	go receiveLogs(bucket, false)

	err := http.ListenAndServe(":"+port, nil)
	done <- true
	handleError(err)
}
