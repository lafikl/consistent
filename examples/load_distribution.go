package main

import (
	"fmt"
	"log"

	"github.com/lafikl/consistent"
)

func main() {
	prefix := "cluster"
	consistentHashing := consistent.New()
	clusters := []string{}
	clusterCount := 120

	for i := 1; i <= clusterCount; i++ {
		cluster := fmt.Sprintf("%s-%d", prefix, i)
		clusters = append(clusters, cluster)
	}
	// adds the hosts to the ring
	consistentHashing.Add("shard-1")
	consistentHashing.Add("shard-2")
	consistentHashing.Add("shard-3")
	log.Printf("------------- %d shards and %d clusters -------------", len(consistentHashing.Servers()), len(clusters))
	loadDistribution := distribute(clusters, consistentHashing)
	printLoadDistribution(loadDistribution)

	consistentHashing.Remove("shard-2")
	log.Printf("------------- %d shards and %d clusters -------------", len(consistentHashing.Servers()), len(clusters))
	loadDistribution = distribute(clusters, consistentHashing)
	printLoadDistribution(loadDistribution)

	consistentHashing.Add("shard-2")
	consistentHashing.Add("shard-4")

	loadDistribution = distribute(clusters, consistentHashing)
	printLoadDistribution(loadDistribution)

	consistentHashing.Remove("shard-3")
	consistentHashing.Remove("shard-4")
	consistentHashing.Remove("shard-5")
	loadDistribution = distribute(clusters, consistentHashing)
	printLoadDistribution(loadDistribution)

	for i := clusterCount; i <= 2*clusterCount; i++ {
		cluster := fmt.Sprintf("%s-%d", prefix, i)
		clusters = append(clusters, cluster)
	}
	log.Printf("------------- %d shards and %d clusters -------------", len(consistentHashing.Servers()), len(clusters))
	loadDistribution = distribute(clusters, consistentHashing)
	printLoadDistribution(loadDistribution)

}

func distribute(clusters []string, c *consistent.Consistent) map[string]int {
	loadDistribution := map[string]int{}
	for _, cluster := range clusters {
		shard, _ := c.Get(cluster)
		log.Printf("cluster: %s managed by shard: %s", cluster, shard)
		loadDistribution[shard]++
	}
	return loadDistribution
}

func printLoadDistribution(distribution map[string]int) {
	for key, value := range distribution {
		log.Printf("shard: %s processes: %d clusters", key, value)
	}
}
