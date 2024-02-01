package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func kyobo() {
	baseURL := "https://mkiosk.kyobobook.co.kr/kiosk/product/bookInfoInk.ink?site=%s&ejkGb=KOR&barcode=%s"
	shopNumberSlice := []string{"01", "58", "15", "23", "41", "66", "33", "72", "68", "36", "46", "74", "29", "90", "56", "49", "70", "52", "13", "47", "42", "25", "38", "69", "57", "59", "87", "04", "02", "05", "24", "45", "39", "77", "31", "28", "34", "48", "43"}
	// shopNameSlice := []string{"광화문", "가든파이브", "강남", "건대", "동대문", "신도림 디큐브", "목동", "서울대", "수유", "영등포", "은평", "이화여대", "잠실", "천호", "청량리", "합정", "광교", "광교월드 스퀘어", "부천", "분당", "송도", "인천", "일산", "판교", "평촌", "경성대ㆍ 부경대", "광주상무", "대구", "대전", "부산", "세종", "센텀시티", "울산", "전북대", "전주", "창원", "천안", "칠곡", "해운대 팝업 스토어"}
	bookBarcode := "9791192300818"

	for _, site := range shopNumberSlice {
		url := fmt.Sprintf(baseURL, site, bookBarcode)

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
			stockNumber := regexp.MustCompile("\\d+").FindString(strongTag.Text())
			if stockNumber != "" {
				fmt.Printf("%s의 도서 재고: %s\n", site, stockNumber)
			} else {
				fmt.Printf("%s의 재고 없음\n", site)
			}
		} else {
			fmt.Printf("%s에서의 태그 오류 또는 재고 정보 없음\n", site)
		}
	}
}

func yp_book() {
	// code := "101272360" //영풍문고만의 코드
	// price := "31000"

	bookURL := "https://www.ypbooks.co.kr/ypbooks_mobile/sub/mBranchStockLoc.jsp?bookCd=101272360&bookCost=31000"

	resp, err := http.Get(bookURL)
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

	storeStock := make(map[string]string)

	doc.Find("td.store").Each(func(_ int, element *goquery.Selection) {
		storeName := element.Find("strong").Text()
		stock := element.Find("span.stock").Text()
		storeStock[storeName] = stock
	})

	fmt.Println("영풍문고 각 가게의 재고 정보:")
	for store, stock := range storeStock {
		fmt.Printf("%s: %s\n", store, stock)
	}
}

func aladin() {
	code := "9791168473690"
	url := fmt.Sprintf("https://www.aladin.co.kr/search/wsearchresult.aspx?SearchTarget=UsedStore&KeyTag=&SearchWord=%s", code)

	// HTTP GET 요청 보내기
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// HTTP 응답 코드 확인
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error: %s", resp.Status)
	}

	// 응답 내용을 goquery.Document로 파싱
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// 가게별 재고 정보를 담을 맵 생성
	storesStock := make(map[string]string)

	// 각 가게의 정보를 파싱하여 맵에 저장
	doc.Find("a.usedshop_off_text3").Each(func(_ int, element *goquery.Selection) {
		storeName := strings.TrimSpace(element.Text())
		storesStock[storeName] = "1"
	})

	// 결과 출력
	fmt.Println("알라딘 각 가게의 재고 정보:")
	for store := range storesStock {
		fmt.Printf("%s: 1\n", store)
	}
}

func main() {
	fmt.Printf("======= 교보문고 =======\n")
	kyobo()
	fmt.Printf("======= 영풍문고 =======\n")
	yp_book()
	fmt.Printf("======= 알라딘 =======\n")
	aladin()
}
