package deepdiff

import (
	"encoding/json"
	"testing"
)

func TestSometimesFails(t *testing.T) {
	left := `{"body":[],"bodyPath":"/ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn","commit":{"author":{"id":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"},"message":"created dataset","path":"/ipfs/QmT8nerkkyqyiUurCPApFF1XW29ouvKKeCCQg3zwt4hrnp","qri":"cm:0","signature":"I/nrDkgwt1IPtdFKvgMQAIRYvOqKfqm6x0qfpuJ14rEtO3+uPnY3K5pVDMWJ7K+pYJz6fyguYWgXHKkbo5wZl0ICVyoIiPa9zIVbqc1d6j1v13WqtRb0bn1CXQvuI6HcBhb7+VqkSW1m+ALpxhNQuI4ZfRv8Nm8MbEpL6Ct55fJpWX1zszJ2rQP1LcH2AlEZ8bl0qpcFMk03LENUHSt1DjlaApxrEJzDgAs5drfndxXgGKYjPpkjdF+qGhn2ALV2tC64I5aIn1SJPAQnVwprUr1FmVZjZcF9m9r8WnzQ6ldj29eZIciiFlT4n2Cbw+dgPo/hNRsgzn7Our2a6r5INw==","timestamp":"2001-01-01T01:01:01.000000001Z","title":"created dataset"},"meta":{"qri":"md:0","title":"example movie data"},"path":"/ipfs/QmVdDACqmUoFGCotChqSuYJMnocPwkXPifEB6kGqiTjhiL","peername":"me","qri":"ds:0","structure":{"checksum":"QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf","depth":2,"entries":8,"errCount":1,"format":"csv","formatConfig":{"headerRow":true,"lazyQuotes":true},"length":224,"qri":"st:0","schema":{"items":{"items":[{"title":"movie_title","type":"string"},{"title":"duration","type":"integer"}],"type":"array"},"type":"array"}},"viz":{"format":"html","qri":"vz:0","renderedPath":"/ipfs/QmXkN5J5yCAtF8GCxwRXARzAQhj3bPaSv1VHoyCCXzQRzN","scriptPath":"/ipfs/QmVM37PFzBcZn3qqKvyQ9rJ1jC8NkS8kYZNJke1Wje1jor"}}`
	rite := `{"body":[["Avatar ",178],["Pirates of the Caribbean: At World's End ",169],["Spectre ",148],["The Dark Knight Rises ",164],["Star Wars: Episode VII - The Force Awakens             ",""],["John Carter ",132],["Spider-Man 3 ",156],["Tangled ",100]],"bodyPath":"/ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn","commit":{"author":{"id":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"},"qri":"cm:0","timestamp":"0001-01-01T00:00:00Z","title":""},"meta":{"qri":"md:0","title":"different title"},"name":"test_ds","path":"/ipfs/QmVdDACqmUoFGCotChqSuYJMnocPwkXPifEB6kGqiTjhiL","peername":"me","previousPath":"/ipfs/QmVdDACqmUoFGCotChqSuYJMnocPwkXPifEB6kGqiTjhiL","qri":"ds:0","structure":{"checksum":"QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf","depth":2,"entries":8,"errCount":1,"format":"csv","formatConfig":{"headerRow":true,"lazyQuotes":true},"length":224,"qri":"st:0","schema":{"items":{"items":[{"title":"movie_title","type":"string"},{"title":"duration","type":"integer"}],"type":"array"},"type":"array"}},"viz":{"format":"html","qri":"vz:0","renderedPath":"/ipfs/QmXkN5J5yCAtF8GCxwRXARzAQhj3bPaSv1VHoyCCXzQRzN","scriptPath":"/ipfs/QmVM37PFzBcZn3qqKvyQ9rJ1jC8NkS8kYZNJke1Wje1jor"}}`

	var leftData interface{}
	var riteData interface{}
	err := json.Unmarshal([]byte(left), &leftData)
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal([]byte(rite), &riteData)
	if err != nil {
		t.Fatal(err)
	}

	expectA := `[{"type":"delete","path":"/commit/message","value":"created dataset"},{"type":"delete","path":"/commit/path","value":"/ipfs/QmT8nerkkyqyiUurCPApFF1XW29ouvKKeCCQg3zwt4hrnp"},{"type":"delete","path":"/commit/signature","value":"I/nrDkgwt1IPtdFKvgMQAIRYvOqKfqm6x0qfpuJ14rEtO3+uPnY3K5pVDMWJ7K+pYJz6fyguYWgXHKkbo5wZl0ICVyoIiPa9zIVbqc1d6j1v13WqtRb0bn1CXQvuI6HcBhb7+VqkSW1m+ALpxhNQuI4ZfRv8Nm8MbEpL6Ct55fJpWX1zszJ2rQP1LcH2AlEZ8bl0qpcFMk03LENUHSt1DjlaApxrEJzDgAs5drfndxXgGKYjPpkjdF+qGhn2ALV2tC64I5aIn1SJPAQnVwprUr1FmVZjZcF9m9r8WnzQ6ldj29eZIciiFlT4n2Cbw+dgPo/hNRsgzn7Our2a6r5INw=="},{"type":"insert","path":"/body/0","value":["Avatar ",178]},{"type":"insert","path":"/body/1","value":["Pirates of the Caribbean: At World's End ",169]},{"type":"insert","path":"/body/2","value":["Spectre ",148]},{"type":"insert","path":"/body/3","value":["The Dark Knight Rises ",164]},{"type":"insert","path":"/body/4","value":["Star Wars: Episode VII - The Force Awakens             ",""]},{"type":"insert","path":"/body/5","value":["John Carter ",132]},{"type":"insert","path":"/body/6","value":["Spider-Man 3 ",156]},{"type":"insert","path":"/body/7","value":["Tangled ",100]},{"type":"update","path":"/commit/timestamp","value":"0001-01-01T00:00:00Z","originalValue":"2001-01-01T01:01:01.000000001Z"},{"type":"update","path":"/commit/title","value":"","originalValue":"created dataset"},{"type":"update","path":"/meta/title","value":"different title","originalValue":"example movie data"},{"type":"insert","path":"/name","value":"test_ds"}]`
	// This version is returned about 6% of the time.
	expectB := `[{"type":"delete","path":"/commit/message","value":"created dataset"},{"type":"delete","path":"/commit/path","value":"/ipfs/QmT8nerkkyqyiUurCPApFF1XW29ouvKKeCCQg3zwt4hrnp"},{"type":"delete","path":"/commit/signature","value":"I/nrDkgwt1IPtdFKvgMQAIRYvOqKfqm6x0qfpuJ14rEtO3+uPnY3K5pVDMWJ7K+pYJz6fyguYWgXHKkbo5wZl0ICVyoIiPa9zIVbqc1d6j1v13WqtRb0bn1CXQvuI6HcBhb7+VqkSW1m+ALpxhNQuI4ZfRv8Nm8MbEpL6Ct55fJpWX1zszJ2rQP1LcH2AlEZ8bl0qpcFMk03LENUHSt1DjlaApxrEJzDgAs5drfndxXgGKYjPpkjdF+qGhn2ALV2tC64I5aIn1SJPAQnVwprUr1FmVZjZcF9m9r8WnzQ6ldj29eZIciiFlT4n2Cbw+dgPo/hNRsgzn7Our2a6r5INw=="},{"type":"insert","path":"","value":{"body":[["Avatar ",178],["Pirates of the Caribbean: At World's End ",169],["Spectre ",148],["The Dark Knight Rises ",164],["Star Wars: Episode VII - The Force Awakens             ",""],["John Carter ",132],["Spider-Man 3 ",156],["Tangled ",100]],"bodyPath":"/ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn","commit":{"author":{"id":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"},"qri":"cm:0","timestamp":"0001-01-01T00:00:00Z","title":""},"meta":{"qri":"md:0","title":"different title"},"name":"test_ds","path":"/ipfs/QmVdDACqmUoFGCotChqSuYJMnocPwkXPifEB6kGqiTjhiL","peername":"me","previousPath":"/ipfs/QmVdDACqmUoFGCotChqSuYJMnocPwkXPifEB6kGqiTjhiL","qri":"ds:0","structure":{"checksum":"QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf","depth":2,"entries":8,"errCount":1,"format":"csv","formatConfig":{"headerRow":true,"lazyQuotes":true},"length":224,"qri":"st:0","schema":{"items":{"items":[{"title":"movie_title","type":"string"},{"title":"duration","type":"integer"}],"type":"array"},"type":"array"}},"viz":{"format":"html","qri":"vz:0","renderedPath":"/ipfs/QmXkN5J5yCAtF8GCxwRXARzAQhj3bPaSv1VHoyCCXzQRzN","scriptPath":"/ipfs/QmVM37PFzBcZn3qqKvyQ9rJ1jC8NkS8kYZNJke1Wje1jor"}}}]`

	a := 0
	b := 0

	for k := 0; k < 1000; k++ {

		stat := Stats{}
		diff, err := Diff(leftData, riteData, OptionSetStats(&stat))
		if err != nil {
			t.Fatal(err)
		}

		actual, err := json.Marshal(diff)
		if err != nil {
			t.Fatal(err)
		}

		if string(actual) == expectA {
			a++
		} else if string(actual) == expectB {
			b++
		} else {
			t.Errorf("did not match!\nactual: %s\n", actual)
		}

	}

	if a > 0 && b > 0 {
		t.Errorf("non-deterministc result, got A %d times, B %d times", a, b)
	}
}
