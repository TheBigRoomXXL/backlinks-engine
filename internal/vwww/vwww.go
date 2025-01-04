package vwww

import (
	"context"
	"html/template"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
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

func NewVWWW(directoryPath string) *VirtualWorldWideWeb {
	return &VirtualWorldWideWeb{directoryPath: directoryPath}
}

func (vwww *VirtualWorldWideWeb) Serve(ctx context.Context) error {
	http.HandleFunc("/{id}", vwww.renderPage)
	http.HandleFunc("/", vwww.renderIndex)

	// Port 80 is necessary to be compatible with the crawler
	log.Println("Serving requests on http://127.0.0.1")
	go func() {
		err := http.ListenAndServe(":80", nil)
		log.Fatal(err)
	}()
	<-ctx.Done()
	return nil
}

func (vwww *VirtualWorldWideWeb) renderIndex(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Welcolme to the VirtualWorlWideWeb."))
}

func (vwww *VirtualWorldWideWeb) renderPage(w http.ResponseWriter, req *http.Request) {
	time.Sleep(100 * time.Microsecond)
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
	w.Header().Add("Content-Type", "test/html")
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
