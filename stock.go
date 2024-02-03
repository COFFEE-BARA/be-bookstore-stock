package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var isbnData [][]string
var kyoboStock [][]string
var ypbookStock [][]string
var aladinStock []string

func kyobo(isbn string) {
	kyoboNumberSlice := []string{"01", "58", "15", "23", "41", "66", "33", "72", "68", "36", "46", "74", "29", "90", "56", "49", "70", "52", "13", "47", "42", "25", "38", "69", "57", "59", "87", "04", "02", "05", "24", "45", "39", "77", "31", "28", "34", "48", "43"}
	kyoboNameSlice := []string{"광화문", "가든파이브", "강남", "건대", "동대문", "신도림 디큐브", "목동", "서울대", "수유", "영등포", "은평", "이화여대", "잠실", "천호", "청량리", "합정", "광교", "광교월드 스퀘어", "부천", "분당", "송도", "인천", "일산", "판교", "평촌", "경성대ㆍ 부경대", "광주상무", "대구", "대전", "부산", "세종", "센텀시티", "울산", "전북대", "전주", "창원", "천안", "칠곡", "해운대 팝업 스토어"}

	kyoboMap := make(map[string]string)
	for i, number := range kyoboNumberSlice {
		if i < len(kyoboNameSlice) {
			kyoboMap[number] = kyoboNameSlice[i]
		}
	}

	url := "https://mkiosk.kyobobook.co.kr/kiosk/product/bookInfoInk.ink?site=%s&ejkGb=KOR&barcode=%s"

	for site, store := range kyoboMap {
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
				kyoboStock = append(kyoboStock, []string{store, stock})
			}
		} else {
			fmt.Printf("%s에서의 태그 오류 또는 재고 정보 없음\n", site)
		}
	}
}

func yp_book(code string, price string) {
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

	for store, stock := range ypbookList {
		if stock != "0" {
			ypbookStock = append(ypbookStock, []string{store, stock})
		}
	}
}

func aladin(isbn string) {
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

	aladinList := make(map[string]string)

	doc.Find("a.usedshop_off_text3").Each(func(_ int, element *goquery.Selection) {
		storeName := strings.TrimSpace(element.Text())
		aladinList[storeName] = "1"
	})

	for store := range aladinList {
		aladinStock = append(aladinStock, store)
	}
}

func main() {
	kyobo("9791168473690")
	yp_book("101272360", "31000")
	aladin("9791168473690")
}
