/*
 * The search engine adapter for Antares
 * 
 *   https://golang.org/pkg/net/http/
 *
 */

package main

import (
  "fmt"
  "net/http"
  "html"
  "log"
  "github.com/schul-cloud/arix-search-adapter/arix"
  "bytes"
  "strconv"
  "net/url"
  "encoding/json"
  "io"
)


const PORT = 8080
const BASE = "/v1/search"
const SERVER = "http://arix.datenbank-bildungsmedien.net/"
const CONTEXT = "HH"
const SEARCH_LIMIT = 10
const SERVER_ID = "ARIX"

/*
 * See the example https://github.com/schul-cloud/resources-api-v1/blob/master/schemas/search-response/examples/valid/five-fictive-resources-null-links.json
 */
type SuccessfulSearchResponse struct {
  Jsonapi    Jsonapi         `json:"jsonapi"`
  Links      Links           `json:"links"`
  Data       []ResourceData  `json:"data"`
}

type Jsonapi struct {
  Version    string      `json:"version"`
  Meta       JsonapiMeta `json:"meta"`
}

type JsonapiMeta struct {
  Name         string    `json:"name"`
  Source       string    `json:"source"`
  Description  string    `json:"description"`
}

type Links struct {
  Self         SelfLink   `json:"self"`
  First        *NullLink  `json:"first"`
  Last        *NullLink   `json:"last"`
  Prev        *NullLink   `json:"prev"`
  Next        *NullLink   `json:"next"`
}

type SelfLink struct {
  Href         string        `json:"href"`
  Meta         SelfLinkMeta  `json:"meta"`
}

type SelfLinkMeta struct {
  Count        int  `json:"count"`
  Offset       int  `json:"offset"`
  Limit        int  `json:"limit"`
}

type NullLink struct {
}

type ResourceData struct {
  Type         string                 `json:"type"`
  Id           string                 `json:"id"`
  Links        ResourceLinks          `json:"links"`
  Attributes   arix.LearningResource  `json:"attributes"`
}

type ResourceLinks struct {
  Self         string         `json:"string"`  /* TODO: allow requesting single resources */
}

func NewSuccessfulSearchResponse(self string, limit int, offset int, resources []arix.LearningResource) SuccessfulSearchResponse {
  var data = []ResourceData{}
  for _, resource := range resources {
    data = append(data, ResourceData{
      Type: "resource",
      Id: fmt.Sprintf("%s-%s", SERVER_ID, resource.Id),
      Links: ResourceLinks{
        Self: "TODO",
      },
      Attributes: resource,
    })
  }
  return SuccessfulSearchResponse{
    Jsonapi: Jsonapi{
      Version: "1.0",
      Meta: JsonapiMeta{
        Name: "arix-search-adapter",
        Source: "https://github.com/schul-cloud/arix-search-adapter",
        Description: "This is a search adapter for Antares.",
      },
    },
    Links: Links{
      Self: SelfLink{
        Href: self,
        Meta: SelfLinkMeta{
          Count: len(resources),
          Offset: offset,
          Limit: SEARCH_LIMIT,
        },
      },
    },
    Data: data,
  }
}


func main() {
  fmt.Printf("Server is starting on port http://localhost:%d%s\n", PORT, BASE)
  http.HandleFunc(BASE, func(w http.ResponseWriter, r *http.Request) {
    /* parse the query */
    query := r.FormValue("q")  /* https://godoc.org/net/http#Request.FormValue */
    /* request content from anatares 
     *
     *  https://stackoverflow.com/a/19253970/1320237
     */
    data := url.Values{}
    data.Set("context", CONTEXT)
    data.Set("xmlstatement", arix.GetSearchRequest(SEARCH_LIMIT, query))
    encoded_data := data.Encode()

    client := &http.Client{}
    arix_search_request, _ := http.NewRequest("POST", SERVER, bytes.NewBufferString(encoded_data))
    arix_search_request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
    arix_search_request.Header.Add("Content-Length", strconv.Itoa(len(encoded_data)))
    
    arix_response, _ := client.Do(arix_search_request)
    found_resources := arix.ParseSearchResult(arix_response.Body)
    search_response := NewSuccessfulSearchResponse(r.URL.Path, SEARCH_LIMIT, 0, found_resources)
    
    w.Header().Set("Content-Type", "application/vnd.api+json") // from https://gist.github.com/tristanwietsma/8444cf3cb5a1ac496203#file-routes-go-L26
    result, _ := json.MarshalIndent(search_response, "", "  ")
    io.WriteString(w, string(result))
    fmt.Printf("Hello, %q \r\n %s:\r\n %s", html.EscapeString(r.URL.Path), arix_response.StatusCode, found_resources)
  })

  log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", PORT), nil))
}