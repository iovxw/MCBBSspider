package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
)

var (
	// 因为go的正则在匹配ReadAll出来的[]byte时有点奇怪，所以必须用[^\w]{0,}?来代替\n

	// 用于获取版块名称
	getForumName = regexp.MustCompile(`<h1 class="xs2">[^\w]{0,}?<a[^>]*>([^<]*)</a>`)
	// 用于获取版块帖子分页数量
	getForumPageNumber = regexp.MustCompile(`<a href="[^"]+" class="last">\.\.\. ([0-9]+)</a>`)
	// 用于获取版块介绍
	getForumIntroduction = regexp.MustCompile(`<div id="forum_rules_[0-9]*"[^>]*>([\w\W]{0,}?)(?:</div>[^\w]{0,}?){3}<div class="drag">`)
	// 用于获取帖子信息
	getPostInfo = regexp.MustCompile(`<tbody id="normalthread_[0-9]+">\n<tr>\n<td class="icn">\n(?:<[^>]+>\n)+</td>\n<th class="\w*">\n<em>\[<a[^>]*>([^<]+)</a>\]</em>\s*<a href="([^"]+)"[^>]*>([^<]+)</a>\n(?:<[^\n]+>\n){0,}?</th>\n<td class="by">\n<cite>\n<a[^>]*>([^<]+)</a></cite>\n<em>(?:<span class="xi1">)?<span(?: title="([^"]+)")?>([^<]+)(</span>){1,2}</em>\n</td>\n(?:<[^\n]+>\n){7}`)
	// 用于获取帖子内容
	getPostBody = regexp.MustCompile(`<div class="pcb">((?:.*\n){0,}?)<div id="comment_[0-9]*" class="cm">`)
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println("请在命令后添加论坛版块FID参数")
		fmt.Println("FID可从版块URL中寻找")
		fmt.Println("数据会保存在db/FID路径中")
		return
	}

	// 论坛版块fid
	fid := os.Args[1]

	resp, err := http.Get("http://www.mcbbs.net/forum.php?mod=forumdisplay&fid=" + fid + "&orderby=dateline&page=1")
	if err != nil {
		printError("http.Get", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		printError("ReadAll", err)
		os.Exit(1)
	}

	if bytes.Contains(body, []byte(`<p>抱歉，指定的版块不存在</p>`)) {
		printError("GetForum", "版块不存在，请检查PID是否正确")
		os.Exit(1)
	}

	// 获取版块名称
	n := getForumName.FindSubmatch(body)
	if len(n) == 0 {
		printError("GetForumName", "获取版块名称出错")
		os.Exit(1)
	}
	forumName := string(n[1])
	printInfo("版块名称", forumName)

	// 获取版块帖子分页数量
	n = getForumPageNumber.FindSubmatch(body)
	if len(n) == 0 {
		printError("GetForumPageNumber", "获取版块分页数量出错")
		os.Exit(1)
	}
	maxPagesNum, err := strconv.Atoi(string(n[1]))
	if err != nil {
		printError("Atoi", err)
		os.Exit(1)
	}
	printInfo("本版块全部分页数量", maxPagesNum)

	// 获取版块介绍
	n = getForumIntroduction.FindSubmatch(body)
	if len(n) == 0 {
		printError("GetForumIntroduction", "获取版块介绍出错")
		os.Exit(1)
	}
	forumIntroduction := string(n[1])
	print(forumIntroduction)

	// 创建数据库
	db, err := leveldb.OpenFile("db/"+fid, nil)
	if err != nil {
		printError("OpenDB", err)
		os.Exit(1)
	}
	defer db.Close()

	// 保存版块信息到数据库
	buf, err := encode(&forumInfo{
		Name:         forumName,
		PageNumber:   maxPagesNum,
		Introduction: forumIntroduction,
	})
	if err != nil {
		printError("EncodeForumInfo", err)
	}
	err = db.Put([]byte("info"), buf, nil)
	if err != nil {
		printError("PutForumInfo", err)
		os.Exit(1)
	}

	// 用于等待全部线程执行完毕
	var wg sync.WaitGroup
	// 用于统计还有多少页未完成
	var pageAmount int
	// 获取每一页的所有帖子
	for i := 0; i < maxPagesNum; i++ {
		wg.Add(1)
		pageAmount++
		go func(page int) {
			postList, err := getPagesList(fid, page)
			if err != nil {
				printError("getPagesList"+string(page), err)
			} else {
				// 获取每个帖子的内容
				for i, v := range postList {
					// for用于重试
					for {
						resp, err := http.Get("http://www.mcbbs.net/" + v.Url)
						if err != nil {
							printError("GetPost", err)
						} else {
							defer resp.Body.Close()

							// 检查服务器是否返回成功
							if resp.StatusCode != 200 {
								// 服务器错误
								printError("GetPost.ServerError", "服务器错误，错误码：", resp.StatusCode)
								if resp.StatusCode == 404 {
									// 如果为404，则没有重试的必要，跳出重试
									printError("GetPost.Retry", "帖子《"+v.Title+"》不存在")
									break
								}
							} else {
								body, err := ioutil.ReadAll(resp.Body)
								if err != nil {
									printError("GetPost.ReadAll", err)
								} else {
									n := getPostBody.FindSubmatch(body)
									// 检查是否获取body成功
									if len(n) == 0 {
										printError("GetPost.FindSubmatch", "未找到页面内文章部分")
									} else {
										postBody := string(n[1])
										// 存入Body
										postList[i].Body = postBody
										printInfo("GetPost", v.Title, "[OK]")
										// 跳出循环重试
										break
									}
								}
							}
						}
						// 获取失败，重试
						printError("GetPost.Retry", "获取帖子《"+v.Title+"》失败，正在重试")
					}
				}
			}
			// 编码
			byt, err := encode(postList)
			if err != nil {
				printError("Encode", err)
				return
			}
			// 存入数据库
			err = db.Put([]byte("page_"+strconv.Itoa(page)), byt, nil)
			if err != nil {
				printError("db.Put", err)
				return
			}

			pageAmount--

			printInfo("OK", "线程", page, "执行完毕")
			printInfo("OK", "版块分页", page, "中的所有帖子已经储存到本地，还有", pageAmount, "页正在下载中")
			wg.Done()
		}(i + 1)
	}
	wg.Wait()
	printInfo("OK", "FID为", fid, "的版块中的所有帖子已储存到本地")
}

// 获取单页面的帖子列表
func getPagesList(fid string, pageNum int) (pageList []postInfo, err error) {
	for {
		resp, err := http.Get("http://www.mcbbs.net/forum.php?mod=forumdisplay&fid=" + fid + "&orderby=dateline&page=" + strconv.Itoa(pageNum))
		if err != nil {
			printError("GetPageList", err)
		} else {
			defer resp.Body.Close()

			// 检查服务器是否返回成功
			if resp.StatusCode != 200 {
				// 服务器错误
				printError("GetPost.ServerError", "服务器错误，错误码：", resp.StatusCode)
			} else {
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					printError("GetPageList.ReadAll", err)
				} else {
					/*
						// getPage所匹配的信息
						// 获取到的参数已用“m[x]说明”标出
						<tbody id="normalthread_233212">
						<tr>
						<td class="icn">
						<a href="thread-372516-1-1.html" title="有新回复 - 新窗口打开" target="_blank">
						<img src="static/image/common/folder_new.gif" />
						</a>
						</td>
						<th class="new">
						<em>[<a href="forum.p899">m[1]分类</a>]</em> <a href="m[2]地址" style="" class="xst" >m[3]标题</a>
						<img src="static/image/filetype/image_s.gif" alt="attach_img" title="图片附件" align="absmiddle" />
						<img src="template/mcbbs/img/mc_agree.gif" align="absmiddle" alt="agree" title="帖子被加分" />
						<span class="tps">&nbsp;...<a href="">2</a><a href="">3</a><a href="html">4</a></span>
						<a href="forum.php?mod=redirect&amp;tid=374039&amp;goto=lastpost#lastpost" class="xi1">New</a>
						</th>
						<td class="by">
						<cite>
						<a href="home.php?mod=space&amp;uid=93634" c="1">m[4]作者</a></cite>
						<em><span class="xi1"><span title="m[5]发帖时间1">m[6]发帖时间2/span></span></em>
						</td>
						<td class="num"><a href="thread-369867-1-1.html" class="xi2">11</a><em>1189</em></td>
						<td class="by">
						<cite><a href="home9%AE" c="1">谢普</a></cite>
						<em><a href="forum.php?mopost"><span title="2014-12-7 14:07">5&nbsp;天前</span></a></em>
						</td>
						</tr>
						</tbody>
					*/
					m := getPostInfo.FindAllStringSubmatch(string(body), -1)

					// 检查是否获取成功
					if len(m) == 0 {
						printError("GetPageList.FindAllStringSubmatch", "未找到分页内帖子")
					} else {
						// 处理单个帖子信息
						for _, v := range m {
							date := v[5]
							// 如果发帖时间1为空，那么使用发帖时间2
							if date == "" {
								date = v[6]
							}
							// 储存帖子信息
							postInf := postInfo{
								Category: v[1],
								Url:      v[2],
								Title:    v[3],
								Author:   v[4],
								Date:     date,
							}
							pageList = append(pageList, postInf)
						}
						// 跳出重试
						break
					}
				}
			}
		}
		// 获取失败，重试
		printError("getPageList", "获取版块分页", pageNum, "失败，正在重试")
	}

	return pageList, nil
}

func printInfo(s string, v ...interface{}) {
	log.Println(append([]interface{}{"[INFO]", s + ":"}, v...)...)
}

func printError(s string, v ...interface{}) {
	log.Println(append([]interface{}{"[ERROR]", s + ":"}, v...)...)
}

func encode(data interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decode(data []byte, to interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(to)
}

type forumInfo struct {
	Name         string
	PageNumber   int
	Introduction string
}

type postInfo struct {
	Category string
	Url      string
	Title    string
	Author   string
	Date     string
	Body     string
}
