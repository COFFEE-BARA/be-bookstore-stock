# be-bookstore-stock

<br/>

# 💿 Dynamo DB table 구조도

| bookstore (PK) | branch (SK) | latitude | longitude |
| --- | --- | --- | --- |

<br/>

# 🤖 API 명세

- URL: BASE_URL/api/book/{책의 isbn 값}/bookstore
- Method: `GET`
- 기능 소개: 책의 재고가 존재하는 서점들의 위치를 알려주는 기능

<br/>

# 🗣️ Request

## ☝🏻Request Header

```
Content-Type: application/json
```

## ✌🏻Request Params

| Name | Type | Description | Required |
| --- | --- | --- | --- |
| 책의 { isbn 값 } | String | 책의 13자리 isbn 값 | Required |

<br/>

# 🗣️ Response

## ☝🏻Response Body

```json
{
  "code": 200,
  "message": "책의 재고 서점 리스트를 가져오는데 성공했습니다.",
  "data": {
    "isbn" : 9791140708116,
    "title" : "아는 만큼 보이는 백엔드 개발 (한 권으로 보는 백엔드 로드맵과 커리어 가이드)",
    "stockResult":{
      "kyoboStockList":null,
      "ypbookStockList":[
        {
          "type":"영풍문고",
          "name":"용산아이파크몰점",
           "stock":"3",
           "latitude":"37.529128500148",
           "longtitude":"126.9654885873"
        }
      ],
      "aladinStockList":[
        {
          "type":"알라딘",
          "name":"강남점",
          "stock":"있음",
          "latitude":"37.5016428104731",
          "longtitude":"127.026336096148"
        },{
          "type":"알라딘",
          "name":"건대점",
          "stock":"있음",
          "latitude":"37.5410629132034",
          "longtitude":"127.07085938264"
        }
      ]
    }
  }
}
```

## ✌🏻실패

1. 필요한 값이 없는 경우
    
    ```json
    {
      "code": 400,
      "message": "isbn값이 없습니다.",
      "data": null
    }
    ```
    
2. isbn 값에 매칭되는 책이 없을 경우
    
    ```json
    {
      "code": 404,
      "message": "없는 책입니다.",
      "data": null
    }
    ```
    
3. 서버에러
    
    ```json
    {
      "code": 500,
      "message": "서버 에러",
      "data": null
    }
    ```

<br/>

# 🏆 Tech Stack

## Programming Language

<img src="https://img.shields.io/badge/go-00ADD8?style=for-the-badge&logo=go&logoColor=white"/>


## DB

<img src="https://img.shields.io/badge/amazondynamodb-4053D6?style=for-the-badge&logo=amazondynamodb&logoColor=white"/> <img src="https://img.shields.io/badge/amazons3-569A31?style=for-the-badge&logo=amazons3&logoColor=white"/>


## CI/CD & Deploy

<img src="https://img.shields.io/badge/codebuild-68A51C?style=for-the-badge&logo=codebuild&logoColor=white"/> <img src="https://img.shields.io/badge/codepipeline-527FFF?style=for-the-badge&logo=codepipeline&logoColor=white"/> <img src="https://img.shields.io/badge/docker-2496ED?style=for-the-badge&logo=docker&logoColor=white"> <img src="https://img.shields.io/badge/awslambda-FF9900?style=for-the-badge&logo=awslambda&logoColor=white"/> <img src="https://img.shields.io/badge/amazonapigateway-FF4F8B?style=for-the-badge&logo=amazonapigateway&logoColor=white"/> <img src="https://img.shields.io/badge/ecr-FC4C02?style=for-the-badge&logo=ecr&logoColor=white"/>


## Develop Tool
 <img src="https://img.shields.io/badge/postman-FF6C37?style=for-the-badge&logo=postman&logoColor=white"> <img src="https://img.shields.io/badge/github-181717?style=for-the-badge&logo=github&logoColor=white"> <img src="https://img.shields.io/badge/git-F05032?style=for-the-badge&logo=git&logoColor=white"> 

## Communication Tool
<img src="https://img.shields.io/badge/slack-4A154B?style=for-the-badge&logo=slack&logoColor=white"> <img src="https://img.shields.io/badge/notion-000000?style=for-the-badge&logo=notion&logoColor=white">


<br/>

# 🏡 be-bookstore-stock architecture
<img width="329" alt="be-bookstore-stock-archi" src="https://github.com/COFFEE-BARA/be-bookstore-stock/assets/72396865/bbf81206-4394-41ce-a3de-083ef55ab137">
<br/> <br/>
1. 각 서점의 홈페이지에서 각 지점명과 주소 스크래핑 <br/>
2. 주소를 카카오 API를 통해 위도, 경도 좌표로 변환 <br/>
3. 수집한 정보를 csv 파일로 저장 <br/>
4. csv 파일을 S3에 업로드 <br/>
5. DynamoDB의 import from S3 기능 사용해 테이블 구축 <br/>
6. API Gateway에서 요청이 들어오면 Lambda 실행 <br/>
7. Lambda는 isbn 값을 이용해 서점의 홈페이지에서 책의 재고를 실시간으로 스크래핑
