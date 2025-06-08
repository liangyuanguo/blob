package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
	"liangyuanguo/aw/blob/pkg/model"
	"strconv"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
)

// BleveMetaStorage 实现 MetaStorage 接口
type BleveMetaStorage struct {
	index bleve.Index
}

func (b *BleveMetaStorage) Get(key string) (*model.Blob, error) {
	// 从索引中获取文档
	req := bleve.NewSearchRequest(query.NewDocIDQuery([]string{key}))
	req.Fields = []string{"*"}
	doc, err := b.index.Search(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %v", err)
	}
	if doc == nil {
		return nil, fmt.Errorf("document not found: %v", key)
	}

	var blobs []*model.Blob
	for _, hit := range doc.Hits {
		// 从存储的字段重建 Blob 对象
		blob, err := hitToBlob(hit)
		if err != nil {
			continue // 跳过错误文档
		}
		blobs = append(blobs, blob)
	}

	if len(blobs) == 0 {
		return nil, fmt.Errorf("document not found: %v", key)
	}

	return blobs[0], nil
}

func createBlobMapping() mapping.IndexMapping {
	// 创建索引映射
	indexMapping := bleve.NewIndexMapping()

	// 创建文档映射
	docMapping := bleve.NewDocumentMapping()

	// 配置 ID 字段 - 字符串索引但不全文索引
	idField := bleve.NewKeywordFieldMapping() // 使用 keyword 类型只做精确匹配
	idField.Name = "id"
	idField.Index = false
	idField.Store = false
	docMapping.AddFieldMappingsAt("id", idField)

	// 配置不被索引但可返回的字段
	unIndexedFields := []string{"contentType", "md5", "path", "size"}
	for _, field := range unIndexedFields {
		fm := bleve.NewTextFieldMapping()
		fm.Index = false
		fm.Store = true
		docMapping.AddFieldMappingsAt(field, fm)
	}

	// 配置其他字段的默认行为
	kwFields := []string{"authorId"}
	for _, field := range kwFields {
		fm := bleve.NewKeywordFieldMapping()
		fm.Index = true
		fm.Store = true
		docMapping.AddFieldMappingsAt(field, fm)
	}

	// 配置其他字段的默认行为
	indexedFields := []string{"name", "desc", "categories", "tags"}
	for _, field := range indexedFields {
		fm := bleve.NewTextFieldMapping()
		fm.Index = true
		fm.Store = true
		docMapping.AddFieldMappingsAt(field, fm)
	}

	// 特殊处理 UploadTime 为日期类型
	dateField := bleve.NewDateTimeFieldMapping()
	docMapping.AddFieldMappingsAt("uploadTime", dateField)

	// 特殊处理 Size 为数字类型 (虽然不被索引)
	sizeField := bleve.NewNumericFieldMapping()
	sizeField.Index = false
	sizeField.Store = true
	docMapping.AddFieldMappingsAt("size", sizeField)

	// 将文档映射添加到索引
	indexMapping.AddDocumentMapping("blob", docMapping)

	return indexMapping
}

// NewBleveMetaStorage 创建新的存储实例
func NewBleveMetaStorage(indexPath string) IMetaStorage {
	// 尝试打开现有索引
	index, err := bleve.Open(indexPath)
	if errors.Is(err, bleve.ErrorIndexPathDoesNotExist) {
		// 如果不存在则创建新索引
		indexMapping := createBlobMapping()
		index, err = bleve.New(indexPath, indexMapping)
		if err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	}

	return &BleveMetaStorage{index: index}
}

func (b *BleveMetaStorage) Put(v *model.Blob) error {
	return b.index.Index(v.ID, v)
}

// Delete 删除文档
func (b *BleveMetaStorage) Delete(key string) error {
	return b.index.Delete(key)
}

// Query 执行查询
func (b *BleveMetaStorage) Query(q map[string][2]string, offset, limit int) ([]*model.Blob, uint64, error) {
	// 创建复合查询
	var queries []query.Query

	for field, condition := range q {
		op := condition[0]    // 操作类型
		value := condition[1] // 值

		switch op {
		case "range":
			// 数值范围查询 (格式: "start,end")
			parts := strings.Split(value, ",")
			if len(parts) != 2 {
				return nil, 0, fmt.Errorf("invalid range format for field %s, expected 'start,end'", field)
			}
			min_, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			if err != nil {
				return nil, 0, fmt.Errorf("invalid min value for field %s: %v", field, err)
			}
			max_, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err != nil {
				return nil, 0, fmt.Errorf("invalid max value for field %s: %v", field, err)
			}

			rangeQ := bleve.NewNumericRangeQuery(&min_, &max_)
			rangeQ.SetField(field)
			queries = append(queries)

		case "kw":
			// 关键字精确匹配
			kwQ := bleve.NewTermQuery(value)
			kwQ.SetField(field)
			queries = append(queries, kwQ)

		case "text":
			// 全文搜索
			textQ := bleve.NewMatchQuery(value)
			textQ.SetField(field)
			queries = append(queries, textQ)

		case "geo":
			// 地理位置查询 (格式: "x,y,distance")
			parts := strings.Split(value, ",")
			if len(parts) != 3 {
				return nil, 0, fmt.Errorf("invalid geo format, expected 'x,y,distance'")
			}
			lon, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			if err != nil {
				return nil, 0, fmt.Errorf("invalid longitude: %v", err)
			}
			lat, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err != nil {
				return nil, 0, fmt.Errorf("invalid latitude: %v", err)
			}
			//distance, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
			//if err != nil {
			//	return nil, fmt.Errorf("invalid distance: %v", err)
			//}
			geoQ := bleve.NewGeoDistanceQuery(lon, lat, strings.TrimSpace(parts[2]))
			geoQ.SetField(field)
			queries = append(queries, geoQ)

		case "time":
			// 时间范围查询 (格式: "start,end")
			parts := strings.Split(value, ",")
			if len(parts) != 2 {
				return nil, 0, fmt.Errorf("invalid time range format for field %s, expected 'start,end'", field)
			}
			startTime, err := time.Parse(time.RFC3339, strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, 0, fmt.Errorf("invalid start time for field %s: %v", field, err)
			}
			endTime, err := time.Parse(time.RFC3339, strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, 0, fmt.Errorf("invalid end time for field %s: %v", field, err)
			}
			timeQ := bleve.NewDateRangeQuery(startTime, endTime)
			timeQ.SetField(field)
			queries = append(queries, timeQ)

		default:
			return nil, 0, fmt.Errorf("unsupported query operator: %s", op)
		}
	}

	var reqQ query.Query
	// 如果没有查询条件，返回空结果
	if len(queries) == 0 {
		reqQ = bleve.NewMatchAllQuery()
	} else {
		reqQ = bleve.NewConjunctionQuery(queries...)
	}

	// 创建搜索请求
	searchRequest := bleve.NewSearchRequest(reqQ)
	searchRequest.Fields = []string{"*"} // 返回所有字段
	searchRequest.Size = limit
	searchRequest.From = offset

	// 执行搜索
	searchResult, err := b.index.Search(searchRequest)
	if err != nil {
		return nil, 0, fmt.Errorf("search failed: %v", err)
	}

	// 转换结果
	var blobs []*model.Blob
	for _, hit := range searchResult.Hits {
		// 从存储的字段重建 Blob 对象
		blob, err := hitToBlob(hit)
		if err != nil {
			continue // 跳过错误文档
		}
		blobs = append(blobs, blob)
	}

	return blobs, searchResult.Total, nil
}

func hitToBlob(hit *search.DocumentMatch) (*model.Blob, error) {
	blob := &model.Blob{
		ID: hit.ID,
	}

	jStr, _ := json.Marshal(hit.Fields)
	err := json.Unmarshal(jStr, &blob)
	if err != nil {
		return nil, err
	}

	return blob, nil
}
