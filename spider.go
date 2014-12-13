package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
)

var (
	// 用于获取帖子信息
	getPage = regexp.MustCompile(`<tbody id="normalthread_[0-9]+">\n<tr>\n<td class="icn">\n(?:<[^>]+>\n)+</td>\n<th class="\w*">\n<em>\[<a[^>]*>([^<]+)</a>\]</em>\s*<a href="([^"]+)"[^>]*>([^<]+)</a>\n(?:<[^\n]+>\n){0,}?</th>\n<td class="by">\n<cite>\n<a[^>]*>([^<]+)</a></cite>\n<em>(?:<span class="xi1">)?<span(?: title="([^"]+)")?>([^<]+)(</span>){1,2}</em>\n</td>\n(?:<[^\n]+>\n){7}`)

	// 用于获取帖子列表总页数
	getMaxPagesNum = regexp.MustCompile(`<a href="[^"]+" class="last">\.\.\. ([0-9]+)</a>`)
)

func main() {
	fid := "139"
	resp, err := http.Get("http://www.mcbbs.net/forum.php?mod=forumdisplay&fid=" + fid + "&orderby=dateline&page=1")
	if err != nil {
		printError("GetPage", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		printError("ReadAll", err)
		os.Exit(1)
	}

	// 获取帖子列表总页数
	buf := getMaxPagesNum.FindStringSubmatch(string(body))
	maxPagesNum, err := strconv.Atoi(buf[1])
	if err != nil {
		printError("Atoi", err)
		os.Exit(1)
	}
	// 获取每一页的所有帖子
	for i := 0; i < maxPagesNum; i++ {
		pageList, err := getPagesList(fid, i+1)
		if err != nil {
			printError("getPagesList"+string(i), err)
			return
		}
		printInfo("List", *pageList)
	}
}

// 获取单页面的所有帖子
func getPagesList(fid string, pageNum int) (*[]pageInfo, error) {
	resp, err := http.Get("http://www.mcbbs.net/forum.php?mod=forumdisplay&fid=" + fid + "&orderby=dateline&page=" + strconv.Itoa(pageNum))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
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
	m := getPage.FindAllStringSubmatch(string(body), -1)
	var pageList = new([]pageInfo)

	for _, v := range m {
		date := v[5]
		// 如果发帖时间1为空，那么使用发帖时间2
		if date == "" {
			date = v[6]
		}
		pageInf := pageInfo{
			category: v[1],
			url:      v[2],
			title:    v[3],
			author:   v[4],
			date:     date,
		}
		*pageList = append(*pageList, pageInf)
	}

	return pageList, nil
}

func printInfo(s string, v ...interface{}) {
	log.Println("[INFO]", s, v)
}

func printError(s string, v ...interface{}) {
	log.Println("[ERROR]", s, v)
}

type pageInfo struct {
	category string
	url      string
	title    string
	author   string
	date     string
}
