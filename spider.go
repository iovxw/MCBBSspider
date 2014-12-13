package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
)

var (
	// 用于获取帖子信息
	getPage = regexp.MustCompile(`<tbody id="normalthread_[0-9]+">\n<tr>\n<td class="icn">\n(?:<[^>]+>\n)+</td>\n<th class="new">\n<em>\[<a[^>]*>([^<]+)</a>\]</em>\s*<a href="([^"]+)"[^>]*>([^<]+)</a>\n(?:<[^\n]+>\n){0,}?</th>\n<td class="by">\n<cite>\n<a[^>]*>([^<]+)</a></cite>\n<em><span>([^<]+)</span></em>\n</td>\n(?:<[^\n]+>\n){7}`)

	// 用于获取帖子列表总页数
	getMaxPagesNum = regexp.MustCompile(`<a href="[^"]+" class="last">\.\.\. ([0-9]+)</a>`)

	// 用于判断时间格式
	day                = regexp.MustCompile(`[0-9]+-[0-9]+-[0-9]+`)
	yesterday          = regexp.MustCompile(`昨天.*`)
	dayBeforeYesterday = regexp.MustCompile(`前天.*`)
	xDaysAgo           = regexp.MustCompile(`([0-9]).*天前`)
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
		err = getPagesList(fid, i+1)
		if err != nil {
			printError("getPagesList"+string(i), err)
			return
		}
	}
}

// 获取单页面的所有帖子
func getPagesList(fid string, pageNum int) error {
	resp, err := http.Get("http://www.mcbbs.net/forum.php?mod=forumdisplay&fid=" + fid + "&orderby=dateline&page=" + strconv.Itoa(pageNum))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
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
		<em><span>m[5]发帖时间</span></em>
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

	for _, v := range m {
		//fmt.Println(v[1:])
		// 转换时间为标准时间格式
		switch {
		case day.MatchString(v[5]):
			printInfo("date", v[5])
		case yesterday.MatchString(v[5]):
			t := time.Unix(time.Now().Unix()-86400, 0)
			date := t.Format("2006-01-02")
			printInfo("yesterday", date)
		case dayBeforeYesterday.MatchString(v[5]):
			t := time.Unix(time.Now().Unix()-86400*2, 0)
			date := t.Format("2006-01-02")
			printInfo("day before yesterday", date)
		case xDaysAgo.MatchString(v[5]):
			x, err := strconv.Atoi(xDaysAgo.FindStringSubmatch(v[5])[1])
			if err != nil {
				printError("Time Format:", err)
				break
			}
			t := time.Unix(time.Now().Unix()-86400*int64(x), 0)
			date := t.Format("2006-01-02")
			printInfo(string(x)+" days ago", date)
		default:
			data := time.Now().Format("2006-01-02")
			printInfo("today", data)
		}
	}

	return nil
}

func printInfo(s string, v ...interface{}) {
	log.Println("[INFO]", s, v)
}

func printError(s string, v ...interface{}) {
	log.Println("[ERROR]", s, v)
}
