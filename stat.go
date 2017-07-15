package MeloyApi

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"time"
	"strings"
	"fmt"
	"sync"
	"encoding/json"
)

type StatManager struct {
	Data map[string] StatData
	DebugLogs []DebugLog

	db *sql.DB
}

type StatData struct {
	Server string
	Host string
	Path string

	TotalMs int64
	Requests int64
	Errors int64
	Hits int64
}

type DebugLog struct {
	Server string `json:"server"`
	Host string `json:"host"`
	Path string `json:"path"`
	URI string `json:"uri"`

	Log string `json:"body"`
	CreatedAt int64 `json:"createdAt"`
}

type ApiMinuteStat struct {
	Hour int `json:"hour"`
	Minute int `json:"minute"`
	Requests int `json:"requests"`
	Errors int `json:"errors"`
	Hits int `json:"hits"`
	AvgMs int `json:"avgMs"`
}

type ApiStat struct {
	AvgMs int `json:"avgMs"`
	Requests int `json:"requests"`
	Hits int `json:"hits"`
	Errors int `json:"errors"`
}

var lastTableDay string = ""
var statMu sync.Mutex

// 初始化
func (manager *StatManager) Init(appDir string) {
	manager.Data = make(map[string] StatData)
	manager.DebugLogs = []DebugLog{}

	//启动数据库
	db, err := sql.Open("sqlite3", appDir + "/data/stat.db")
	if err != nil {
		log.Fatal("Can not open database at '" + appDir + "/data/stat.db" + "':", err.Error())
	}
	manager.db = db

	// 准备数据库表
	manager.PrepareDailyTable()

	//启动定时器，每分钟导出数据到本地
	go func() {
		tick := time.Tick(1 * time.Minute)
		for {
			<- tick

			if manager.PrepareDailyTable() {
				manager.Dump()
			}
		}
	}()
}

// 准备每天的数据表格
func (manager *StatManager) PrepareDailyTable() bool {
	now := time.Now()
	date := fmt.Sprintf("%d%02d%02d", now.Year(), int(now.Month()), now.Day())

	if lastTableDay == date {
		return true
	}

	sqlStmt := `
	CREATE TABLE IF NOT EXISTS stat_%{date} (
		id integer not null primary key autoincrement,
		server text,
		host text,
		path text,
		ms integer,
		year integer,
		month integer,
		day integer,
		hour integer,
		minute integer,
		requests integer,
		errors integer,
		hits integer
	);
	CREATE INDEX IF NOT EXISTS server ON stat_%{date} (server);
	CREATE INDEX IF NOT EXISTS host ON stat_%{date} (host);
	CREATE INDEX IF NOT EXISTS path_index ON stat_%{date} (path);
	CREATE INDEX IF NOT EXISTS date_minute_index ON stat_%{date} (path, year, month, day, hour, minute);
	CREATE INDEX IF NOT EXISTS date_day_index ON stat_%{date} (path, year, month, day);


	CREATE TABLE IF NOT EXISTS debug_logs_%{date} (
		id integer not null primary key autoincrement,
		server text,
		host text,
		path text,
		uri text,
		body string,
		created_at integer
	);
	CREATE INDEX IF NOT EXISTS path_index ON debug_logs_%{date} (path);

	`

	sqlStmt = strings.Replace(sqlStmt, "%{date}", date, -1)

	log.Println("create table for date '" + date + "'")

	_, err := manager.db.Exec(sqlStmt)
	if err != nil {
		log.Println("error:" + err.Error())
		return false
	}

	lastTableDay = date

	return true
}

// 发送统计信息
func (manager *StatManager) Send(address ApiAddress, path string, uri string, timeMs int64, errors int64, hits int64) {
	key := address.Server + "$$" + address.Host + "$$" + path
	value, ok := manager.Data[key]
	if !ok {
		value = StatData {
			address.Server,
			address.Host,
			path,
			timeMs,
			1,
			errors,
			hits,
		}
	} else {
		value.TotalMs += timeMs
		value.Requests += 1
		value.Errors += errors
		value.Hits += hits
	}
	manager.Data[key] = value

	if appManager.IsDebug {
		bytes, err := json.MarshalIndent(Map {
			"Api": path,
			"Address": address.URL,
			"URI": uri,
			"TimeMs": timeMs,
			"HasErrors": errors > 0,
			"HitCache": hits > 0,
		}, "", "    ")
		if err != nil {
			log.Println(err)
		} else {
			log.Println(string(bytes))
		}
	}
}

// 发送调试信息
func (manager *StatManager) SendDebug(address ApiAddress, path string, uri string, _log string) {
	manager.DebugLogs = append(manager.DebugLogs, DebugLog{
		address.Server,
		address.Host,
		path,
		uri,
		_log,
		time.Now().Unix(),
	})

	if appManager.IsDebug {
		bytes, err := json.MarshalIndent(Map {
			"Api": path,
			"Address": address.URL,
			"URI": uri,
			"Log": _log,
		}, "", "    ")
		if err != nil {
			log.Println(err)
		} else {
			log.Println(string(bytes))
		}
	}
}

// 导出数据到数据库
func (manager *StatManager) Dump() {
	data := manager.Data

	//清空
	manager.Data = map [string] StatData {}

	//导数据
	stmt, err := manager.db.Prepare("INSERT INTO stat_" + lastTableDay + " (server,host,path,ms, year,month,day,hour, minute,requests,errors,hits) VALUES (?,?,?,?, ?,?,?,?, ?,?,?,?)")
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}

	defer stmt.Close()

	now := time.Now()
	for _, statData := range data {
		_, err := stmt.Exec(statData.Server, statData.Host, statData.Path, statData.TotalMs / statData.Requests, now.Year(), int(now.Month()), now.Day(), now.Hour(), now.Minute(), statData.Requests, statData.Errors, statData.Hits)
		if err != nil {
			log.Println("Error:" + err.Error())
			continue
		}
	}

	//导日志
	statManager.FlushDebugLogs()
}

// 取得当天的总统计
func (manager *StatManager) AvgStat(path string) ApiStat  {
	now := time.Now()
	return manager.FindAvgStatForDay(path, now.Year(), int(now.Month()), now.Day())
}

// 取得某一天的总统计
func (manager *StatManager) FindAvgStatForDay(path string, year int, month int, day int) ApiStat {
	date := fmt.Sprintf("%d%02d%02d", year, month, day)
	stmt, err := manager.db.Prepare("SELECT SUM(ms),SUM(requests),SUM(hits),SUM(errors) FROM stat_" + date + " WHERE path=?")
	if err != nil {
		log.Println("Error:" + err.Error())
		return ApiStat{ AvgMs: 0, Requests: 0, Hits: 0, Errors: 0 }
	}

	defer stmt.Close()

	row := stmt.QueryRow(path)
	var totalMs int
	var requests int
	var hits int
	var errors int
	err = row.Scan(&totalMs, &requests, &hits, &errors)

	if err != nil {
		//log.Println("Error:" + err.Error())
		return ApiStat{ AvgMs: 0, Requests: 0, Hits: 0, Errors: 0 }
	}

	return ApiStat {
		AvgMs: totalMs / requests,
		Requests: requests,
		Hits: hits,
		Errors: errors,
	}
}

// 取得某天的分钟统计
func (manager *StatManager) FindMinuteStatForDay(path string, year int, month int, day int) (stats []ApiMinuteStat) {
	stats = []ApiMinuteStat{}

	date := fmt.Sprintf("%d%02d%02d", year, month, day)
	stmt, err := manager.db.Prepare("SELECT ms,requests,errors,hits,hour,minute FROM stat_" + date + " WHERE path=? ORDER BY id ASC")
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}

	defer stmt.Close()

	rows, err := stmt.Query(path)
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}

	defer rows.Close()

	for rows.Next() {
		var ms int
		var requests int
		var errors int
		var hits int
		var hour int
		var minute int

		rows.Scan(&ms, &requests, &errors, &hits, &hour, &minute)

		stats = append(stats, ApiMinuteStat{
			Hour: hour,
			Minute: minute,
			AvgMs:ms,
			Requests:requests,
			Errors: errors,
			Hits:hits,
		})
	}

	return
}

// 取得某个接口的调试日志
func (manager *StatManager) FindDebugLogsForPath(path string) (logs []DebugLog) {
	logs = []DebugLog{}

	now := time.Now()
	date := fmt.Sprintf("%d%02d%02d", now.Year(), int(now.Month()), now.Day())

	stmt, err := manager.db.Prepare("SELECT server, host, path, uri, body, created_at FROM debug_logs_" + date + " WHERE path=? ORDER BY id DESC LIMIT 100")
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}

	defer stmt.Close()

	rows, err := stmt.Query(path)
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}

	for rows.Next() {
		var server string
		var host string
		var path string
		var uri string
		var body string
		var createdAt int64

		rows.Scan(&server, &host, &path, &uri, &body, &createdAt)

		logs = append(logs, DebugLog {
			server,
			host,
			path,
			uri,
			body,
			createdAt,
		})
	}

	return
}

// 刷新调试数据
func (manager *StatManager) FlushDebugLogs() (err error, count int) {
	now := time.Now()
	date := fmt.Sprintf("%d%02d%02d", now.Year(), int(now.Month()), now.Day())

	statMu.Lock()

	//导日志
	debugLogs := manager.DebugLogs
	manager.DebugLogs = []DebugLog{}

	count = len(debugLogs)

	insertDebugStmt, err := manager.db.Prepare("INSERT INTO debug_logs_" + date + " (server, host, path, uri, body, created_at) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Println("Error:" + err.Error())
		statMu.Unlock()
		return
	}

	defer insertDebugStmt.Close()

	for _, debugLog := range debugLogs {
		_, err := insertDebugStmt.Exec(debugLog.Server, debugLog.Host, debugLog.Path, debugLog.URI, debugLog.Log, debugLog.CreatedAt)
		if err != nil {
			log.Println("Error:" + err.Error())
			continue
		}
	}

	statMu.Unlock()
	return
}

// 关闭统计管理器
func (manager *StatManager) Close()  {
	manager.db.Close()
}