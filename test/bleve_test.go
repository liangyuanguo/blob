package test

import (
	"fmt"
	"github.com/blevesearch/bleve/v2"
	"log"
	"testing"
)

func TestBleve(t *testing.T) {

	//mapping := bleve.NewIndexMapping()
	index, err := bleve.Open("C:\\Users\\russionbear\\Documents\\GitHub\\aw\\blob\\meta-storage.bleve")
	if err != nil {
		log.Fatal(err)
	}

	// 添加文档到索引
	document := struct {
		Title string
		Doc   string
	}{
		Title: "Bleve 开源搜索引擎",
		Doc:   "2021-01-06",
	}
	err = index.Index("1", document)
	if err != nil {
		return
	}

	// 查询索引
	query := bleve.NewQueryStringQuery("Bleve")
	query2 := bleve.NewDateRangeStringQuery("2021-01-01", "2021-01-08")
	q := bleve.NewBooleanQuery()
	q.AddMust(query)
	q.AddMust(query2)

	searchRequest := bleve.NewSearchRequest(q)
	searchRequest.Fields = []string{"Title", "Doc"}
	searchRequest.From = 1
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(searchResult.Hits[0].Fields)
}
