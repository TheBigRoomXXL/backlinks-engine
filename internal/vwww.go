package internal

import (
	"bytes"
	"html/template"
	"log"
	"math/rand/v2"
	"net/http"

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
		<div><a href="{{.}}" > {{.}} </a></div>
	{{end}}
  </body>
</html>
`))

var vwww *VirtualWorldWideWeb

type VirtualPage struct {
	Id      string
	Html    string
	Visited int
}
type VirtualLink struct {
	From string
	To   string
}

func NewVirtualPage(id string, linksTo []string) *VirtualPage {
	var html bytes.Buffer
	HTMLTemplate.Execute(&html, linksTo)

	return &VirtualPage{
		Id:      id,
		Html:    html.String(),
		Visited: 0,
	}

}

type VirtualWorldWideWeb struct {
	Pages []VirtualPage
	Links []VirtualLink
	Seed  string
}

func NewVirtualWorldWideWeb(nbPage int) *VirtualWorldWideWeb {
	// Step 0: init Pages and Links
	var pages []VirtualPage
	var links []VirtualLink

	// Step 1: prepare a list of IDs
	ids := make([]string, nbPage)
	for i := 0; i < len(ids); i++ {
		ids[i] = uuid.NewString()
	}

	//  Step 2: for each page, generate some links
	for i := 0; i < len(ids); i++ {
		availableIds := copyAndRemove(ids, i)
		nbLinks := rand.IntN(nbPage)
		targets := make([]string, nbLinks)
		for j := 0; j < nbLinks; j++ {
			k := rand.IntN(len(availableIds))
			targets[j] = availableIds[k]
			availableIds = copyAndRemove(availableIds, k)
		}

		pageLinks := make([]VirtualLink, len(targets))
		for j := 0; j < len(targets); j++ {
			pageLinks[j] = VirtualLink{ids[i], targets[j]}
		}
		pages = append(pages, *NewVirtualPage(ids[i], targets))
		links = append(links, pageLinks...)
	}

	return &VirtualWorldWideWeb{
		Pages: pages,
		Links: links,
		Seed:  pages[0].Id,
	}
}

func copyAndRemove(slice []string, i int) []string {
	s := append([]string(nil), slice...) // Copy
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func ServeVWWW(virtualWorldWideWeb *VirtualWorldWideWeb) {
	vwww = virtualWorldWideWeb
	http.HandleFunc("/{id}", renderPage)
	http.HandleFunc("/", renderIndex)

	// Port 80 is necessary to be compatible with the crawler
	log.Println("Serving requests on http://127.0.0.1")
	err := http.ListenAndServe(":80", nil)
	log.Fatal(err)
}

func renderIndex(w http.ResponseWriter, req *http.Request) {
	ids := make([]string, len(vwww.Pages))
	for i := 0; i < len(vwww.Pages); i++ {
		ids[i] = "/" + vwww.Pages[i].Id
	}
	HTMLTemplate.Execute(w, ids)
	log.Printf("404 - GET /\n")
}

func renderPage(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	if id == "" {
		w.WriteHeader(404)
		w.Write([]byte("Not Found"))
		log.Printf("404 - GET /%s\n", id)
	}

	for _, page := range vwww.Pages {
		if page.Id == id {
			w.Write([]byte(page.Html))
			log.Printf("200 - GET /%s\n", id)
			return
		}
	}

	w.WriteHeader(404)
	w.Write([]byte("Not Found"))
	log.Printf("404 - GET /%s\n", id)
}
