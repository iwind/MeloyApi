package MeloyApi

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"time"
)

type StatManager struct {
	Data map[string] StatData
	db *sql.DB
}

type StatData struct {
	TotalMs int64
	Requests int64
}

func (manager *StatManager) Init(configDir string) {
	manager.Data = make(map[string] StatData)

	//启动数据库
	db, err := sql.Open("sqlite3", configDir + "/data/stat.db")
	if err != nil {
		log.Fatal("Can not open database at '" + configDir + "/data/stat.db" + "'")
	}
	manager.db = db

	//启动定时器，每分钟导出数据到本地
	go func() {
		tick := time.Tick(1 * time.Minute)
		for {
			<- tick
			manager.Dump()
		}
	}()
}

func (manager *StatManager) Send(path string, timeMs int64) {
	value, ok := manager.Data[path]
	if !ok {
		value = StatData{
			timeMs,
			1,
		}
	} else {
		value.TotalMs += timeMs
		value.Requests += 1
	}
	manager.Data[path] = value

	log.Printf("%s, %d, %d, %d\n", path, value.TotalMs, value.Requests, value.TotalMs / value.Requests)
}

func (manager *StatManager) Dump() {
	data := manager.Data

	//清空
	manager.Data = make(map [string] StatData)

	//导数据
	stmt, err := manager.db.Prepare("INSERT INTO stat (path,ms,year,month,day,hour,minute,requests) VALUES (?,?,?,?,?,?,?,?)")
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}

	defer stmt.Close()

	now := time.Now()
	for path, statData := range data {
		_, err := stmt.Exec(path, statData.TotalMs / statData.Requests, now.Year(), int(now.Month()), now.Day(), now.Hour(), now.Minute(), statData.Requests)
		if err != nil {
			log.Println("Error:" + err.Error())
			continue
		}
	}
}

func (manager *StatManager) AvgStat(path string) ApiStat  {
	stmt, err := manager.db.Prepare("SELECT SUM(ms),SUM(requests) FROM stat WHERE path=? AND year=? AND month=? AND day=?")
	if err != nil {
		log.Println("Error:" + err.Error())
		return ApiStat{ AvgMs: 0, Requests: 0 }
	}

	defer stmt.Close()

	now := time.Now()
	row := stmt.QueryRow(path, now.Year(), int(now.Month()), now.Day())

	var totalMs int
	var requests int
	err = row.Scan(&totalMs, &requests)

	if err != nil {
		//log.Println("Error:" + err.Error())
		return ApiStat{ AvgMs: 0, Requests: 0 }
	}

	log.Printf("total:%d, requests:%d", totalMs, requests)

	return ApiStat{
		AvgMs:totalMs / requests,
		Requests:requests,
	}
}

func (manager *StatManager) Close()  {
	manager.db.Close()
}