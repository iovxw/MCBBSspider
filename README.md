MCBBSspider
===========

mcbbs.net爬虫

数据储存结构
----------

数据库使用`Level DB`

数据文件放在执行目录下的`db/fid`文件夹内，其中的fid为论坛版块fid

每个论坛版块都是独立数据库文件（文件夹结构就是上面说的）

每个数据库中有一个key为`info`的数据

里面存放着数据结构为

```go
type forumInfo struct {
	Name         string
	PageNumber   int
	Introduction string
}
```

这样的经过golang的`gob`编码的`[]byte`

`Name`为版块名称

`PageNum`为版块分页数量

`Introduction`为版块介绍

然后有和版块分页数量相同的key为`page_x`的数据，x为分页

比如key为`page_1`里存放的就是版块分页第一页里面的所有帖子

储存的数据为经过`gob`编码的结构体切片

结构体的定义为

```go
type postInfo struct {
	Category string
	Url      string
	Title    string
	Author   string
	Date     string
	Body     string
}
```
