# Package consistent
A Golang implementation of Consistent Hashing and Consistent Hashing With Bounded Loads.

https://en.wikipedia.org/wiki/Consistent_hashing

https://research.googleblog.com/2017/04/consistent-hashing-with-bounded-loads.html


### Consistent Hashing Example

```go

package main

import (
	"github.com/dlwm/consistent"
	"log"
)

func main() {
	c := consistent.New(0)

	// adds the hosts to the ring
	c.Add("127.0.0.1:8000", 1)
	c.Add("92.0.0.1:8000", 1)

	// Returns the host that owns `key`.
	//
	// As described in https://en.wikipedia.org/wiki/Consistent_hashing
	//
	// It returns ErrNoHosts if the ring has no hosts in it.
	host, err := c.Get("/app.html")
	if err != nil {
		log.Fatal(err)
	}

	log.Println(host)
}

```


### Consistent Hashing With Bounded Loads Example

```go

package main

import (
	"github.com/dlwm/consistent"
	"log"
)

func main() {
	c := consistent.New(0)

	// adds the hosts to the ring
	c.Add("127.0.0.1:8000",1)
	c.Add("92.0.0.1:8000",1)

	// It uses Consistent Hashing With Bounded loads
	// https://research.googleblog.com/2017/04/consistent-hashing-with-bounded-loads.html
	// to pick the least loaded host that can serve the key
	//
	// It returns ErrNoHosts if the ring has no hosts in it.
	//
	host, err := c.GetLeast("/app.html")
	if err != nil {
		log.Fatal(err)
	}

	// increases the load of `host`, we have to call it before sending the request
	c.Inc(host)

	// send request or do whatever
	log.Println("send request to", host)

	// call it when the work is done, to update the load of `host`.
	c.Done(host)

}

```


# License

```
