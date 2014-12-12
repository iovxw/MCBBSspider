package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
)

func main() {
	resp, err := http.Get("http://www.mcbbs.net/forum.php?mod=forumdisplay&fid=139&orderby=dateline&page=1")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	/*
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
	reg := regexp.MustCompile(`<tbody id="normalthread_[0-9]+">\n<tr>\n<td class="icn">\n(?:<[^>]+>\n)+</td>\n<th class="new">\n<em>\[<a[^>]*>([^<]+)</a>\]</em>\s*<a href="([^"]+)"[^>]*>([^<]+)</a>\n(?:<[^\n]+>\n){0,}?</th>\n<td class="by">\n<cite>\n<a[^>]*>([^<]+)</a></cite>\n<em><span>([^<]+)</span></em>\n</td>\n(?:<[^\n]+>\n){7}`)
	m := reg.FindAllStringSubmatch(string(body), -1)
	for _, v := range m {
		fmt.Println(v[1:])
	}
}
