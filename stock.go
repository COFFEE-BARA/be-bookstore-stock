package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/joho/godotenv"
)

type BookstoreInfo struct {
	Bookstore string
	Branch    string
	Stock     string
	Latitude  string
	Longitude string
}

type StockResult struct {
	KyoboStock  []BookstoreInfo
	YpbookStock []BookstoreInfo
	AladinStock []BookstoreInfo
}

type Location struct {
	Latitude  string
	Longitude string
}

// func main() {
// 	http.HandleFunc("/api/book/stock", getStockHandler)

// 	fmt.Println("서버를 시작합니다. http://localhost:8080")
// 	http.ListenAndServe(":8080", nil)
// }

// func getStockHandler(w http.ResponseWriter, r *http.Request) {
// 	isbn := r.URL.Query().Get("isbn")
// 	price := r.URL.Query().Get("price")

// 	kyoboStock, err := kyobo(isbn)
// 	if err != nil {
// 		return
// 	}
// 	ypbookStock, err := yp_book(isbn, price)
// 	if err != nil {
// 		return
// 	}

// 	aladinStock, err := aladin(isbn)
// 	if err != nil {
// 		return
// 	}

// 	fmt.Println("----------교보----------")
// 	fmt.Println(kyoboStock)
// 	fmt.Println("----------영풍----------")
// 	fmt.Println(ypbookStock)
// 	fmt.Println("----------알라딘----------")
// 	fmt.Println(aladinStock)
// }

func main() {
	lambda.Start(getStockHandler)
}

func getStockHandler(ctx context.Context, event map[string]string) (StockResult, error) {
	isbn := event["isbn"]
	price := event["price"]

	kyoboStock, err := kyobo(isbn)
	if err != nil {
		return StockResult{}, nil
	}
	ypbookStock, err := yp_book(isbn, price)
	if err != nil {
		return StockResult{}, nil
	}
	aladinStock, err := aladin(isbn)
	if err != nil {
		return StockResult{}, nil
	}

	stockResult := StockResult{
		KyoboStock:  kyoboStock,
		YpbookStock: ypbookStock,
		AladinStock: aladinStock,
	}

	fmt.Println("----------교보----------")
	fmt.Println(kyoboStock)
	fmt.Println("----------영풍----------")
	fmt.Println(ypbookStock)
	fmt.Println("----------알라딘----------")
	fmt.Println(aladinStock)

	return stockResult, nil
}

func kyobo(isbn string) ([]BookstoreInfo, error) {
	var result []BookstoreInfo
	kyoboNumberSlice := []string{"01", "58", "15", "23", "41", "66", "33", "72", "68", "36", "46", "74", "29", "90", "56", "49", "70", "52", "13", "47", "42", "25", "38", "69", "57", "59", "87", "04", "02", "05", "24", "45", "39", "77", "31", "28", "34", "48", "43"}
	kyoboNameSlice := []string{"광화문", "가든파이브", "강남", "건대", "동대문", "신도림 디큐브", "목동", "서울대", "수유", "영등포", "은평", "이화여대", "잠실", "천호", "청량리", "합정", "광교", "광교월드 스퀘어", "부천", "분당", "송도", "인천", "일산", "판교", "평촌", "경성대ㆍ 부경대", "광주상무", "대구", "대전", "부산", "세종", "센텀시티", "울산", "전북대", "전주", "창원", "천안", "칠곡", "해운대 팝업 스토어"}

	kyoboMap := make(map[string]string)
	for i, number := range kyoboNumberSlice {
		if i < len(kyoboNameSlice) {
			kyoboMap[number] = kyoboNameSlice[i]
		}
	}

	url := "https://mkiosk.kyobobook.co.kr/kiosk/product/bookInfoInk.ink?site=%s&ejkGb=KOR&barcode=%s"

	for site, branch := range kyoboMap {
		url := fmt.Sprintf(url, site, isbn)

		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Fatalf("Error: %s", resp.Status)
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		strongTag := doc.Find("strong:contains('재고 :')")
		if strongTag.Length() > 0 {
			stock := regexp.MustCompile("\\d+").FindString(strongTag.Text())
			if stock != "0" {
				locations := connectDynamodbAndImportLocation("교보문고", branch, isbn)
				if len(locations) == 0 {
					continue
				}

				latitude := locations[0].Latitude
				longitude := locations[0].Longitude

				bookstoreInfo := BookstoreInfo{
					Bookstore: "교보문고",
					Branch:    branch,
					Stock:     stock,
					Latitude:  latitude,
					Longitude: longitude,
				}
				result = append(result, bookstoreInfo)
			}
		} else {
			fmt.Printf("%s에서의 태그 오류 또는 재고 정보 없음\n", site)
		}
	}
	return result, nil
}

func yp_book(isbn string, price string) ([]BookstoreInfo, error) {
	var result []BookstoreInfo
	code := detailYP(isbn)

	url := fmt.Sprintf("https://www.ypbooks.co.kr/ypbooks_mobile/sub/mBranchStockLoc.jsp?bookCd=%s&bookCost=%s", code, price)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error: %s", resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	ypbookList := make(map[string]string)

	doc.Find("td.store").Each(func(_ int, element *goquery.Selection) {
		storeName := element.Find("strong").Text()
		stock := element.Find("span.stock").Text()
		ypbookList[storeName] = stock
	})

	for branch, stock := range ypbookList {
		if stock != "0" {
			locations := connectDynamodbAndImportLocation("영풍문고", branch, isbn)
			if len(locations) == 0 {
				continue
			}
			latitude := locations[0].Latitude
			longitude := locations[0].Longitude

			bookstoreInfo := BookstoreInfo{
				Bookstore: "영풍문고",
				Branch:    branch,
				Stock:     stock,
				Latitude:  latitude,
				Longitude: longitude,
			}
			result = append(result, bookstoreInfo)
		}
	}
	return result, nil
}

func detailYP(code string) string {
	url := "https://www.ypbooks.co.kr/ypbooks/search/requestAjaxSearchTab.jsp"

	data := strings.NewReader("query=" + code + "&collection=ALL&searchfield=ALL&showCnt=&sortField=RANK&notSoldOut=Y&catesearch=false&c1=&c2=&c3=&viewStyle=list&pageNum=1")

	req, err := http.NewRequest("POST", url, data)
	if err != nil {
		fmt.Println("Error creating request:", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
	}

	responseText := string(body)

	result := extractString(responseText, `<input\s+name="checkboxCartBook"\s+[^>]*value="([^"]*)"[^>]*>`)

	return result
}

func extractString(text, pattern string) string {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(text)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func aladin(isbn string) ([]BookstoreInfo, error) {
	var result []BookstoreInfo
	url := fmt.Sprintf("https://www.aladin.co.kr/search/wsearchresult.aspx?SearchTarget=UsedStore&KeyTag=&SearchWord=%s", isbn)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error: %s", resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	doc.Find("a.usedshop_off_text3").Each(func(_ int, element *goquery.Selection) {
		branch := strings.TrimSpace(element.Text())
		locations := connectDynamodbAndImportLocation("알라딘", branch, isbn)
		if len(locations) == 0 {
			return
		}
		latitude := locations[0].Latitude
		longitude := locations[0].Longitude

		bookstoreInfo := BookstoreInfo{
			Bookstore: "알라딘",
			Branch:    branch,
			Stock:     "있음",
			Latitude:  latitude,
			Longitude: longitude,
		}
		result = append(result, bookstoreInfo)
	})
	return result, nil
}

func connectDynamodbAndImportLocation(bookstore string, branch string, isbn string) []Location {
	loadEnv()

	sess, err := createNewSession()
	if err != nil {
		log.Println("Error creating session:", err)
		return []Location{}
	}

	result, err := scanDynamoDB(sess)
	if err != nil {
		log.Println(err)
		return []Location{}
	}

	location := bookstoreHandler(result, bookstore, branch, isbn)

	return location
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func createNewSession() (*session.Session, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	})
	if err != nil {
		return nil, fmt.Errorf("error scanning createNewSession: %v", err)
	}
	return sess, nil
}

func scanDynamoDB(sess *session.Session) (*dynamodb.ScanOutput, error) {
	svc := dynamodb.New(sess)
	tableName := os.Getenv("TABLE_NAME")

	input := &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	}

	result, err := svc.Scan(input)
	if err != nil {
		return nil, fmt.Errorf("error scanning table: %v", err)
	}

	return result, nil
}

func bookstoreHandler(result *dynamodb.ScanOutput, bookstore string, branch string, isbn string) []Location {
	var locations []Location
	for _, item := range result.Items {
		if *item["branch"].S == branch {
			latitude := *item["lati"].S
			longitude := *item["long"].S
			location := Location{
				Latitude:  latitude,
				Longitude: longitude,
			}
			locations = append(locations, location)
		}
	}
	return locations
}
