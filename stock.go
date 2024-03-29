package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/joho/godotenv"
)

type Response struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    *ResponseData `json:"data"`
}

type ResponseData struct {
	Isbn        string      `json:"isbn"`
	Title       string      `json:"title"`
	StockResult StockResult `json:"stockResult"`
}
type StockResult struct {
	KyoboStockList  []BookstoreInfo `json:"kyoboStockList"`
	YpbookStockList []BookstoreInfo `json:"ypbookStockList"`
	AladinStockList []BookstoreInfo `json:"aladinStockList"`
}

type BookstoreInfo struct {
	Bookstore string `json:"type"`
	Branch    string `json:"name"`
	Stock     string `json:"stock"`
	Latitude  string `json:"latitude"`
	Longitude string `json:"longtitude"`
}

type Location struct {
	Latitude  string
	Longitude string
}

func main() {
	// 람다
	lambda.Start(getStockHandler)

	// //test~~~~~~~~~~~~~~~~~~~~~~~~~~
	// testEventFile, err := os.Open("test-event.json")
	// if err != nil {
	// 	log.Fatalf("Error opening test event file: %s", err)
	// }
	// defer testEventFile.Close()

	// // Decode the test event JSON
	// var testEvent events.APIGatewayProxyRequest
	// err = json.NewDecoder(testEventFile).Decode(&testEvent)
	// if err != nil {
	// 	log.Fatalf("Error decoding test event JSON: %s", err)
	// }

	// // Invoke the Lambda handler function with the test event
	// response, err := getStockHandler(context.Background(), testEvent)
	// if err != nil {
	// 	log.Fatalf("Error invoking Lambda handler: %s", err)
	// }

	// // Print the response
	// fmt.Printf("%v\n", response.StatusCode)
	// fmt.Printf("%v\n", response.Body)
}

// /api/book/${isbn}/bookstore?lat=${lat}&lon=${lon}
func getStockHandler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	//0. 환경변수 로드
	loadEnv()

	headers := map[string]string{
		"Access-Control-Allow-Origin":  "*", // 클라이언트 도메인
		"Access-Control-Allow-Headers": "Content-Type",
		"Access-Control-Allow-Methods": "OPTIONS,POST", // 허용되는 메서드
	}

	//1. url parameter 받아오기
	isbn := request.PathParameters["isbn"]
	fmt.Println("ISBN : ", isbn)

	//2. esCloud에서 책이름 가져오기
	esClient, err := connectElasticSearch(os.Getenv("CLOUD_ID"), os.Getenv("API_KEY"))
	if err != nil {
		fmt.Println("Error connecting to Elasticsearch:", err)
		return events.APIGatewayProxyResponse{StatusCode: 500, Headers: headers}, err
	}

	title, err := searchTitle(esClient, os.Getenv("INDEX_NAME"), os.Getenv("FIELD_NAME"), isbn)
	if err != nil {
		fmt.Println("인덱스 검색 중 오류 발생:", err)
		return events.APIGatewayProxyResponse{StatusCode: 500, Headers: headers}, err
	}

	//3. isbn 값으로 각 서점들 재고 찾아오기

	kyoboStock, err := kyobo(isbn)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 404, Headers: headers}, err
	}

	ypbookStock, err := yp_book(isbn)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 404, Headers: headers}, err
	}

	aladinStock, err := aladin(isbn)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 404, Headers: headers}, err
	}

	stockResult := StockResult{
		KyoboStockList:  kyoboStock,
		YpbookStockList: ypbookStock,
		AladinStockList: aladinStock,
	}

	fmt.Println("----------교보----------")
	fmt.Println(kyoboStock)
	fmt.Println("----------영풍----------")
	fmt.Println(ypbookStock)
	fmt.Println("----------알라딘----------")
	fmt.Println(aladinStock)

	jsonData, err := json.Marshal(Response{
		Code:    200,
		Message: "책의 재고 서점 리스트를 가져오는데 성공했습니다.",
		Data: &ResponseData{
			Isbn:        isbn,
			Title:       title,
			StockResult: stockResult,
		},
	})
	if err != nil {
		fmt.Println("JSON 인코딩 오류:", err)
		return events.APIGatewayProxyResponse{StatusCode: 500, Headers: headers}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    headers,
		Body:       string(jsonData),
	}, nil
}

func connectElasticSearch(CLOUD_ID, API_KEY string) (*elasticsearch.Client, error) {
	config := elasticsearch.Config{
		CloudID: CLOUD_ID,
		APIKey:  API_KEY,
	}

	es, err := elasticsearch.NewClient(config)
	if err != nil {
		fmt.Print(err)
		return nil, err
	}

	fmt.Print("엘라스틱 클라이언트 : ", es)

	// Elasticsearch 서버에 핑을 보내 연결을 테스트합니다.
	res, err := es.Ping()
	if err != nil {
		fmt.Println("Elasticsearch와 연결 중 오류 발생:", err)
		return nil, err
	}
	defer res.Body.Close()

	fmt.Println("Elasticsearch 클라이언트가 성공적으로 연결되었습니다.")

	return es, nil

}

func searchTitle(es *elasticsearch.Client, indexName, fieldName, value string) (string, error) {

	//검색 쿼리 작성
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				fieldName: value,
			},
		},
	}

	// 쿼리를 JSON으로 변환합니다.
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return "", err
	}

	// 검색 요청을 수행합니다.
	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex(indexName),
		es.Search.WithBody(bytes.NewReader(queryJSON)),
	)
	if err != nil {
		return "", err
	}

	// 검색 응답을 디코딩합니다.
	var searchResponse map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&searchResponse); err != nil {
		fmt.Println("검색 응답 디코딩 중 오류 발생:", err)
		return "", err
	}

	// 히트를 추출하고 후 저장
	hits := searchResponse["hits"].(map[string]interface{})["hits"].([]interface{})
	temp := hits[0].(map[string]interface{})["_source"].(map[string]interface{})

	return temp["Title"].(string), nil

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
			continue
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			log.Fatal(err)
			continue
		}

		strongTag := doc.Find("strong:contains('재고 :')")

		if strongTag.Length() > 0 {
			stock := regexp.MustCompile("\\d+").FindString(strongTag.Text())

			if stock != "" {
				locations, err := connectDynamodbAndImportLocation("교보문고", branch, isbn, stock)
				if err != nil {
					fmt.Printf("다이나모 디비 연결 및 장소 임포트 에러 : ", err)
				}
				if len(locations) == 0 {
					continue
				}

				latitude := locations[0].Latitude
				longitude := locations[0].Longitude

				bookstoreInfo := BookstoreInfo{
					Bookstore: "교보문고",
					Branch:    branch + "점",
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

type yp struct {
	Code string
	Isbn string
}

func yp_book(isbn string) ([]BookstoreInfo, error) {

	var result []BookstoreInfo

	data, err := detailYP(isbn)
	if err != nil {
		return []BookstoreInfo{}, nil
	}
	code := data[isbn][0]
	price := data[isbn][1]

	url := fmt.Sprintf("https://www.ypbooks.co.kr/ypbooks_mobile/sub/mBranchStockLoc.jsp?bookCd=%s&bookCost=%s", code, price)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("http Status 에러: %s", resp.Status)
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
			locations, err := connectDynamodbAndImportLocation("영풍문고", branch, isbn, stock)
			if err != nil {
				fmt.Printf("다이나모 디비 연결 및 장소 임포트 에러 : ", err)
			}
			if len(locations) == 0 {
				continue
			}
			latitude := locations[0].Latitude
			longitude := locations[0].Longitude

			bookstoreInfo := BookstoreInfo{
				Bookstore: "영풍문고",
				Branch:    branch + "점",
				Stock:     stock,
				Latitude:  latitude,
				Longitude: longitude,
			}
			result = append(result, bookstoreInfo)
		}
	}
	return result, nil
}

func detailYP(isbn string) (map[string][]string, error) {
	resultDict := make(map[string][]string)
	url := "https://www.ypbooks.co.kr/ypbooks/search/requestAjaxSearchTab.jsp"

	resultDict = make(map[string][]string)
	resultDict["query"] = []string{isbn}
	resultDict["collection"] = []string{"ALL"}
	resultDict["searchfield"] = []string{"ALL"}
	resultDict["showCnt"] = []string{""}
	resultDict["sortField"] = []string{"RANK"}
	resultDict["notSoldOut"] = []string{"Y"}
	resultDict["catesearch"] = []string{"false"}
	resultDict["c1"] = []string{""}
	resultDict["c2"] = []string{""}
	resultDict["viewStyle"] = []string{"list"}
	resultDict["pageNum"] = []string{"1"}

	resp, err := http.PostForm(url, resultDict)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	soup, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	checkbox := soup.Find("input[name='checkboxCartBook']")
	costSpan := soup.Find("span.cost")

	checkboxValue := ""
	if checkbox.Length() > 0 {
		checkboxValue, _ = checkbox.Attr("value")
	}

	costText := ""
	if costSpan.Length() > 0 {
		costText = costSpan.Text()
	}

	re := regexp.MustCompile(`[^0-9]`)
	costNumber := re.ReplaceAllString(costText, "")

	resultDict[isbn] = []string{checkboxValue, costNumber}

	return resultDict, nil
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

		locations, err := connectDynamodbAndImportLocation("알라딘", branch, isbn, "1")
		if err != nil {
			fmt.Printf("다이나모 디비 연결 및 장소 임포트 에러 : ", err)
		}
		if len(locations) != 0 {
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

		}

	})
	return result, nil
}

func connectDynamodbAndImportLocation(bookstore string, branch string, isbn string, stock string) ([]Location, error) {

	sess, err := createNewSession()
	if err != nil {
		log.Println("Error creating session:", err)
		return nil, err
	}

	result, err := scanDynamoDB(sess)
	if err != nil {
		fmt.Println("다이나모 디비 스캔에러 : ", err)
		return nil, err
	}

	location, err := bookstoreHandler(result, bookstore, branch, isbn, stock)
	if err != nil {
		fmt.Println("북스토어 핸들러 에러 : ", err)
		return nil, err
	}

	return location, nil
}

func loadEnv() {
	err := godotenv.Load(".env")
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

func bookstoreHandler(result *dynamodb.ScanOutput, bookstore string, branch string, isbn string, stock string) ([]Location, error) {
	var locations []Location

	for _, item := range result.Items {

		latitude := *item["lati"].S
		longitude := *item["long"].S

		if *item["branch"].S == branch && stock != "" {
			location := Location{
				Latitude:  latitude,
				Longitude: longitude,
			}
			locations = append(locations, location)
		}
	}
	return locations, nil
}

func calculateDistance(location Location, latitude string, longitude string) float64 {
	lat1, _ := strconv.ParseFloat(location.Latitude, 64)
	lon1, _ := strconv.ParseFloat(location.Longitude, 64)
	lat2, _ := strconv.ParseFloat(latitude, 64)
	lon2, _ := strconv.ParseFloat(longitude, 64)

	const earthRadius = 6371

	lat1 = lat1 * math.Pi / 180
	lon1 = lon1 * math.Pi / 180
	lat2 = lat2 * math.Pi / 180
	lon2 = lon2 * math.Pi / 180

	dlon := lon2 - lon1
	dlat := lat2 - lat1
	a := math.Pow(math.Sin(dlat/2), 2) + math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(dlon/2), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := earthRadius * c

	return distance
}
