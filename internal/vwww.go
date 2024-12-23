package internal

import (
	"encoding/csv"
	"html/template"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"

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
	Id      string
	Targets []string
	Visited int
}

type VirtualWorldWideWeb struct {
	Pages []VirtualPage
	Seeds []string
}

func NewVWWW(nbPage int, nbSeed int) *VirtualWorldWideWeb {
	ids := make([]string, nbPage)
	for i := 0; i < nbPage; i++ {
		ids[i] = uuid.NewString()
	}

	pages := make([]VirtualPage, nbPage)
	for i := 0; i < nbPage; i++ {
		// DO NOT FORGET:
		//  - targets can be any page, including the current page.
		//  - there can be dupplicates in targets

		targets := randomSample(ids)
		targets = append(targets, ids[(i+1)%nbPage]) // Ensure the graph is cyclic
		pages[i] = VirtualPage{ids[i], targets, 0}
	}

	seeds := make([]string, nbSeed)
	for i := 0; i < nbSeed; i++ {
		seeds[i] = pages[rand.Intn(nbPage)].Id
	}

	return &VirtualWorldWideWeb{pages, seeds}
}

func NewVWWWFromCSV(directoryPath string) (*VirtualWorldWideWeb, error) {
	if !strings.HasPrefix(directoryPath, "/") {
		directoryPath = directoryPath + "/"
	}

	f1, err := os.Open(directoryPath + "pages")
	if err != nil {
		return nil, err
	}
	defer f1.Close()
	csv1 := csv.NewReader(f1)
	csv1.FieldsPerRecord = -1
	lines, err := csv1.ReadAll()
	if err != nil {
		return nil, err
	}
	pages := make([]VirtualPage, len(lines))
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		pages[i] = VirtualPage{line[0], append([]string{}, line[2:]...), 0}
	}

	f2, err := os.Open(directoryPath + "seeds")
	if err != nil {
		return nil, err
	}
	defer f2.Close()
	csv2 := csv.NewReader(f2)
	csv2.FieldsPerRecord = 1
	lines, err = csv2.ReadAll()
	if err != nil {
		return nil, err
	}
	seeds := make([]string, len(lines))
	for i := 0; i < len(lines); i++ {
		seeds[i] = lines[i][0]
	}

	return &VirtualWorldWideWeb{pages, seeds}, nil
}

func (vwww *VirtualWorldWideWeb) DumpCSV(directoryPath string) error {
	err := os.MkdirAll(directoryPath, 0o755)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(directoryPath, "/") {
		directoryPath = directoryPath + "/"
	}

	f1, err := os.Create(directoryPath + "pages")
	if err != nil {
		return err
	}
	defer f1.Close()
	csv1 := csv.NewWriter(f1)
	for _, page := range vwww.Pages {
		fields := append([]string{page.Id}, page.Targets...)
		err = csv1.Write(fields)
		if err != nil {
			return err
		}
	}
	csv1.Flush()

	f2, err := os.Create(directoryPath + "seeds")
	if err != nil {
		return err
	}
	defer f2.Close()
	csv2 := csv.NewWriter(f2)
	for _, seed := range vwww.Seeds {
		err = csv2.Write([]string{seed})
		if err != nil {
			return err
		}
	}
	csv2.Flush()
	return nil
}

func (vwww *VirtualWorldWideWeb) Serve() error {
	http.HandleFunc("/seeds", vwww.renderSeed)
	http.HandleFunc("/{id}", vwww.renderPage)
	http.HandleFunc("/", vwww.renderIndex)

	// Port 80 is necessary to be compatible with the crawler
	log.Println("Serving requests on http://127.0.0.1")
	return http.ListenAndServe(":80", nil)
}

func (vwww *VirtualWorldWideWeb) renderIndex(w http.ResponseWriter, req *http.Request) {
	ids := make([]string, len(vwww.Pages))
	for i := 0; i < len(vwww.Pages); i++ {
		ids[i] = vwww.Pages[i].Id
	}
	HTMLTemplate.Execute(w, ids)
	log.Printf("200 - GET /\n")
}

func (vwww *VirtualWorldWideWeb) renderSeed(w http.ResponseWriter, req *http.Request) {
	HTMLTemplate.Execute(w, vwww.Seeds)
	log.Printf("200 - GET /seeds\n")
}

func (vwww *VirtualWorldWideWeb) renderPage(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	if id == "" {
		w.WriteHeader(404)
		w.Write([]byte("Not Found"))
		log.Printf("404 - GET /%s\n", id)
	}

	for i := 0; i < len(vwww.Pages); i++ {
		if vwww.Pages[i].Id == id {
			HTMLTemplate.Execute(w, vwww.Pages[i].Targets)
			vwww.Pages[i].Visited += 1
			log.Printf("200 - GET /%s\n", id)
			return
		}
	}

	w.WriteHeader(404)
	w.Write([]byte("Not Found"))
	log.Printf("404 - GET /%s\n", id)
}

func randomSample[T any](data []T) []T {
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
