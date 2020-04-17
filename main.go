package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/jinzhu/gorm"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

//初始化数据库连接信息
//dsn 数据连接信息
//db gorm框架中的数据库客户端结构体
//error 连接可能出现的错误
var (
	dsn = "root:Dr-741168669@tcp(dreamidea.top:3306)/universityranking?charset=utf8&parseTime=True&loc=Local"
	db  *gorm.DB
	err error
)

//Rank 每一条数据元组
//可根据不用榜单进行添加
type Rank struct {
	ID             uint    //id
	Brand          string  //榜单名称
	Year           int     //发布年份
	Rank           int     //当年排名
	Name           string  //高校名称
	Category       string  //高校归类
	Location       string  //高校地点
	RankInlocation int     //高校在本地区排名 from校友会2017
	Score          float64 //高校评分 from武书连榜单、校友会榜单
	Star           int     //高校星级 from？
	Level          string  //高校评级 from校友会榜单
}

//main 函数
func main() {
	//连接数据库
	db, err = gorm.Open("mysql", dsn)
	//延迟关闭数据库
	defer db.Close()
	//出错处理
	if err != nil {
		log.Fatal(err.Error())
	}
	//AutoMigrate 对指定模型运行自动迁移，只会添加缺少的字段，不会删除/更改当前数据
	db.AutoMigrate(&Rank{})

	//处理链接https://www.dxsbb.com/news/46702.html，标记未2018年
	crawlWsl("https://www.dxsbb.com/news/46702.html", 2018)
	//处理链接https://www.dxsbb.com/news/5463.html，标记为2019年
	crawlXyh("https://www.dxsbb.com/news/5463.html", 2019, 1)
	//处理链接https://www.dxsbb.com/news/1383.html，标记为2018年
	crawlXyh("https://www.dxsbb.com/news/1383.html", 2018, 0)
	//处理链接https://www.dxsbb.com/news/1383.html，标记为2018年，此为第二页
	crawlXyh("https://www.dxsbb.com/news/1383_2.html", 2018, 0)
	//处理链接https://www.sohu.com/a/126283593_356902，标记为2018年
	crawlXyh2017start("https://www.sohu.com/a/126283593_356902", 2017, 0)
	for i := 1; i <= 6; i++ {
		crawlXyh2017("https://www.sohu.com/a/126283593_356902", 2017, i)
	}

	//提示结束
	log.Println("end...")
}

//处理武书连榜单，捆绑于https://www.dxsbb.com/news下页面
func crawlWsl(url string, year int) {
	//通过链接获取文档
	doc, err := fetchDoc(url)
	if err != nil {
		log.Println(err.Error())
	}
	//声明行切片
	var row []string
	//声明表切片
	var rows [][]string

	//在doc中查找“tr”节点
	doc.Find("tr").Each(func(i int, tr *goquery.Selection) {
		//tr节点中查找“td”节点
		tr.Find("td").Each(func(j int, td *goquery.Selection) {
			//将结果存入行切片
			row = append(row, td.Text())
		})
		if len(row) != 0 {
			//将行存入表切片
			rows = append(rows, row)
		}
		//清空行切片
		row = nil
	})

	for _, r := range rows {
		//初始数据库元组结构体
		var item Rank
		item.Brand = "wsl"
		item.Year = year
		item.Rank, _ = strconv.Atoi(r[0])
		item.Name = r[1]
		item.Category = r[2]
		item.Location = r[3]
		item.Score = stringToFloat64(r[4])
		// save每一个元组
		save(&item)
	}
}

//处理校友会榜单，捆绑于https://www.dxsbb.com/news下页面
func crawlXyh(url string, year int, tableIdx int) {
	doc, err := fetchDoc(url)
	if err != nil {
		log.Println(err.Error())
	}

	var row []string
	var rows [][]string
	doc.Find("table").Each(func(i int, ta *goquery.Selection) {
		if i == tableIdx {
			ta.Find("tr").Each(func(i int, tr *goquery.Selection) {
				tr.Find("td").Each(func(j int, td *goquery.Selection) {
					row = append(row, td.Text())
				})
				if len(row) != 0 {
					rows = append(rows, row)
				}
				row = nil
			})
		}
	})

	for _, r := range rows[1:] {
		var item Rank
		item.Brand = "xyh"
		item.Year = year
		item.Rank, _ = strconv.Atoi(r[0])
		item.Name = r[1]
		item.Score = stringToFloat64(r[2])
		if year == 2019 {
			item.Level = r[3]
		}
		if year == 2018 {
			starNum := strings.TrimRight(r[3], "星级")
			item.Star, _ = strconv.Atoi(starNum)
			item.Level = r[4]
		}

		// save
		save(&item)
	}
}

//save 将rank元组存入数据库
func save(rank *Rank) {
	//使用已连接的数据库插入一个元组
	db.Create(rank)
	log.Printf("saved item: %+v\n", rank)
}

//fetchDoc 抓取文档
//输入：目标URL
//输出：goquery.Document结构体，错误
func fetchDoc(url string) (*goquery.Document, error) {
	log.Printf("crwaling url: %s", url)
	// Request the HTML page.
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}
	// Transform
	encodingmethod := determineEncoding(res.Body)
	utf8Reader := transform.NewReader(res.Body, encodingmethod.NewDecoder())
	// Load the HTML document
	return goquery.NewDocumentFromReader(utf8Reader)
}

//stringToFloat64 将字符串转换为数值
func stringToFloat64(s string) float64 {
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return value
}

//crawlxyh2017 绑定链接https://www.sohu.com/a/126283593_356902
func crawlXyh2017start(url string, year int, tableIdx int) {
	doc, err := fetchDoc(url)
	if err != nil {
		log.Println(err.Error())
	}

	var row []string
	var rows [][]string
	doc.Find("table").Each(func(i int, ta *goquery.Selection) {
		if i == tableIdx {
			ta.Find("tr").Each(func(i int, tr *goquery.Selection) {
				tr.Find("td").Each(func(j int, td *goquery.Selection) {
					row = append(row, td.Text())
				})
				if len(row) != 0 {
					rows = append(rows, row)
				}
				row = nil
			})
		}
	})

	for _, r := range rows[1:] {
		var item Rank
		item.Brand = "xyh"
		item.Year = year
		item.Rank, _ = strconv.Atoi(r[0])
		item.Name = r[1]
		//item.Score = stringToFloat64(r[2])
		item.Location = r[2]
		item.RankInlocation, _ = strconv.Atoi(r[3])
		item.Score = stringToFloat64(r[4])
		item.Category = r[5]
		item.Star, _ = strconv.Atoi(strings.Trim(r[6], "星级"))
		item.Level = r[7]

		// save
		save(&item)
	}
}
func crawlXyh2017(url string, year int, tableIdx int) {
	doc, err := fetchDoc(url)
	if err != nil {
		log.Println(err.Error())
	}

	var row []string
	var rows [][]string
	doc.Find("table").Each(func(i int, ta *goquery.Selection) {
		if i == tableIdx {
			ta.Find("tr").Each(func(i int, tr *goquery.Selection) {
				tr.Find("td").Each(func(j int, td *goquery.Selection) {
					row = append(row, td.Text())
				})
				if len(row) != 0 {
					rows = append(rows, row)
				}
				row = nil
			})
		}
	})

	for _, r := range rows[:] {
		var item Rank
		item.Brand = "xyh"
		item.Year = year
		item.Rank, _ = strconv.Atoi(r[0])
		item.Name = r[1]
		//item.Score = stringToFloat64(r[2])
		item.Location = r[2]
		item.RankInlocation, _ = strconv.Atoi(r[3])
		item.Score = stringToFloat64(r[4])
		item.Category = r[5]
		item.Star, _ = strconv.Atoi(strings.Trim(r[6], "星级"))
		item.Level = r[7]

		// save
		save(&item)
	}
}

//查看网页的编码
func determineEncoding(r io.Reader) encoding.Encoding {
	bytes, err := bufio.NewReader(r).Peek(1024)
	if err != nil {
		panic(err)
	}
	e, _, _ := charset.DetermineEncoding(bytes, "")
	return e
}
