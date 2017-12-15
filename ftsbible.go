package main

import (
	"encoding/json"
	"fmt"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/analysis/lang/en"
	"github.com/blevesearch/bleve/analysis/tokenizer/whitespace"
	"github.com/blevesearch/bleve/mapping"
	"github.com/blevesearch/bleve/search"
	enx "github.com/blevesearch/blevex/lang/en"
	"github.com/googollee/go-socket.io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

//	"github.com/blevesearch/bleve/analysis/analyzer/simple"

type Verse struct {
	Id      string
	Book    int
	Chapter int
	Verse   int
	Text    string
}

type ByMatch []*search.DocumentMatch

func buildIndexMapping() (*mapping.IndexMappingImpl, error) {
	indexMapping := bleve.NewIndexMapping()

	var err error
	err = indexMapping.AddCustomAnalyzer("nonstopstem",
		map[string]interface{}{
			"type":         custom.Name,
			"char_filters": []interface{}{},
			"tokenizer":    whitespace.Name,
			"token_filters": []interface{}{
				enx.StemmerName,
			},
		})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return indexMapping, nil
}
func index() {

	file, e := ioutil.ReadFile("./bible.json")
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		os.Exit(1)
	}

	var bible []Verse
	json.Unmarshal(file, &bible)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultAnalyzer = en.AnalyzerName

	englishTextFieldMapping := bleve.NewTextFieldMapping()
	englishTextFieldMapping.Analyzer = en.AnalyzerName

	verseMapping := bleve.NewDocumentMapping()
	verseMapping.AddFieldMappingsAt("Text", englishTextFieldMapping)

	indexMapping.AddDocumentMapping("verse", verseMapping)
	indexMapping.DefaultType = "verse"

	index, err := bleve.New("bibleidx_en_v2.bleve", indexMapping)
	defer index.Close()
	if err != nil {
		panic(err)
	}

	i := 100
	for _, verse := range bible {
		fmt.Println(verse.Id)
		index.Index(verse.Id, verse)
		i -= 1
		if i == 0 {
			// break
		}
	}

}

func query(index bleve.Index, phrase string) []map[string]interface{} {
	var results []map[string]interface{}
	query := bleve.NewQueryStringQuery(phrase)
	//query := bleve.NewMatchQuery(phrase)
	//query.SetField("Text")

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Highlight = bleve.NewHighlight()
	searchRequest.Size = 20
	searchResult, _ := index.Search(searchRequest)
	fmt.Println(searchResult)
	if searchResult == nil {
		return results
	}
	for _, match := range searchResult.Hits {
		result := make(map[string]interface{})
		id := match.ID
		fragmentText := ""
		fragments := match.Fragments["Text"]
		if len(fragments) > 0 {
			fragmentText = fragments[0]
		}
		result["id"] = id
		result["text"] = fragmentText
		result["score"] = match.Score
		results = append(results, result)
		fmt.Println(id, fragmentText)
	}
	return results
}

func main() {
	// index()

	index, _ := bleve.Open("bibleidx_en_v2.bleve")
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	server.On("connection", func(so socketio.Socket) {
		log.Println("on connection")

		so.On("query:event", func(msg string) []map[string]interface{} {
			result := query(index, msg)
			return result
		})

		so.On("disconnection", func() {
			log.Println("on disconnect")
		})
	})
	server.On("error", func(so socketio.Socket, err error) {
		log.Println("error:", err)
	})

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./asset")))
	log.Println("Serving at localhost:5331...")
	log.Fatal(http.ListenAndServe(":5331", nil))

}
