package vwww

import (
	"fmt"
	"html/template"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var HTMLTemplate = template.Must(template.New("VirtualPage").Parse(`
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>Test Page</title>
  </head>
  <body>
    {{range .}}
		<div><a href="/{{.}}" > /{{.}} </a></div>
	{{end}}
  </body>
</html>
`))

type VirtualPage struct {
	Targets []string
}

type VirtualWorldWideWeb struct {
	directoryPath string
}

func GenerateVWWW(nbPage int, nbSeed int, directoryPath string) error {
	// 1. Prepare file structure
	err := os.MkdirAll(directoryPath, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create vwww directory: %w", err)
	}

	if !strings.HasPrefix(directoryPath, "/") {
		directoryPath = directoryPath + "/"
	}

	// 2. Generate page
	ids := make([]string, nbPage)
	for i := 0; i < nbPage; i++ {
		ids[i] = uuid.NewString()
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 64)
	defer close(semaphore)
	for i := 0; i < nbPage; i++ {
		wg.Add(1)
		semaphore <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-semaphore }()

			// 2.1 Prepare a file for each page
			pageFile, err := os.Create(directoryPath + ids[i])
			if err != nil {
				log.Printf("failed to create page file %s: %s", ids[i], err)
			}
			defer pageFile.Close()

			// 2.2 Generate targets
			cyclicId := ids[(i+1)%nbPage] // Ensure all nodes are connected
			targets := randomSample(ids)
			fields := append([]string{cyclicId}, targets...)
			_, err = pageFile.WriteString(strings.Join(fields, "\n"))
			if err != nil {
				log.Printf("failed to wrtie to page file %s: %s", ids[i], err)
			}
			err = pageFile.Sync()
			if err != nil {
				log.Printf("failed to sync page file %s: %s", ids[i], err)
			}
		}()
	}
	wg.Wait()

	// 3. Generate seeds
	seedFile, err := os.Create(directoryPath + "seeds")
	if err != nil {
		return fmt.Errorf("failed to create seeds file: %w", err)
	}
	defer seedFile.Close()
	for i := 0; i < nbSeed; i++ {
		seed := ids[rand.Intn(nbPage)]
		_, err = seedFile.WriteString(seed + "\n")
		if err != nil {
			return fmt.Errorf("failed to write to seed file: %w", err)
		}
	}
	err = seedFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync seed file: %w", err)
	}
	return nil
}

func NewVWWW(directoryPath string) *VirtualWorldWideWeb {
	return &VirtualWorldWideWeb{directoryPath: directoryPath}
}

func (vwww *VirtualWorldWideWeb) Serve() error {
	http.HandleFunc("/{id}", vwww.renderPage)
	http.HandleFunc("/", vwww.renderIndex)

	// Port 80 is necessary to be compatible with the crawler
	log.Println("Serving requests on http://127.0.0.1")
	return http.ListenAndServe(":80", nil)
}

func (vwww *VirtualWorldWideWeb) renderIndex(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Welcolme to the VirtualWorlWideWeb."))
}

func (vwww *VirtualWorldWideWeb) renderPage(w http.ResponseWriter, req *http.Request) {
	time.Sleep(20 * time.Microsecond)
	id := req.PathValue("id")
	if id == "" {
		w.WriteHeader(404)
		w.Write([]byte("Not Found"))
		return
	}
	content, err := os.ReadFile(vwww.directoryPath + id)
	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte("Not Found"))
		return
	}
	targets := strings.Split(string(content), "\n")
	HTMLTemplate.Execute(w, targets)
}

func randomSample[T any](data []T) []T {
	// DO NOT FORGET:
	//  - targets can be any page, including the current page.
	//  - there can be dupplicates in targets
	var alpha float64
	var max float64
	if len(data) <= 100_000 {
		alpha = 1.5
		max = float64(len(data)) / 5
	} else {
		alpha = 2.0
		max = float64(len(data)) / 10
	}

	nbSample := randomPowerLaw(alpha, 1, max)

	keys := make([]int, nbSample)
	for i := 0; i < nbSample; i++ {
		keys[i] = rand.Intn(len(data))
	}

	results := make([]T, nbSample)
	for i, key := range keys {
		results[i] = data[key]
	}

	return results
}

func randomPowerLaw(alpha float64, min float64, max float64) int {
	if min <= 0 {
		panic("min must be greater than 0 to ensure the power-law is well-defined")
	}
	u := rand.Float64()
	power := 1 - alpha
	x := math.Pow(u*(math.Pow(max, power)-math.Pow(min, power))+math.Pow(min, power), 1/power)
	return int(math.Round(x))
}
