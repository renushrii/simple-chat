package elasticsearch

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var EsPool = make(map[string]*Elasticsearch)

type Elasticsearch struct {
	Client *elasticsearch.Client
}

func GetEs(name string, host string) *Elasticsearch {
	if e, ok := EsPool[name]; ok {
		return e
	}
	e := CreateEs(host)
	EsPool[name] = e
	return e
}

func CreateEs(host string) *Elasticsearch {
	var err error
	url := strings.Split(host, ",")
	cfg := elasticsearch.Config{
		Addresses: url,
		Transport: &http.Transport{ //有很多选项，具体点击查看
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Second * 60,
			DialContext: (&net.Dialer{
				Timeout: time.Second * 10,
			}).DialContext,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS11,
			},
		},
	}
	es := &Elasticsearch{}
	es.Client, err = elasticsearch.NewClient(cfg)
	if err != nil {
		panic(err)
	}
	_, err = es.Client.Ping()
	if err != nil {
		panic(err)
	}
	return es
}

//CreateIndex 创建索引
func (e *Elasticsearch) CreateIndex(ctx context.Context) {
	mapping := `{
		"mappings":	{
			"properties": {
				"id": {
					"type": "integer"
				},
				"location": {
					"type": "geo_point"
				}
			}
		}
	}`
	req := esapi.IndicesCreateRequest{
		Index: "test_index",
		Body:  bytes.NewReader([]byte(mapping)),
	}

	res, _ := req.Do(ctx, e.Client)
	defer res.Body.Close()
	fmt.Println(res.String())
}

//查询，类似于mysql方式
func (e *Elasticsearch) Query(ctx context.Context) {
	query := map[string]interface{}{
		"query": "select id,location from test_index order by id desc limit 2", 
	}
	jsonBody, _ := json.Marshal(query)
	req := esapi.SQLQueryRequest{Body: bytes.NewReader(jsonBody)}
	res, _ := req.Do(ctx, e.Client)
	defer res.Body.Close()
	fmt.Println(res.String())
}

func (e *Elasticsearch) DeleteIndex(ctx context.Context) {
	req := esapi.IndicesDeleteRequest{
		Index: []string{"test_index"},
	}
	res, _ := req.Do(ctx, e.Client)
	defer res.Body.Close()
	fmt.Println(res.String())
}

func (e *Elasticsearch) InsertToEs(ctx context.Context) {
	body := map[string]interface{}{
		"id":       1,
		"location": map[string]float64{"lat": 3.1415, "lon": 110.2567},
	}
	jsonBody, _ := json.Marshal(body)

	req := esapi.CreateRequest{
		Index:      "test_index",
		DocumentID: "test_1",
		Body:       bytes.NewReader(jsonBody),
		Timeout:    5 * time.Second,
	}
	res, err := req.Do(ctx, e.Client)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()
	fmt.Println(res.String())

}

func (e *Elasticsearch) InsertBatch(ctx context.Context) {
	var bodyBuf bytes.Buffer
	for i := 2; i < 10; i++ {
		createLine := map[string]interface{}{
			"create": map[string]interface{}{
				"_index": "test_index",
				"_id":    "test_" + strconv.Itoa(i),
			},
		}
		jsonStr, _ := json.Marshal(createLine)
		bodyBuf.Write(jsonStr)
		bodyBuf.WriteByte('\n')

		body := map[string]interface{}{
			"id":       i,
			"location": map[string]float64{"lat": 3.1415, "lon": 110.2567},
		}
		jsonStr, _ = json.Marshal(body)
		bodyBuf.Write(jsonStr)
		bodyBuf.WriteByte('\n')
	}

	req := esapi.BulkRequest{
		Body: &bodyBuf,
	}
	res, _ := req.Do(ctx, e.Client)
	defer res.Body.Close()
	fmt.Println(res.String())
}

func (e *Elasticsearch) SelectBySearch(ctx context.Context) {
	query := `{"query" : {"bool" : {"must": [{"match_all" : {}}]}},"from" : 0,"size" : 2,"sort" : [{"id": "desc"}]}`

	req := esapi.SearchRequest{
		Index: []string{"test_index"},
		Body:  bytes.NewReader([]byte(query)),
	}
	res, _ := req.Do(ctx, e.Client)
	defer res.Body.Close()
	fmt.Println(res.String())
}

//根据id修改
func (e *Elasticsearch) UpdateSingle(ctx context.Context) {
	body := map[string]interface{}{
		"doc": map[string]interface{}{
			"location": map[string]float64{"lat": 3.5555, "lon": 110.66666},
		},
	}
	jsonBody, _ := json.Marshal(body)
	req := esapi.UpdateRequest{
		Index:      "test_index",
		DocumentID: "test_1",
		Body:       bytes.NewReader(jsonBody),
	}
	res, _ := req.Do(ctx, e.Client)
	defer res.Body.Close()
	fmt.Println(res.String())
}

//根据条件修改
func (e *Elasticsearch) UpdateByQuery(ctx context.Context) {
	body := map[string]interface{}{
		"script": map[string]interface{}{
			"lang": "painless",
			"source": `
                ctx._source.location = params.location;
                ctx._source.id = params.id;
            `,
			"params": map[string]interface{}{
				"location": map[string]float64{"lat": 3.9999, "lon": 110.8888},
				"id":       10,
			},
		},
		"query": map[string]interface{}{"term": map[string]interface{}{"id": 1}}, 
	}
	jsonBody, _ := json.Marshal(body)
	req := esapi.UpdateByQueryRequest{
		Index: []string{"test_index"},
		Body:  bytes.NewReader(jsonBody),
	}
	res, _ := req.Do(ctx, e.Client)
	defer res.Body.Close()
	fmt.Println(res.String())
}

func (e *Elasticsearch) DeleteSingle(ctx context.Context) {
	req := esapi.DeleteRequest{
		Index:      "test_index",
		DocumentID: "test_1",
	}
	res, _ := req.Do(ctx, e.Client)
	defer res.Body.Close()
	fmt.Println(res.String())
}

func (e *Elasticsearch) DeleteByQuery(ctx context.Context) {
	body := map[string]interface{}{
		"query": map[string]interface{}{"term": map[string]interface{}{"id": 2}}, 
	}
	jsonBody, _ := json.Marshal(body)
	req := esapi.DeleteByQueryRequest{
		Index: []string{"test_index"},
		Body:  bytes.NewReader(jsonBody),
	}
	res, _ := req.Do(ctx, e.Client)
	defer res.Body.Close()
	fmt.Println(res.String())
}
