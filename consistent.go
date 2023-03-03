// Package consistent An implementation of Consistent Hashing and
// Consistent Hashing With Bounded Loads.
//
// https://en.wikipedia.org/wiki/Consistent_hashing
//
// https://research.googleblog.com/2017/04/consistent-hashing-with-bounded-loads.html
package consistent

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/minio/blake2b-simd"
)

// Virtual hash node count
const replicationFactor = 10

var ErrNoHosts = errors.New("no hosts added")

type Host struct {
	Name   string
	Load   int64
	Weight int64
}

type Consistent struct {
	hosts       map[uint64]string
	sortedSet   []uint64
	loadMap     map[string]*Host
	totalLoad   int64
	totalWeight int64

	sync.RWMutex
}

func New() *Consistent {
	return &Consistent{
		hosts:     map[uint64]string{},
		sortedSet: []uint64{},
		loadMap:   map[string]*Host{},
	}
}

func (c *Consistent) Add(host string, weight int64) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.loadMap[host]; ok {
		return
	}

	c.loadMap[host] = &Host{Name: host, Load: 0, Weight: weight}
	for i := 0; i < replicationFactor; i++ {
		h := c.hash(fmt.Sprintf("%s%d", host, i))
		c.hosts[h] = host
		c.sortedSet = append(c.sortedSet, h)

	}
	// sort hashes ascendingly
	sort.Slice(c.sortedSet, func(i int, j int) bool {
		if c.sortedSet[i] < c.sortedSet[j] {
			return true
		}
		return false
	})

	c.totalWeight += weight
}

// Get Returns the host that owns `key`.
//
// As described in https://en.wikipedia.org/wiki/Consistent_hashing
//
// It returns ErrNoHosts if the ring has no hosts in it.
func (c *Consistent) Get(key string) (string, error) {
	c.RLock()
	defer c.RUnlock()

	if len(c.hosts) == 0 {
		return "", ErrNoHosts
	}

	h := c.hash(key)
	idx := c.search(h)
	return c.hosts[c.sortedSet[idx]], nil
}

// GetLeast It uses Consistent Hashing With Bounded loads
//
// https://research.googleblog.com/2017/04/consistent-hashing-with-bounded-loads.html
//
// to pick the least loaded host that can serve the key
//
// It returns ErrNoHosts if the ring has no hosts in it.
func (c *Consistent) GetLeast(key string) (string, error) {
	c.RLock()
	defer c.RUnlock()

	if len(c.hosts) == 0 {
		return "", ErrNoHosts
	}

	h := c.hash(key)
	idx := c.search(h)

	i := idx
	for {
		host := c.hosts[c.sortedSet[i]]
		if c.loadOK(host) {
			return host, nil
		}
		i++
		if i >= len(c.hosts) {
			i = 0
		}
	}
}

func (c *Consistent) search(key uint64) int {
	idx := sort.Search(len(c.sortedSet), func(i int) bool {
		return c.sortedSet[i] >= key
	})

	// 0 ~ c.sortedSet[0]
	if idx >= len(c.sortedSet) {
		idx = 0
	}
	return idx
}

// UpdateLoad Sets the load of `host` to the given `load`
func (c *Consistent) UpdateLoad(host string, load, weight int64) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.loadMap[host]; !ok {
		return
	}
	c.totalLoad -= c.loadMap[host].Load
	c.totalWeight -= c.loadMap[host].Weight
	c.loadMap[host].Load = load
	c.loadMap[host].Weight = weight
	c.totalLoad += load
	c.totalWeight += weight
}

// Inc Increments the load of host by 1
//
// should only be used with if you obtained a host with GetLeast
func (c *Consistent) Inc(host string) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.loadMap[host]; !ok {
		return
	}
	atomic.AddInt64(&c.loadMap[host].Load, 1)
	atomic.AddInt64(&c.totalLoad, 1)
}

// Done Decrements the load of host by 1
//
// should only be used with if you obtained a host with GetLeast
func (c *Consistent) Done(host string) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.loadMap[host]; !ok {
		return
	}
	atomic.AddInt64(&c.loadMap[host].Load, -1)
	atomic.AddInt64(&c.totalLoad, -1)
}

// Remove Deletes host from the ring
func (c *Consistent) Remove(host string) bool {
	c.Lock()
	defer c.Unlock()

	for i := 0; i < replicationFactor; i++ {
		h := c.hash(fmt.Sprintf("%s%d", host, i))
		delete(c.hosts, h)
		c.delSlice(h)
	}
	delete(c.loadMap, host)

	c.totalWeight -= c.loadMap[host].Weight

	return true
}

// Hosts Return the list of hosts in the ring
func (c *Consistent) Hosts() (hosts []string) {
	c.RLock()
	defer c.RUnlock()
	for k := range c.loadMap {
		hosts = append(hosts, k)
	}
	return hosts
}

// GetLoads Returns the loads of all the hosts
func (c *Consistent) GetLoads() map[string]int64 {
	loads := map[string]int64{}

	for k, v := range c.loadMap {
		loads[k] = v.Load
	}
	return loads
}

// MaxLoad Returns the maximum load of the single host
// which is:
// (total_load/number_of_hosts)*1.25
// total_load = is the total number of active requests served by hosts
// for more info:
// https://research.googleblog.com/2017/04/consistent-hashing-with-bounded-loads.html
func (c *Consistent) MaxLoad(host string) int64 {
	if c.totalLoad == 0 {
		c.totalLoad = 1
	}
	var avgLoadPerNode float64
	avgLoadPerNode = float64(c.totalLoad / c.totalWeight)
	if avgLoadPerNode == 0 {
		avgLoadPerNode = 1
	}
	avgLoadPerNode = math.Ceil(avgLoadPerNode * 1.25)
	if host == "" {
		return int64(avgLoadPerNode)
	}
	return int64(avgLoadPerNode) * c.loadMap[host].Weight
}

func (c *Consistent) loadOK(host string) bool {
	// a safety check if someone performed c.Done more than needed
	if c.totalLoad < 0 {
		c.totalLoad = 0
	}

	var avgLoadPerNode float64
	avgLoadPerNode = float64((c.totalLoad + 1) / c.totalWeight)
	if avgLoadPerNode == 0 {
		avgLoadPerNode = 1
	}
	avgLoadPerNode = math.Ceil(avgLoadPerNode * 1.25)

	bHost, ok := c.loadMap[host]
	if !ok {
		panic(fmt.Sprintf("given host(%s) not in loadsMap", bHost.Name))
	}

	if float64(bHost.Load)+1 <= avgLoadPerNode*float64(bHost.Weight) {
		return true
	}

	return false
}

func (c *Consistent) delSlice(val uint64) {
	idx := -1
	l := 0
	r := len(c.sortedSet) - 1
	for l <= r {
		m := (l + r) / 2
		if c.sortedSet[m] == val {
			idx = m
			break
		} else if c.sortedSet[m] < val {
			l = m + 1
		} else if c.sortedSet[m] > val {
			r = m - 1
		}
	}
	if idx != -1 {
		c.sortedSet = append(c.sortedSet[:idx], c.sortedSet[idx+1:]...)
	}
}

func (c *Consistent) hash(key string) uint64 {
	out := blake2b.Sum512([]byte(key))
	return binary.LittleEndian.Uint64(out[:])
}
