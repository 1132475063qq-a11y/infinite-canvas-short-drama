package service

func graphReachable(adjacency map[string]map[string]struct{}, from string, to string, via []string) bool {
	points := append([]string{from}, via...)
	points = append(points, to)
	for i := 0; i < len(points)-1; i++ {
		if !reachable(adjacency, points[i], points[i+1]) {
			return false
		}
	}
	return true
}

func reachable(adjacency map[string]map[string]struct{}, from string, to string) bool {
	if from == to {
		return true
	}
	if _, ok := adjacency[from]; !ok {
		return false
	}
	queue := []string{from}
	visited := map[string]bool{from: true}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for next := range adjacency[current] {
			if next == to {
				return true
			}
			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}
	return false
}
