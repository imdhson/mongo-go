package modules

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	BATCHSIZE = 100
)

func will_send_append(dbres *Dj_jobs_detail, input *Dj_jobs_detail_s, score int) {
	var tmp bool = false
	for i, v := range *input {
		//log.Println(v.ID, dbres.ID, v.ID == dbres.ID)
		if v.ID == dbres.ID { //will_send에 이미 포함 되어있는 데이터일때
			tmp = true
			log.Println("!!!!!!이미 포함됨", v.ID, dbres.ID)
			log.Println(v.AI_List_score, dbres.AI_List_score)
			//v.AI_List_score += score //포인터 변수가 잘 수정되는지 확인 필요
			(*input)[i].AI_List_score += score //포인터 타고가서 실제값 수정 성공
			return
		} else { //포함되지 않았을 때 dbres를 append함
			log.Println(v.AI_List_score, dbres.AI_List_score)
			tmp = false
		}
	}
	if !tmp { //포함되지 않았을 때 dbres를 append함
		log.Println("어펜드 시도", dbres.ID)
		(*dbres).AI_List_score += score
		*input = append(*input, *dbres)
	}
}

func contains(input []string, v string) bool {
	if len(input) <= 1 { //빈칸일경우 무조건 true 반환
		return true
	}
	for _, v2 := range input {
		if v2 == v {
			return true
		}
	}
	return false
}
func type_inters(t1 string, t2 string, t3 string) []string { //장애유형을 받아서 교집합만 배열로 반환해줌
	var rst []string
	t1 = strings.ReplaceAll(t1, " ", "") // space가 있으면 소거
	t2 = strings.ReplaceAll(t2, " ", "")
	t3 = strings.ReplaceAll(t3, " ", "")
	t1_splited := strings.Split(t1, ",") //,를 기준으로 split
	if len(t1_splited) <= 1 {            //1번 마저도 빈칸이면 모든 부위 사용 가능으로 간주함
		t1_splited = []string{"팔", "다리", "시각", "음성", "귀"}
	}
	t2_splited := strings.Split(t2, ",")
	t3_splited := strings.Split(t3, ",")

	for _, v := range t1_splited {
		if contains(t2_splited, v) && contains(t3_splited, v) {
			rst = append(rst, v)
		}
	}
	return rst
}
func AIListSender(w http.ResponseWriter, r *http.Request) { //메인화면 직장 리스트
	err := godotenv.Load()
	Critical(err)
	URI := os.Getenv("MONGODB_URI")
	if URI == "" {
		Critical(err)
	}
	db, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(URI))
	Critical(err)
	defer func() {
		err := db.Disconnect(context.TODO())
		Critical(err)
	}()

	//filter에 적용할 user의 데이터를 가져옴
	user_struct := OidTOuser_struct(SessionTO_oid(w, r))
	splited_loc := strings.Split(user_struct.Settings.Loc, " ")
	if len(splited_loc) <= 1 { //인덱스 런타임 에러 방지
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		err_msg := map[string]string{"error": "users 관련 처리 중 오류 발생"}
		err_msg_json, _ := json.MarshalIndent(err_msg, " ", "	")
		w.Write(err_msg_json)
		return
	}

	coll_avty := db.Database("dj_jobs").Collection("type_availability")
	var typeavt [3]Dj_jobs_typeavt
	err = coll_avty.FindOne(context.TODO(), bson.D{{"종류", user_struct.Settings.Type1}}).Decode(&typeavt[0])
	ErrOK(err)
	err = coll_avty.FindOne(context.TODO(), bson.D{{"종류", user_struct.Settings.Type2}}).Decode(&typeavt[1])
	ErrOK(err)
	err = coll_avty.FindOne(context.TODO(), bson.D{{"종류", user_struct.Settings.Type3}}).Decode(&typeavt[2])
	ErrOK(err)

	avt_inters := type_inters(typeavt[0].Availability, typeavt[1].Availability, typeavt[2].Availability) //교집합 구하기
	log.Println("avt inters:", avt_inters)
	// avt 관련 쿼리 종료

	//직장 쿼리 시작
	//
	//
	coll := db.Database("dj_jobs").Collection("job_list")
	// **도 쿼리 시작
	var will_send Dj_jobs_detail_s
	var filter_loc_0 string
	var filter_loc_1 string
	if len(splited_loc) <= 1 { //빈칸일경우 모든 지역 포함간주
		filter_loc_0 = ""
		filter_loc_1 = ""
	} else {
		filter_loc_0 = splited_loc[0]
		filter_loc_1 = splited_loc[1]
	}
	filter := bson.D{
		{"$and", bson.A{
			bson.D{{"사업장 주소", bson.D{{"$regex", filter_loc_0}}}},
		}}}
	cursor, err := coll.Find(context.TODO(), filter)
	ErrOK(err)
	defer cursor.Close(context.TODO())
	cnt := 0
	for cursor.Next(context.TODO()) {
		if cnt > BATCHSIZE {
			break
		}
		var dbres_loc1 Dj_jobs_detail = Dj_jobs_detail{}
		cursor.Decode(&dbres_loc1)
		will_send_append(&dbres_loc1, &will_send, 200)
		cnt++
	}

	// **시 쿼리 시작
	filter = bson.D{
		{"$and", bson.A{
			bson.D{{"사업장 주소", bson.D{{"$regex", filter_loc_0}}}},
			bson.D{{"사업장 주소", bson.D{{"$regex", filter_loc_1}}}},
		}}}
	cursor, err = coll.Find(context.TODO(), filter)
	ErrOK(err)
	defer cursor.Close(context.TODO())
	cnt = 0
	for cursor.Next(context.TODO()) {
		if cnt > BATCHSIZE {
			break
		}
		var dbres_loc2 Dj_jobs_detail = Dj_jobs_detail{}
		cursor.Decode(&dbres_loc2)
		will_send_append(&dbres_loc2, &will_send, 100)
		cnt++
	}
	// type_inters 순회하여 쿼리 시작
	for _, v := range avt_inters { //avt 순회
		if cnt > BATCHSIZE {
			break
		}
		filter_avt := v
		filter := bson.D{
			{"$and", bson.A{
				bson.D{{"필수부위", bson.D{{"$regex", filter_avt}}}},
			}}}
		cursor, err := coll.Find(context.TODO(), filter)
		ErrOK(err)
		defer cursor.Close(context.TODO())
		cnt = 0
		for cursor.Next(context.TODO()) {
			if cnt > BATCHSIZE {
				break
			}
			var dbres_type Dj_jobs_detail
			cursor.Decode(&dbres_type)
			will_send_append(&dbres_type, &will_send, 110)
			cnt++
		}
	}

	//
	//
	//

	//score을 기반으로 sort 시작
	sort.Sort(Dj_jobs_detail_s(will_send))
	will_send_json, _ := json.MarshalIndent(will_send, " ", "	")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(will_send_json)
}