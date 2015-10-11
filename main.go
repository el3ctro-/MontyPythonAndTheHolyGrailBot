package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/boltdb/bolt"
)

type jsonResult struct {
	Ok          bool     `json:"ok"`
	Description string   `json:"description"`
	Result      []result `json:"result"`
	ErrorCode   int      `json:"error_code"`
}

type result struct {
	// UpdateID is the offset
	UpdateID int     `json:"update_id"`
	Message  message `json:"message"`
}

type message struct {
	MessageID int    `json:"message_id"`
	From      from   `json:"from"`
	Text      string `json:"text"`
}

type from struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func main() {
	db, err := bolt.Open("montypythonandtheholygrail.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("Offset"))
		if err != nil {
			return fmt.Errorf("Error creating bucket: %s", err)
		}
		v := b.Get([]byte("Offset number"))
		offsetNum, err := strconv.Atoi(string(v))
		if offsetNum > 859484443 {
			return nil
		}
		// Default value
		b.Put([]byte("Offset number"), []byte("1"))
		return nil
	})

	db.Update(func(tx *bolt.Tx) error {
		// Insert lyrics
		quotes, err := tx.CreateBucketIfNotExists([]byte("Quotes"))
		if err != nil {
			return fmt.Errorf("Error creating bucket: %s", err)
		}

		f, err := os.Open("./quotes.txt")
		if err != nil {
			fmt.Printf("Err: %#v", err)
		}
		defer f.Close()
		var quotesCollection []string
		reader := bufio.NewReader(f)
		for i := 0; i < 255; i++ {
			result, err := reader.ReadString(byte(42))
			if err != nil {
				break
			}
			if len(result) > 1 {
				quotes.Put([]byte(result), []byte(""))
				quotesCollection = append(quotesCollection, result)
			} else {
				break
			}
		}
		quotes.Put([]byte("Number Of Quotes"), []byte(strconv.Itoa(len(quotesCollection))))
		fmt.Println("inserting 8ball")
		// Insert 8ball answers

		eightBall, err := tx.CreateBucketIfNotExists([]byte("8Ball"))
		if err != nil {
			return fmt.Errorf("Error creating bucket: %s", err)
		}

		f, err = os.Open("./8ball.txt")
		if err != nil {
			fmt.Printf("Err: %#v", err)
		}
		defer f.Close()
		var eightBallList []string
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			eightBallList = append(eightBallList, scanner.Text())
			eightBall.Put([]byte(scanner.Text()), []byte(""))
		}
		eightBall.Put([]byte("Number Of Answers"), []byte(strconv.Itoa(len(eightBallList))))

		return nil
	})

	tr := &http.Transport{
		DisableKeepAlives: false,
	}
	client := &http.Client{Transport: tr}
	go func() {

		for {
			var offset int
			db.View(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte("Offset"))
				c := b.Get([]byte("Offset number"))
				offset, err = strconv.Atoi(string(c))
				if err != nil {
					return err
				}
				return nil
			})
			postStr := "offset=" + strconv.Itoa(offset) + "&timeout=604800&limit=10"

			bytesFromTelegram := sendToTelegram("getUpdates", postStr, client)
			// Create empty result object
			var result jsonResult
			err = json.Unmarshal(bytesFromTelegram, &result)
			if err != nil {
				fmt.Errorf("Error! %s", err)
			}
			if len(result.Result) > 0 {
				finalResult := result.Result[len(result.Result)-1]

				db.Update(func(tx *bolt.Tx) error {
					fmt.Printf("Updating offset: %v", finalResult.UpdateID)
					fmt.Printf("\n%s\n", finalResult.Message.Text)
					b := tx.Bucket([]byte("Offset"))
					err := b.Put([]byte("Offset number"), []byte(strconv.Itoa(finalResult.UpdateID+1)))
					if err != nil {
						fmt.Printf("Error updating offset: %s\n", err)
					}
					return nil
				})

				for _, result := range result.Result {
					switch result.Message.Text {
					default:
						var answerStr string
						err := db.View(func(tx *bolt.Tx) error {
							b := tx.Bucket([]byte("Quotes"))
							lyricNum, err := strconv.Atoi(string(b.Get([]byte("Number Of Quotes"))))
							if err != nil {
								fmt.Printf("Error converting to int: %d", lyricNum)
							}
							c := b.Cursor()
							var step int
							rand.Seed(int64(rand.Intn(2222)))
							fmt.Printf("Random num: %d", rand.Intn(lyricNum))
							randomNumber := rand.Intn(lyricNum - 1)
							for k, _ := c.First(); k != nil; k, _ = c.Next() {
								step++
								if step == randomNumber {
									answerStr = string(k[0 : len(k)-1])
								}
							}
							// result = k
							return nil
						})

						err = db.View(func(tx *bolt.Tx) error {
							b := tx.Bucket([]byte("8Ball"))
							answerNum, err := strconv.Atoi(string(b.Get([]byte("Number Of Answers"))))
							if err != nil {
								fmt.Printf("Error converting to int: %d", answerNum)
							}
							c := b.Cursor()
							var step int
							rand.Seed(int64(rand.Intn(2222)))
							randomNumber := rand.Intn(answerNum - 1)
							for k, _ := c.First(); k != nil; k, _ = c.Next() {
								step++
								if step == randomNumber {
									answerStr = answerStr + "\n" + string(k) + ", " + result.Message.From.FirstName + "."
									return nil
								}
							}
							return nil
						})

						bytesFromTelegram := sendToTelegram("sendMessage", "chat_id="+strconv.Itoa(result.Message.From.ID)+"&text="+answerStr, client)
						var result jsonResult
						err = json.Unmarshal(bytesFromTelegram, &result)
						if err != nil {
							fmt.Errorf("Error! %s", err)
						}
						fmt.Printf("%#v", result)
						// default:
						// 	bytesFromTelegram := sendToTelegram("sendMessage", "chat_id="+strconv.Itoa(result.Message.From.ID)+"&text=Try me, "+result.Message.From.FirstName+".&reply_markup={keyboard:[['Send me lyrics']]}", client)
						// 	var result jsonResult
						// 	err = json.Unmarshal(bytesFromTelegram, &result)
						// 	if err != nil {
						// 		fmt.Errorf("Error! %s", err)
						// 	}
						// 	fmt.Printf("%#v", result)
					}
				}

			} else {
				fmt.Println("no results")
			}
		}
	}()
	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	// Block until a signal is received.
	s := <-c
	db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte("Quotes"))
		if err != nil {
			return fmt.Errorf("delete bucket error: %s", err)
		}
		err = tx.DeleteBucket([]byte("8Ball"))
		if err != nil {
			return fmt.Errorf("delete bucket error: %s", err)
		}
		return nil
	})
	db.Close()

	fmt.Println("Got signal:", s)
}

func sendToTelegram(method string, param string, client *http.Client) []byte {
	fmt.Println("\nSending...\nParam " + param + "\nMethod " + method)
	telegramUrl := os.Getenv("MONTYPYTHONBOT")
	if len(telegramUrl) == 0 {
		log.Fatal("Could not find environment variable MONTYPYTHONBOT.  Please set it in your .bashrc or .bash_profile")
	}
	resp, err := client.Post(telegramUrl+method, "application/x-www-form-urlencoded", strings.NewReader(param))
	defer resp.Body.Close()
	if err != nil {
		log.Fatalf("Fatal Error! %s\n\n\n", err)
	}
	myBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("\n\nError! %s\n", err)
	}
	return myBytes

}
