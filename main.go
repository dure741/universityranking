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

//设置国外排行榜网页网址中变化的量格式 http://www.compassedu.hk/qs_2019
var WorldUniversityRankings = []string{"guardian", "usnews", "the", "arwu", "times", "qs"}

//Rank 每一条国内数据元组
//可根据不同榜单进行添加
type Rank struct {
	ID                      uint    //id
	Brand                   string  //榜单名称
	Year                    int     //发布年份
	Rank                    int     //当年排名
	Name                    string  //高校名称
	Category                string  //高校归类
	TypeOfSchool            string  //办学类型
	Location                string  //高校地点
	RankInlocation          int     //高校在本地区排名 from校友会2017
	Score                   float64 //高校评分 from武书连榜单、校友会榜单
	Star                    int     //高校星级 from？
	Level                   string  //高校评级 from校友会榜单
	ScientificResearchScore float64 //科学研究评分
	TelentScore             float64 //人才培养评分
}

//WordRank
type WorldRank struct {
	//通用
	ID             uint   //id
	Brand          string //榜单
	Year           int    //年份
	Ranking        int    //当年排名
	UniversityName string //学校名称
	//Guardian
	GUARDIAN_GuardianScore         float64 //Guardian卫报综合分数
	GUARDIAN_SatisfiedWithCourse   float64 //Guardian 课程满意度
	GUARDIAN_SatisfiedWithTeaching float64 //Guardian	教学满意度
	GUARDIAN_SatisfiedWithFeedback float64 //Guatdian	反馈满意度
	GUARDIAN_StudentTaffRatio      float64 //Guardian	学生和职员比例
	GUARDIAN_SpendPerStudent       float64 //Guardian	学生平均花销
	GUARDIAN_EntryTariff           float64 //Guardian	入学标准
	GUARDIAN_ValueAddedScore       float64 //Guardian	加分
	GUARDIAN_CareerAfter6Mths      float64 //Guardian	6月后事业状况
	//Usnews
	USNEWS_TuitionAndFees                 float64 //学杂费用
	USNEWS_TotalEnrollment                float64 //总入学
	USNEWS_AcceptenceRate                 float64 //入学率
	USNEWS_SixYearGraduationRate          float64 //六年毕业率
	USNEWS_ClassesWithFewerThan20Students float64 //少于20人班级
	USNEWS_OverallScore                   float64 //总分
	//The
	THE_Location             string  //学校所在地址
	THE_Teaching             float64 //教学分数
	THE_InternationalOutlook float64 //国际化
	THE_Research             float64 //研究分数
	THE_Citations            float64 //引文
	THE_IndustryIncome       float64 //行业收入
	THE_OverallScore         float64 //the总分
	//Arwu
	ARWU_CountryRegion   string //学校所在国家/地区
	ARWU_NationalRanking int    //学校在国内排名
	ARWU_Alumni          float64
	ARWU_Award           float64 //奖学金
	ARWU_HiCi            float64
	ARWU_NandS           float64
	ARWU_PUB             float64
	ARWU_PCP             float64
	ARWU_TotalScore      float64
	//Times
	TIMES_TeachingQuality         float64 //教学质量
	TIMES_StudentSatisfaction     float64 //学生满意度
	TIMES_ResearchAssessment      float64
	TIMES_Ucasentry_Points        int
	TIMES_GraduateProspects       float64
	TIMES_FirstsTwotoOnes         float64
	TIMES_Completion              float64
	TIMES_StudentTaffRatio        float64
	TIMES_ServicesFacilitiesSpend int
	TIMES_TotalScore              int
	//Qs
	QS_CountryRegion         string
	QS_AcademicReputation    float64
	QS_EmployerReputation    float64
	QS_FacultyStudent        float64
	QS_InternationalFaculty  float64
	QS_InternationalStudents float64
	QS_CitationsPerFaculty   float64
	QS_OverallScore          float64
}

//main 函数
func main() {
	//连接数据库
	db, err = gorm.Open("mysql", dsn)
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
	//处理链接https://www.phb123.com/jiaoyu/gx/32517.html，标记为2019年
	crawlWsl2019("https://www.phb123.com/jiaoyu/gx/32517.html", 2019)
	//处理链接http://www.gaokao.com/e/20160111/569382097691b.shtml，标记为2016年
	crawlXyh2016("http://www.gaokao.com/e/20160111/569382097691b.shtml", 2016, 0)
	//处理链接https://www.phb123.com/jiaoyu/gx/3427.html，标记为2015年
	crawlXyh2015("https://www.phb123.com/jiaoyu/gx/3427.html", 2015, 0)
	//处理链接https://www.sohu.com/a/202060292_760366，标记为2012017
	crawlNetBig2017("https://www.sohu.com/a/202060292_760366", 2015, 0)
	// //至此国内排行榜大致结束

	db.Close()

	//连接数据库
	db, err = gorm.Open("mysql", dsn)
	//出错处理
	if err != nil {
		log.Fatal(err.Error())
	}
	//AutoMigrate 对指定模型运行自动迁移，只会添加缺少的字段，不会删除/更改当前数据
	db.AutoMigrate(&WorldRank{})
	//知晓当前年份，以方便爬取国外排行榜
	crawlWorldRanking("guardian", 2020)
	crawlWorldRanking("usnews", 2019)
	crawlWorldRanking("the", 2019)
	crawlWorldRanking("arwu", 2018)
	crawlWorldRanking("times", 2020)
	crawlWorldRanking("qs", 2020)

	//关闭数据库
	db.Close()

	//提示结束
	log.Println("end...")
}

//处理国外榜单url绑定 http://www.compassedu.hk
func crawlWorldRanking(brand string, latestyear int) {
	for i := latestyear; ; i-- {

		doc, err := fetchDoc("http://www.compassedu.hk/" + brand + "_" + strconv.Itoa(i))
		if err != nil {
			log.Println(err.Error())
			break
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
		if len(rows) == 0 {
			log.Println("Table is already empty!")
			break
		}

		//存入数据项
		for _, r := range rows {
			var item WorldRank
			item.Brand = brand
			item.Year = i
			item.Ranking, _ = strconv.Atoi(r[0])
			item.UniversityName = r[1]
			switch brand {
			case "guardian":
				item.GUARDIAN_GuardianScore = stringToFloat64(r[2])
				item.GUARDIAN_SatisfiedWithCourse = stringToFloat64(r[3])
				item.GUARDIAN_SatisfiedWithTeaching = stringToFloat64(r[4])
				item.GUARDIAN_SatisfiedWithFeedback = stringToFloat64(r[5])
				item.GUARDIAN_StudentTaffRatio = stringToFloat64(r[6])
				item.GUARDIAN_SpendPerStudent = stringToFloat64(r[7])
				item.GUARDIAN_EntryTariff = stringToFloat64(r[8])
				item.GUARDIAN_ValueAddedScore = stringToFloat64(r[9])
				item.GUARDIAN_CareerAfter6Mths = stringToFloat64(r[10])

			case "usnews":
				item.USNEWS_TuitionAndFees = stringToFloat64(r[2])
				item.USNEWS_TotalEnrollment = stringToFloat64(r[3])
				item.USNEWS_AcceptenceRate = stringToFloat64(r[4])
				item.USNEWS_SixYearGraduationRate = stringToFloat64(r[5])
				item.USNEWS_ClassesWithFewerThan20Students = stringToFloat64(r[6])
				item.USNEWS_OverallScore = stringToFloat64(r[7])

			case "the":
				item.THE_Location = r[2]
				item.THE_InternationalOutlook = stringToFloat64(r[3])
				item.THE_Research = stringToFloat64(r[4])
				item.THE_Citations = stringToFloat64(r[5])
				item.THE_IndustryIncome = stringToFloat64(r[6])
				item.THE_OverallScore = stringToFloat64(r[7])
			case "arwu":
				item.ARWU_CountryRegion = r[2]
				item.ARWU_NationalRanking, _ = strconv.Atoi(r[3])
				item.ARWU_Alumni = stringToFloat64(r[4])
				item.ARWU_Award = stringToFloat64(r[5])
				item.ARWU_HiCi = stringToFloat64(r[6])
				item.ARWU_NandS = stringToFloat64(r[7])
				item.ARWU_PUB = stringToFloat64(r[8])
				item.ARWU_PCP = stringToFloat64(r[9])
				item.ARWU_TotalScore = stringToFloat64(r[10])
			case "times":
				item.TIMES_TeachingQuality = stringToFloat64(r[2])
				item.TIMES_StudentSatisfaction = stringToFloat64(r[3])
				item.TIMES_ResearchAssessment = stringToFloat64(r[4])
				item.TIMES_Ucasentry_Points, _ = strconv.Atoi(r[5])
				item.TIMES_GraduateProspects = stringToFloat64(r[6])
				item.TIMES_FirstsTwotoOnes = stringToFloat64(r[7])
				item.TIMES_Completion = stringToFloat64(r[8])
				item.TIMES_StudentTaffRatio = stringToFloat64(r[9])
				item.TIMES_ServicesFacilitiesSpend, _ = strconv.Atoi(r[10])
				item.TIMES_TotalScore, _ = strconv.Atoi(r[11])
			case "qs":
				item.QS_CountryRegion = r[2]
				item.QS_AcademicReputation = stringToFloat64(r[3])
				item.QS_EmployerReputation = stringToFloat64(r[4])
				item.QS_FacultyStudent = stringToFloat64(r[5])
				item.QS_InternationalFaculty = stringToFloat64(r[6])
				item.QS_InternationalStudents = stringToFloat64(r[7])
				item.QS_CitationsPerFaculty = stringToFloat64(r[8])
				item.QS_OverallScore = stringToFloat64(r[9])
			}

			// save每一个元组
			// fmt.Println(&item)
			worldSave(&item)

		}

	}
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

//处理武书连榜单，捆绑于https://www.phb123.com/jiaoyu/gx/32517.html下页面
func crawlWsl2019(url string, year int) {
	//通过链接获取文档
	doc, err := fetchDoc(url)
	if err != nil {
		log.Println(err.Error())
	}
	//声明行切片
	var row []string
	//声明表切片
	var rows [][]string

	//本网页字段包含无法识别字符，创造字符串变量将其置换
	var deprecate string

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

	deprecate = strings.TrimRight(rows[0][0], "名次")
	for _, r := range rows[1:] {
		//初始数据库元组结构体
		var item Rank
		item.Brand = "wsl"
		item.Year = year
		item.Rank, _ = strconv.Atoi(strings.ReplaceAll(r[0], deprecate, ""))
		item.Name = strings.ReplaceAll(r[1], deprecate, "")
		item.Location = strings.ReplaceAll(r[2], deprecate, "")
		item.Category = strings.ReplaceAll(r[3], deprecate, "")
		item.ScientificResearchScore = stringToFloat64(strings.ReplaceAll(r[4], deprecate, ""))
		item.TelentScore = stringToFloat64(strings.ReplaceAll(r[5], deprecate, ""))
		item.Score = stringToFloat64(strings.ReplaceAll(r[6], deprecate, ""))
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

//校友会2016
func crawlXyh2016(url string, year int, tableIdx int) {
	doc, err := fetchDoc(url)
	if err != nil {
		log.Println(err.Error())
	}

	//本网页字段包含无法识别字符，创造字符串变量将其置换
	var deprecate string

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
	deprecate = strings.Split(rows[2][0], "1")[0]
	for _, r := range rows[2:] {
		var item Rank
		item.Brand = "xyh"
		item.Year = year
		item.Rank, _ = strconv.Atoi(strings.Trim(r[0], deprecate))
		item.Name = strings.Trim(r[1], deprecate)
		item.Category = strings.Trim(r[2], deprecate)
		item.Location = strings.Trim(r[3], deprecate)
		item.Score = stringToFloat64(strings.Trim(r[4], deprecate))
		item.TypeOfSchool = strings.Trim(r[5], deprecate)
		item.Star, _ = strconv.Atoi(strings.TrimRight(strings.Trim(r[6], deprecate), "星级"))
		item.Level = strings.Trim(r[7], deprecate)

		// save
		save(&item)
	}
}

//校友会2015
func crawlXyh2015(url string, year int, tableIdx int) {
	doc, err := fetchDoc(url)
	if err != nil {
		log.Println(err.Error())
	}

	//本网页字段包含无法识别字符，创造字符串变量将其置换
	var deprecate string

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
	deprecate = strings.Split(rows[2][0], "1")[0]
	for _, r := range rows[2:] {
		var item Rank
		item.Brand = "xyh"
		item.Year = year
		item.Rank, _ = strconv.Atoi(strings.Trim(r[0], deprecate))
		item.Name = strings.Trim(r[1], deprecate)
		item.Category = strings.Trim(r[2], deprecate)
		item.Location = strings.Trim(r[3], deprecate)
		item.RankInlocation, _ = strconv.Atoi(strings.Trim(r[4], deprecate))
		item.Score = stringToFloat64(strings.Trim(r[5], deprecate))
		item.TypeOfSchool = strings.Trim(r[6], deprecate)
		item.Star, _ = strconv.Atoi(strings.TrimRight(strings.Trim(r[7], deprecate), "星级"))
		item.Level = strings.Trim(r[8], deprecate)

		// save
		save(&item)
	}
}

//网大2017
func crawlNetBig2017(url string, year int, tableIdx int) {
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

	for _, r := range rows[2:] {
		var item Rank
		item.Brand = "NetBig"
		item.Year = year
		item.Rank, _ = strconv.Atoi(r[0])
		item.Name = r[1]
		item.Location = r[2]
		item.Star, _ = strconv.Atoi(strings.TrimRight(r[3], "星级"))
		item.Level = r[4]
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

func worldSave(worldrank *WorldRank) {
	//使用已连接的数据库插入一个元组
	db.Create(worldrank)
	log.Printf("saved item: %+v\n", worldrank)
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
		item.TypeOfSchool = r[5]
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
		item.TypeOfSchool = r[5]
		item.Star, _ = strconv.Atoi(strings.Trim(r[6], "星级"))
		item.Level = strings.TrimRight(r[7], "返回搜狐，查看更多")

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
