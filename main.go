package main

import (
	"fmt"
	"io/ioutil"
	"log"

	// "net/http"
	"os"
	"time"

	"github.com/dying/gista"
	"github.com/dying/gista/errs"
	gistahelpers "github.com/dying/gista/gista-helpers"
	"gopkg.in/yaml.v2"
)

// Note: struct fields must be public in order for unmarshal to
// correctly populate the data.
type Conf struct {
	Instagram struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	}
	Delay     int `yaml:"delay"`
	Followers int `yaml:"followers"`
}

func (c *Conf) getConf() *Conf {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	if _, err := os.Stat(dir + "/config.yml"); os.IsNotExist(err) {
		log.Print("[-] Generating config file...")
		var username string
		log.Print("[?] What is your username?")
		_, err := fmt.Scanln(&username)
		if err != nil {
			log.Fatal(err)
		}
		c.Instagram.Username = username

		var password string
		log.Print("[?] What is your password?")
		_, err = fmt.Scanln(&password)
		if err != nil {
			log.Fatal(err)
		}
		c.Instagram.Password = password

		var delay int
		log.Print("[?] What is the delay in ms (each time that it will check for new followers) ?")
		_, err = fmt.Scanln(&delay)
		if err != nil {
			log.Fatal(err)
		}
		c.Delay = delay

		var followers int
		log.Print("[?] What is the followers limit (number of followers to accept everytime, 0 for no limit) ?")
		_, err = fmt.Scanln(&followers)
		if err != nil {
			log.Fatal(err)
		}

		c.Followers = followers

		d, err := yaml.Marshal(&c)
		if err != nil {
			log.Fatal(err)
		}

		log.Print("[-] Writing config file...")
		err = ioutil.WriteFile("config.yml", d, 0644)
		if err != nil {
			log.Fatal(err)
		}
		log.Print("[+] Done!")

	}

	yamlFile, err := ioutil.ReadFile("config.yml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

func main() {
	var c Conf
	c.getConf()

	log.Print("[-] Logging in...\n")
	ig, err := gista.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	// Log to instagram
	login(ig, c.Instagram.Username, c.Instagram.Password)

	log.Print("[+] Logged!\n")

	/*
		log.Print("[-] Getting authorization...")



		url := fmt.Sprintf("https://api.init.wtf/api/v1/autoaccepter/check?username=%s", ig.Username)
		client := &http.Client{}
		request, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatalln(err)
		}
		request.Header.Set("Authorization", "BneE4BSTZphu8Omwc7ZPgtxiA69RZxJEODjGJp8Pgzp1fbdZKx6FrCYv1BvUbgRT")
		resp, err := client.Do(request)
		if err != nil {
			log.Fatal(err)
		}
		if resp.StatusCode != 200 {
			log.Fatal("[-] USERNAME NOT AUTHORIZED. QUITTING.")
		}

		log.Print("[+] Authorized. Welcome.\n")

	*/

	approved := 0
	for {
		requests, err := ig.Account.GetPendingFollowRequests()
		if err != nil {
			log.Fatal(err)
		}

		for number, user := range requests.Users {
			if _, err = ig.People.ApproveFriendship(user.Pk); err != nil {
				log.Fatal(err)
			}

			approved++

			if c.Followers != 0 {
				if number == c.Followers {
					break
				}
			}
		}

		log.Printf("[+] Accepted every followers request (%d). Retrying in %d ms ", approved, c.Delay)
		time.Sleep(time.Duration(c.Delay) * time.Millisecond)
	}

}

func login(ig *gista.Instagram, username, password string) {
	err := ig.Login(username, password, false)
	if err != nil {
		switch err.(type) {
		// If 2FA is enabled, ask the code
		case errs.TwoFactorRequired:
			twoFactorErr := err.(errs.TwoFactorRequired)
			// Get the TwoFactorIdentifier, for the Instagram API
			// i.e: email, sms or authy
			twoFactorInfo := twoFactorErr.GetTwoFactorInfo()
			var code string
			log.Print("[?] What is your 2FA code?")
			_, err := fmt.Scanln(&code)
			if err != nil {
				log.Fatal(err)
			}
			if err := ig.FinishTwoFactorLogin(ig.Username, ig.Password, twoFactorInfo.TwoFactorIdentifier, code); err != nil {
				log.Fatal(err)
			}
		case errs.CheckpointRequired:
			log.Print("checkpoint")
			checkPointErr := err.(errs.CheckpointRequired)
			address := checkPointErr.GetCheckpointUrl()
			cs := gistahelpers.NewChallengeSolver()
			ch, err := cs.GetChallengeByUrl(address)
			if err != nil {
				log.Fatal(err)
				return
			}
			_, err = cs.GetSolveChallengeByEmail(address, ch.Config.CsrfToken, ch.RollOutHash)
			if err != nil {
				log.Fatal(err)
				return
			}
			var code string
			log.Print("[?] What is your 2FA code?")
			_, err = fmt.Scanln(&code)
			if err != nil {
				log.Fatal(err)
			}
			chsr, err := cs.SolveChallenge(address, ch.Config.CsrfToken, ch.RollOutHash, code)
			if err != nil {
				log.Fatal(err)
				return
			}
			log.Print(chsr)
			login(ig, username, password)
		case errs.ChallengeRequired:
			checkPointErr := err.(errs.ChallengeRequired)
			address := checkPointErr.GetChallenge()
			cs := gistahelpers.NewChallengeSolver()
			ch, err := cs.GetChallengeByUrl("https://i.instagram.com" + address.ApiPath)
			if err != nil {
				log.Fatal(err)
				return
			}
			_, err = cs.GetSolveChallengeByEmail("https://i.instagram.com"+address.ApiPath, ch.Config.CsrfToken, ch.RollOutHash)
			if err != nil {
				log.Fatal(err)
				return
			}
			var code string
			log.Print("[?] What is your 2FA code?")
			_, err = fmt.Scanln(&code)
			if err != nil {
				log.Fatal(err)
			}
			chsr, err := cs.SolveChallenge("https://i.instagram.com"+address.ApiPath, ch.Config.CsrfToken, ch.RollOutHash, code)
			if err != nil {
				log.Fatal(err)
			}
			log.Print(chsr)
			login(ig, username, password)
		default:
			log.Fatal(err)
		}
	}
}
