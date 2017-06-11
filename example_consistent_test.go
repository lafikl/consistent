package consistent_test

import (
	"fmt"
	"log"
	"testing"

	"github.com/lafikl/consistent"
)

func ExampleGetLeast(t *testing.T) {
	c := consistent.New()

	// adds the hosts to the ring
	c.Add("127.0.0.1:8000")
	c.Add("92.0.0.1:8000")

	// find the least loaded node that can take our request
	host, err := c.GetLeast("/app.html")
	if err != nil {
		// it returns ErrExhausted if and only if the all of the nodes load above the MaxLoad
		// which should never happen if the library is used right
		//
		// `host` will always contain a host, if an error returned
		// it'll be whatever c.Get(host) would return
		log.Fatal(err)
	}
	// increases the load of `host`, we have to call it before sending the request
	c.Inc(host)
	// send request or do whatever
	fmt.Println("send request to", host)
	// call it when the work is done, to update the load of `host`.
	c.Done(host)

}
