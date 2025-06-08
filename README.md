# blob
简易文件上传、下载和检索服务



查询指令
```bash
# http://host?q[Field]=COMMAND=VALUE
curl http://localhost?q[size]=range=1,100&q[name]=kw=kw&q[desc]=text=kw
```

## Features
- 支持模式
  - local 本地文件系统
  - s3    s3或minio
  - none  静态服务器，属于基础功能
- 支持JWT无状态认证，用于服务间调用，可开关
- 依赖bleve 存储检索数据，适合中小型项目，大型项目还是用quickwit、es
- 数据与元数据分离，元数据结构
  ```go
  type Blob struct {
      ID   string `json:"id"`
      Name string `json:"name"`
      Desc string `json:"desc"`
  
      Size        int64     `json:"size"`
      Path        string    `json:"path"`
      UploadTime  time.Time `json:"uploadTime"`
      MD5         string    `json:"md5,omitempty"`
      ContentType string    `json:"contentType,omitempty"`
  
      AuthorId string `json:"authorId"`
      IsPublic bool   `json:"isPublic"`
      
      Categories string `json:"categories" `
      Tags       string `json:"tags" `
  }
  ```

## TODO
> 这几个用不上就先不管了
- [ ] s3 测试
- [ ] 服务间jwt认证测试
- [ ] 依据IsPublic 拦截