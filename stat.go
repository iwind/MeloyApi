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
	"strconv"
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
				manager.dump()
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


	CREATE TABLE IF NOT EXISTS stat_global (
		id integer not null primary key autoincrement,
		requests integer,
		hits integer,
		errors integer,
		created_at integer
	);
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
func (manager *StatManager) send(address ApiAddress, path string, uri string, timeMs int64, errors int64, hits int64) {
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
func (manager *StatManager) sendDebug(address ApiAddress, path string, uri string, _log string) {
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
func (manager *StatManager) dump() {
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

	//当日统计
	now := time.Now()
	for _, statData := range data {
		_, err := stmt.Exec(statData.Server, statData.Host, statData.Path, statData.TotalMs / statData.Requests, now.Year(), int(now.Month()), now.Day(), now.Hour(), now.Minute(), statData.Requests, statData.Errors, statData.Hits)
		if err != nil {
			log.Println("Error:" + err.Error())
			continue
		}
	}

	//总体统计
	statManager.updateGlobalStat(data)

	//导日志
	statManager.flushDebugLogs()
}

// 更新全局统计
func (manager *StatManager) updateGlobalStat(data map[string]StatData) {
	//总体统计
	globalStmt, err := manager.db.Prepare("SELECT id FROM stat_global LIMIT 1")
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}

	defer globalStmt.Close()

	row := globalStmt.QueryRow()
	var id int
	err = row.Scan(&id)
	if err != nil {
		manager.db.Exec("INSERT INTO stat_global (requests, hits, errors, created_at) VALUES (?, ?, ?, ?)", 0, 0, 0, time.Now().Unix())
	}

	requests := 0
	hits := 0
	errors := 0

	for _, stat := range data {
		requests += int(stat.Requests)
		hits += int(stat.Hits)
		errors += int(stat.Errors)
	}

	if requests > 0 || hits > 0 || errors > 0 {
		manager.db.Exec("UPDATE stat_global SET requests=requests+?,hits=hits+?,errors=errors+?", requests, hits, errors)
	}
}

// 取得当天的总统计
func (manager *StatManager) avgStat(path string) ApiStat  {
	now := time.Now()
	return manager.findAvgStatForDay(path, now.Year(), int(now.Month()), now.Day())
}

// 取得某一天的总统计
func (manager *StatManager) findAvgStatForDay(path string, year int, month int, day int) ApiStat {
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
func (manager *StatManager) findMinuteStatForDay(path string, year int, month int, day int) (stats []ApiMinuteStat) {
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
func (manager *StatManager) findDebugLogsForPath(path string) (logs []DebugLog) {
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
func (manager *StatManager) flushDebugLogs() (err error, count int) {
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

// 按请求数排序
func (manager *StatManager) findRequestsRank(size int) (apis []Map, err error) {
	apis = []Map{}
	stmt, err := manager.db.Prepare("SELECT path,SUM(requests) AS sum FROM stat_" + lastTableDay + " GROUP BY path ORDER BY sum DESC LIMIT " + strconv.Itoa(size))
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}

	for rows.Next() {
		var path string
		var sum int

		rows.Scan(&path, &sum)

		apis = append(apis, Map {
			"path": path,
			"count": sum,
		})
	}

	return
}

// 按缓存命中数排序
func (manager *StatManager) findHitsRank(size int) (apis []Map, err error) {
	apis = []Map{}
	stmt, err := manager.db.Prepare("SELECT path,AVG(hits * 100/requests) AS sum FROM stat_" + lastTableDay + " WHERE hits>0 GROUP BY path ORDER BY sum DESC LIMIT " + strconv.Itoa(size))
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}

	for rows.Next() {
		var path string
		var sum float32

		rows.Scan(&path, &sum)

		apis = append(apis, Map {
			"path": path,
			"percent": sum,
		})
	}

	return
}

// 按错误率排序
func (manager *StatManager) findErrorsRank(size int) (apis []Map, err error) {
	apis = []Map{}
	stmt, err := manager.db.Prepare("SELECT path,AVG(errors * 100/requests) AS sum FROM stat_" + lastTableDay + " WHERE errors>0 GROUP BY path ORDER BY sum DESC LIMIT " + strconv.Itoa(size))
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}

	for rows.Next() {
		var path string
		var sum float32

		rows.Scan(&path, &sum)

		apis = append(apis, Map {
			"path": path,
			"percent": sum,
		})
	}

	return
}

// 按照耗时排序
func (manager *StatManager) findCostRank(size int) (apis []Map, err error) {
	apis = []Map{}
	stmt, err := manager.db.Prepare("SELECT path,AVG(ms) AS sum FROM stat_" + lastTableDay + " GROUP BY path ORDER BY sum DESC LIMIT " + strconv.Itoa(size))
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}

	for rows.Next() {
		var path string
		var sum float32

		rows.Scan(&path, &sum)

		apis = append(apis, Map {
			"path": path,
			"ms": int(sum),
		})
	}

	return
}

// 整体请求频率、命中率、错误率
func (manager *StatManager) findStat()(result Map, err error) {
	result = Map{
		"requests": 0,
		"hits": 0,
		"errors": 0,
	}
	stmt, err := manager.db.Prepare("SELECT AVG(requests) as requests, SUM(hits) * 100/SUM(requests) AS hits, SUM(errors) * 100/SUM(requests) AS errors, AVG(ms) AS ms FROM stat_" + lastTableDay)
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}

	defer stmt.Close()

	row := stmt.QueryRow()
	var requests float32
	var hits float32
	var errors float32
	var ms float32
	err = row.Scan(&requests, &hits, &errors, &ms)
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}

	result = Map{
		"requests": int(requests),
		"hits": hits,
		"errors": errors,
		"ms": int(ms),
	}
	return
}

// 全部的请求数、命中数、错误数
func (manager *StatManager) findGlobalStat() (result Map) {
	result = Map {
		"requests": 0,
		"hits": 0,
		"errors": 0,
		"dateFrom": "",
		"apis": 0,
	}
	row := manager.db.QueryRow("SELECT requests, hits, errors, created_at FROM stat_global LIMIT 1")
	var requests int
	var hits int
	var errors int
	var createdAt int
	err := row.Scan(&requests, &hits, &errors, &createdAt)
	if err != nil {
		log.Println(err)
		requests = 0
		hits = 0
		errors = 0
	} else {
		dateFrom := time.Unix(int64(createdAt), 0)
		result["dateFrom"] = fmt.Sprintf("%d-%02d-%02d", dateFrom.Year(), int(dateFrom.Month()), dateFrom.Day())
	}

	result["requests"] = requests
	result["hits"] = hits
	result["errors"] = errors
	result["apis"] = len(ApiArray)

	return
}

// 关闭统计管理器
func (manager *StatManager) close()  {
	manager.db.Close()
}