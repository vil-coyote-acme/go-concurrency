package client

import (
	"github.com/vil-coyote-acme/go-concurrency/commons"
	"net/http"
	"time"
	"log"
	"fmt"
)

var (
	regChan      chan commons.RegistrationWrapper
	registration map[string]commons.Registration
	rAddr string
	started   bool
)

func StartClient(redisAddr string)  {
	log.Println(fmt.Sprintf("client | create the client with the redis addr : %s", redisAddr))
	rAddr = redisAddr
	initRegistration()
	mux := http.NewServeMux()
	initRegistrationHandling(mux)
	if !started {
		log.Println("client | the client is starting, listening on 4444 port")
		started = true
		err := http.ListenAndServe(":4444", mux)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

}

func initRegistration() {
	regChan = make(chan commons.RegistrationWrapper)
	registration = make(map[string]commons.Registration, 10)
	go handleRegistration()
}

func initRegistrationHandling(mux *http.ServeMux) {
	mux.HandleFunc("/registration", registrationEndPoint)
}

func registrationEndPoint(w http.ResponseWriter, r *http.Request) {
	var reg commons.Registration
	commons.UnmarshalRegistrationFromHttp(r, &reg)
	log.Println(fmt.Sprintf("client | receive the registration query : %s", reg))
	regW := commons.RegistrationWrapper{Registration: reg, ResChan: make(chan bool)}
	regChan <- regW
	// beware here -> the producer must take in account that the consumer may be absent
	res, timedOut := commons.WaitAnswerWithTimeOut(regW.ResChan, time.Second*5)
	if timedOut {
		log.Println(fmt.Sprintf("client | registration unknow issue on query : %s", reg))
		w.WriteHeader(500)
		return
	}
	if res {
		log.Println(fmt.Sprintf("client | registration successful on query : %s", reg))
		w.WriteHeader(200)
		return
	}
	log.Println(fmt.Sprintf("client | registration failed for consistency issue on query : %s", reg))
	w.WriteHeader(403)
}

func handleRegistration() {
	for {
		// todo think about time out and test it !
		rw := <-regChan
		noConflict := hasNoConflict(&rw.Registration)
		if noConflict {
			registration[rw.PlayerId] = rw.Registration
		}
		rw.ResChan <- noConflict
	}
}

func hasNoConflict(r *commons.Registration) (res bool) {
	res = true
	re, ex := registration[r.PlayerId]
	if ex {
		res = re.Ip == r.Ip
		return
	}
	for _, val := range registration {
		if val.Ip == r.Ip {
			res = false
			break
		}
	}
	return
}