package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"

	"github.com/dhconnelly/rtreego"
)

type Point struct {
	ID  string  `json:"id"`
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type ClusterOutput struct {
	CentroidLat float64  `json:"centroid_lat"`
	CentroidLon float64  `json:"centroid_lon"`
	IDs         []string `json:"ids"`
}

type rtreeItem struct {
	rect  rtreego.Rect
	index int
}

func (item rtreeItem) Bounds() rtreego.Rect {
	return item.rect
}

func main() {
	eps := flag.Float64("eps", 0.01, "epsilon distance (in degrees for Euclidean, or meters if using haversine)")
	minPts := flag.Int("minPts", 5, "minimum number of points per cluster")
	flag.Parse()

	points, err := readInput(os.Stdin)
	if err != nil {
		log.Fatalf("failed to read input: %v", err)
	}

	clusters := dbscan(points, *eps, *minPts)
	out, _ := json.Marshal(clusters)
	fmt.Println(string(out))
}

func readInput(r io.Reader) ([]Point, error) {
	var pts []Point
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var p Point
		if err := json.Unmarshal(scanner.Bytes(), &p); err != nil {
			return nil, err
		}
		pts = append(pts, p)
	}
	return pts, scanner.Err()
}

func dbscan(points []Point, eps float64, minPts int) []ClusterOutput {
	tree := rtreego.NewTree(2, 25, 50)
	for i, p := range points {
		rect := rtreego.Point{p.Lat, p.Lon}.ToRect(eps)
		tree.Insert(rtreeItem{rect: rect, index: i})
	}

	visited := make([]bool, len(points))
	var clusters [][]int
	for i := range points {
		if visited[i] {
			continue
		}
		visited[i] = true
		neighbors := regionQuery(points, tree, i, eps)
		if len(neighbors) < minPts {
			continue
		}
		cluster := []int{i}
		expandCluster(points, &cluster, neighbors, visited, tree, eps, minPts)
		clusters = append(clusters, cluster)
	}

	var result []ClusterOutput
	for _, cluster := range clusters {
		var sumLat, sumLon float64
		ids := make([]string, len(cluster))
		for j, idx := range cluster {
			sumLat += points[idx].Lat
			sumLon += points[idx].Lon
			ids[j] = points[idx].ID
		}
		n := float64(len(cluster))
		result = append(result, ClusterOutput{
			CentroidLat: sumLat / n,
			CentroidLon: sumLon / n,
			IDs:         ids,
		})
	}
	return result
}

func expandCluster(points []Point, cluster *[]int, neighbors []int, visited []bool, tree *rtreego.Rtree, eps float64, minPts int) {
	for j := 0; j < len(neighbors); j++ {
		idx := neighbors[j]
		if !visited[idx] {
			visited[idx] = true
			nbrs2 := regionQuery(points, tree, idx, eps)
			if len(nbrs2) >= minPts {
				neighbors = append(neighbors, nbrs2...)
			}
		}
		if !contains(*cluster, idx) {
			*cluster = append(*cluster, idx)
		}
	}
}

func regionQuery(points []Point, tree *rtreego.Rtree, idx int, eps float64) []int {
	rect := rtreego.Point{points[idx].Lat, points[idx].Lon}.ToRect(eps)
	candidates := tree.SearchIntersect(rect)

	var neighbors []int
	for _, obj := range candidates {
		item := obj.(rtreeItem)
		p := points[item.index]
		dx := points[idx].Lat - p.Lat
		dy := points[idx].Lon - p.Lon
		d := math.Hypot(dx, dy)
		if d <= eps {
			neighbors = append(neighbors, item.index)
		}
	}
	return neighbors
}

func contains(slice []int, val int) bool {
	for _, x := range slice {
		if x == val {
			return true
		}
	}
	return false
}
