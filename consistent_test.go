package consistent

import (
	"fmt"
	"testing"
)

func TestAdd(t *testing.T) {
	c := New()
	c.Add("server-001")
	if c.clients.Len() != c.replicationFactor {
		t.Fatal("vnodes number is incorrect")
	}
}

func TestGet(t *testing.T) {
	c := New()
	client := "client-001"
	server := "server-001"
	c.Add(server)
	host, err := c.Get(client)
	if err != nil {
		t.Fatal(err)
	}
	// there is only one server, so it should process the client
	if host != server {
		t.Fatalf("returned host is not what expected: %s got %s", host, client)
	}
}

func TestRemove(t *testing.T) {
	c := New()
	server := "server-001"
	c.Add(server)
	c.Remove(server)

	if c.clients.Len() != 0 && len(c.servers) != 0 {
		t.Fatal(("remove is not working"))
	}

}

func TestGetLeast(t *testing.T) {
	c := New()

	c.Add("shard-1")
	c.Add("shard-2")

	for i := 0; i < 100; i++ {
		host, err := c.GetLeast("92.0.0.1:80001")
		if err != nil {
			t.Fatal(err)
		}
		c.Inc(host)
	}

	for k, v := range c.GetLoads() {
		if v > c.MaxLoad() {
			t.Fatalf("host %s is overloaded. %d > %d\n", k, v, c.MaxLoad())
		}
	}
	fmt.Println("Max load per node", c.MaxLoad())
	fmt.Println(c.GetLoads())

}

func TestIncDone(t *testing.T) {
	c := New()

	c.Add("127.0.0.1:8000")
	c.Add("92.0.0.1:8000")

	host, err := c.GetLeast("92.0.0.1:8000")
	if err != nil {
		t.Fatal(err)
	}

	c.Inc(host)
	if c.loadMap[host].Load != 1 {
		t.Fatalf("host %s load should be 1\n", host)
	}

	c.Done(host)
	if c.loadMap[host].Load != 0 {
		t.Fatalf("host %s load should be 0\n", host)
	}

}

func TestHosts(t *testing.T) {
	hosts := []string{
		"127.0.0.1:8000",
		"92.0.0.1:8000",
	}

	c := New()
	for _, h := range hosts {
		c.Add(h)
	}
	fmt.Println("hosts in the ring", c.Servers())

	addedHosts := c.Servers()
	for _, h := range hosts {
		found := false
		for _, ah := range addedHosts {
			if h == ah {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("missing host", h)
		}
	}
	c.Remove("127.0.0.1:8000")
	fmt.Println("hosts in the ring", c.Servers())

}
