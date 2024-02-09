import csv
import requests
from bs4 import BeautifulSoup
from geopy.geocoders import Nominatim
import re

geo_local = Nominatim(user_agent='South Korea')

# 위도, 경도 반환하는 함수
def geocoding(address):
    try:
        geo = geo_local.geocode(address)
        x_y = [geo.latitude, geo.longitude]
        return x_y

    except:
        return [0, 0]


headers = {
    'user-agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)',
}

url_base = 'https://www.kyobobook.co.kr/store?storeCode='

store_codes = [
    '001', '058', '015', '023', '041', '066', '033', '072', '068', '036',
    '046', '074', '029', '090', '056', '049', '070', '052', '013', '047',
    '042', '025', '038', '069', '057', '059', '087', '004', '002', '005',
    '024', '045', '039', '077', '031', '028', '034', '048', '043', '065'
]

store_name = [
    '광화문', '가든파이브', '강남', '건대스타시티', '동대문', '신도림디큐브시티', '목동', '서울대', '수유', '영등포',
    '은평', '이화여대', '잠실', '천호', '청량리', '합정', '광교', '광교월드스퀘어', '부천', '분당',
    '송도', '인천', '일산', '판교', '평촌', '경성대ㆍ 부경대', '광주상무', '대구', '대전', '부산',
    '세종', '센텀시티', '울산', '전북대', '전주', '창원', '천안', '칠곡', '해운대 팝업 스토어', '거제팝업스토어'
]

result_entries = []

for idx, store_code in enumerate(store_codes):
    url = f"{url_base}{store_code}"
    response = requests.get(url, headers=headers)

    result_entry = {
        "bookstore": "교보문고",
        "branch": store_name[idx],
        "address": "",
        "lati": 0,
        "long": 0
    }

    if response.status_code == 200:
        soup = BeautifulSoup(response.content, 'html.parser')

        all_span_elements = soup.find_all('span')

        for span_element in all_span_elements:
            if '매장주소' in span_element.get_text():
                info_desc_span = span_element.find_next('span', class_='info_desc')
                if info_desc_span:
                    info_desc_text = info_desc_span.get_text(strip=True)

                    if ',' in info_desc_text:
                        info_desc_text = info_desc_text.split(',')[0].strip()
                    else:
                        match = re.search(r'\d[^\d\W]*', info_desc_text)
                        if match:
                            info_desc_text = match.group()

                    result_entry["address"] = info_desc_text
                    result_entry["lati"], result_entry["long"] = geocoding(info_desc_text)

                    break

        result_entries.append(result_entry)

    else:
        print(f"Failed to retrieve data for Store Code: {store_code}. Status code: {response.status_code}")

for entry in result_entries:
    print(entry)

csv_file_path = 'result_entries.csv'
csv_header = ["bookstore", "branch", "address", "lati", "long"]

with open(csv_file_path, 'w', newline='', encoding='utf-8-sig') as csv_file:
    csv_writer = csv.DictWriter(csv_file, fieldnames=csv_header)
    csv_writer.writeheader()
    csv_writer.writerows(result_entries)

print(f"Results saved to {csv_file_path}")
