package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/mail"
	"net/smtp"
	"strconv"
	"time"
)

type responsecenter struct {
	Centers []center `json:"centers"`
}

type center struct {
	Address      string    `json:"address"`
	BlockName    string    `json:"block_name"`
	CenterID     int       `json:"center_id"`
	DistrictName string    `json:"district_name"`
	FeeType      string    `json:"fee_type"`
	From         string    `json:"from"`
	Lat          int       `json:"lat"`
	Long         int       `json:"long"`
	Name         string    `json:"name"`
	Pincode      int       `json:"pincode"`
	StateName    string    `json:"state_name"`
	To           string    `json:"to"`
	Sessions     []session `json:"sessions"`
}

type session struct {
	AvailableCapacity int      `json:"available_capacity"`
	Date              string   `json:"date"`
	MinAgeLimit       int      `json:"min_age_limit"`
	SessionID         string   `json:"session_id"`
	Vaccine           string   `json:"vaccine"`
	Slots             []string `json:"slots"`
}

// smtpServer data to smtp server.
type smtpServer struct {
	host string
	port string
}

const (
	URL          = "https://cdn-api.co-vin.in/api/v2/appointment/sessions/public/calendarByPin"
	SENDER_EMAIL = "**********"
	PASSWORD     = "*******"
)

// Address URI to smtp server.
func (s *smtpServer) Address() string {
	return s.host + ":" + s.port
}

func valid(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func main() {
	err := false
	name := flag.String("name", "COVAXIN", "name of the vaccine: COVAXIN OR COVISHIELD")
	age := flag.Int("age", 18, "min age: 18 or 45")
	pc := flag.Int("pincode", 814112, "pincode of your area")
	email := flag.String("email", "", "Please enter your email the notification")
	flag.Parse()

	if *name != "COVAXIN" && *name != "COVISHIELD" {
		fmt.Println("Please provide name either: COVAXIN OR COVISHIELD")
		err = true
	}
	if *age != 18 && *age != 45 {
		fmt.Println("Please provide name either: 18 or 45")
		err = true
	}
	length := len(strconv.Itoa(*pc))
	if length != 6 {
		fmt.Println("Pincode format wrong. Eg: 700001")
		err = true
	}
	if *email == "" {
		fmt.Println("Email can not be empty")
		err = true
	}
	if !valid(*email) {
		fmt.Println("Please enter a valid email")
		err = true
	}
	if err == true {
		return
	}
	pincode := fmt.Sprintf("%v", *pc)
	ticker := time.NewTicker(9 * time.Second)
	quit := make(chan struct{})
	wait := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				waitin := make(chan struct{})

				loc, _ := time.LoadLocation("Asia/Kolkata")
				t := time.Now().In(loc).Format("02-01-2006")
				t7 := time.Now().AddDate(0, 0, 7).In(loc).Format("02-01-2006")
				t14 := time.Now().AddDate(0, 0, 14).In(loc).Format("02-01-2006")

				go sendNotification(pincode, t, waitin, *name, *age, *email)
				go sendNotification(pincode, t7, waitin, *name, *age, *email)
				go sendNotification(pincode, t14, waitin, *name, *age, *email)

				<-waitin
				<-waitin
				<-waitin
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	<-wait

}

func sendNotification(pincode string, date string, wait chan struct{}, name string, age int, email string) error {

	url := URL + "?pincode=" + pincode + "&date=" + date
	// startTime := time.Now().UnixNano() / 1000000

	// ==============================start block mail send ======================================
	from := SENDER_EMAIL
	password := PASSWORD
	to := []string{email}
	smtpServer := smtpServer{host: "smtp.gmail.com", port: "587"}
	auth := smtp.PlainAuth("", from, password, smtpServer.host)
	// ==============================end block mail send ========================================

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request")
		return err
	}
	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.76 Safari/537.36")

	var client http.Client
	// var acceptedStatusCodes = []int32{200, 201, 202, 203, 204}

	response, err := client.Do(request)
	if err != nil {
		fmt.Println("Error sending message ", err)
		return err
	} else {
		// endTime := time.Now().UnixNano() / 1000000
		// fmt.Println(endTime-startTime, "Millisecond")

		fmt.Println("Date ", date, "Status ", response.Status)
		bodyBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Println(err)
		}

		var centersresp responsecenter
		if err := json.Unmarshal(bodyBytes, &centersresp); err != nil {
			panic(err)
		}

		for _, center := range centersresp.Centers {
			for _, session := range center.Sessions {
				if session.Vaccine == name && session.MinAgeLimit == age && session.AvailableCapacity > 0 {

					fromAdd := fmt.Sprintf("From: <%s>\r\n", SENDER_EMAIL)
					toAddress := fmt.Sprintf("To: <%s>\r\n", email)
					subject := "Subject: Vaccine Availability\r\n"
					body := "Availablilty info: " + center.Name + center.Address + session.Date + ".\r\nBye\r\n"

					msg := fromAdd + toAddress + subject + "\r\n" + body
					fmt.Println("availablilty info: ", center.Name, center.Address, session.Date, session.AvailableCapacity)
					err := smtp.SendMail(smtpServer.Address(), auth, from, to, []byte(msg))
					if err != nil {
						fmt.Println(err)
					}
					fmt.Println("Email Sent!")
				}
			}
		}

	}
	err = response.Body.Close()
	if err != nil {
		fmt.Println("Error closing response body", err)
	}
	wait <- struct{}{}
	return nil
}
