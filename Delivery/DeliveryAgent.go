package main

import (
	"time"
	"net/http"
	"io"
	"os"
	"io/ioutil"
	"strconv"
	"log"
	"github.com/garyburd/redigo/redis"
)

const (
	ADDRESS = "127.0.0.1:6379"
	ERROR_LOG = "/var/log/DeliveryAgent/error.txt"
	INFO_LOG = "/var/log/DeliveryAgent/postback_log.txt"
	PENDING_QUEUE = "Pending"
	WORKING_SET = "Working"
	STATS_HASH = "Stats"
	VALUES_HASH = "Values"
)

var (
	c, err = redis.Dial("tcp", ADDRESS)
	Error *log.Logger
	Info *log.Logger
)

//Function to initialize all log handlers
func initLogs() {
	errorFile, err := os.OpenFile(ERROR_LOG, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open error log file", ":", err)
	}

	infoFile, err := os.OpenFile(INFO_LOG, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open postback log file", ":", err)
	}

	multiError := io.MultiWriter(errorFile, os.Stdout)

	//Log handler for errors
	Error = log.New(multiError,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	//Log handler for postback information
	Info = log.New(infoFile, 
		"",
		0)
}

func main() {
	initLogs()

	//Loop constantly checking for new postback requests in Redis
	for {
		if err != nil {
			Error.Println("Error connecting to Redis")
			break
		}

		uuid, pendingErr := c.Do("RPOP", PENDING_QUEUE)

		if pendingErr != nil {
			Error.Println(pendingErr)
			continue
		}
		if uuid == nil {
			continue
		}

		var id string = string(uuid.([]byte))
		handlePostback(id)
	}
}

func handlePostback(uuid string) {
	millis := getTime()

	//Add UUID to Working set in case processing fails
	resp, err := c.Do("ZADD" , WORKING_SET, millis , uuid) 
	if err != nil || resp == nil {
		Error.Println("Could not add " + uuid + " to Working")
	}

	//Obtain the postback method type 
	method, methodErr := c.Do("HGET", VALUES_HASH, uuid + ":method")
	if methodErr != nil || method == nil {
		Error.Println("No Method for UUID: " + uuid);
		return
	}

	//Obtain the postback url
	url, urlErr := c.Do("HGET", VALUES_HASH, uuid)
	if urlErr != nil || url == nil {
		Error.Println("No URL for UUID: " + uuid);
		return
	}

	//Obtain the request origin time
	startTime, timeErr := c.Do("HGET", STATS_HASH, uuid + ":start")
	if timeErr != nil || startTime == nil {
		Error.Println("No start time for UUID: " + uuid);
		return
	}

	if string(method.([]byte)) == "GET" {
		handleGET(string(url.([]byte)), uuid, string(startTime.([]byte)))
	}

}

func handleGET(url string, uuid string, startTime string ) {
	deliveryTime := strconv.FormatInt(getTime(), 10)

	response, err := http.Get(url)

	if err != nil {
		Info.Println("HTTP GET FAILED")
		Error.Println("HTTP Get failed: " + uuid);
	} else {
		defer response.Body.Close()

		responseTime := strconv.FormatInt(getTime(), 10)

		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			Error.Println("Response read failed: " + uuid);
		}

		responseCode := strconv.Itoa(response.StatusCode)
		body := string(contents)

		logResponse(startTime, deliveryTime, responseTime, responseCode, body)

		cleanRedisData(uuid)
	}

}

func logResponse(startTime string, deliveryTime string, responseTime string, responseCode string,  body string) {
	Info.Println("*****");
	Info.Println("Request Time: " + startTime)
	Info.Println("Delivery Time: " + deliveryTime)
	Info.Println("Response Time: " + responseTime)
	Info.Println("Status Code: " + responseCode)
	Info.Println("Response Body: " + body)
	Info.Println("*****")
}

//Function to clean up Redis data after postback delivery
func cleanRedisData(uuid string) {
	c.Do("ZREM", WORKING_SET, uuid)
	c.Do("HDEL", VALUES_HASH, uuid)
	c.Do("HDEL", VALUES_HASH, uuid + ":method")
	c.Do("HDEL", STATS_HASH, uuid + ":start")
}

func getTime() int64 {
	now := time.Now().UTC()
	millis := now.UnixNano() / int64(time.Millisecond)
	return millis;
}
