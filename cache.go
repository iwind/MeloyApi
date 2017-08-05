package MeloyApi

import (
	"net/http"
	"sync"
	"time"
)

// 缓存管理器
type CacheManager struct {
	MaxSize int

	Values map[string]CacheEntry
	Tags   map[string]map[string]string // tag [key] map[key] ""

	Mutex sync.RWMutex
}

// 缓存条目
type CacheEntry struct {
	Bytes       []byte
	Header      http.Header
	LifeMs      int64
	ExpiredAtMs int64
	Tags        []string
}

var cacheInitOnce sync.Once

//初始化
func (manager *CacheManager) init() {
	manager.MaxSize = 1024 * 1024 * 10

	cacheInitOnce.Do(func() {
		manager.Values = make(map[string]CacheEntry)
		manager.Tags = make(map[string]map[string]string)
	})

	//每一分钟清理一次过期的条目
	go func() {
		tick := time.Tick(1 * time.Minute)
		for {
			<-tick

			manager.clearExpired()
		}
	}()
}

// 清除过期的条目
func (manager *CacheManager) clearExpired() {
	//清除过期条目
	for key, value := range manager.Values {
		if value.ExpiredAtMs < int64(time.Now().UnixNano()/1000000) {
			manager.deleteValue(key)
		}
	}

	//数量是否超出最大范围，如果超出则截为一半的数量
	mapSize := len(manager.Values)
	if mapSize > manager.MaxSize {
		newSize := mapSize / 2

		if newSize > 0 {
			for key := range manager.Values {
				if newSize <= 0 {
					break
				}

				manager.deleteValue(key)

				newSize --
			}
		}
	}
}

// 清除所有的条目
func (manager *CacheManager) clearAll() (count int) {
	manager.Mutex.Lock()
	defer manager.Mutex.Unlock()

	count = len(manager.Values)
	manager.Values = make(map[string]CacheEntry)
	manager.Tags = make(map[string]map[string]string)
	return
}

// 删除某个标签关联的条目
func (manager *CacheManager) deleteTag(tag string) (count int) {
	keyMap, ok := manager.Tags[tag]
	if !ok {
		return
	}

	if keyMap == nil {
		manager.Mutex.Lock()

		delete(manager.Tags, tag)

		manager.Mutex.Unlock()

		return
	}

	count = len(keyMap)

	manager.Mutex.Lock()
	delete(manager.Tags, tag)
	manager.Mutex.Unlock()

	for key := range keyMap {
		manager.deleteValue(key)
	}

	return
}

// 设置条目内容
func (manager *CacheManager) set(key string, tags []string, _bytes []byte, header http.Header, lifeMs int64) {
	nowMs := time.Now().UnixNano() / 1000000

	manager.Mutex.Lock()
	defer manager.Mutex.Unlock()

	if tags == nil {
		tags = []string{}
	}

	manager.Values[key] = CacheEntry{
		Bytes:       _bytes,
		Header:      header,
		LifeMs:      lifeMs,
		ExpiredAtMs: nowMs + lifeMs,
		Tags:        tags,
	}

	//设置tag
	if len(tags) > 0 {
		for _, tag := range tags {
			keyMapping, ok := manager.Tags[tag]
			if !ok {
				keyMapping = make(map[string]string)
			}

			keyMapping[key] = ""
			manager.Tags[tag] = keyMapping
		}
	}
}

// 取得条目内容
func (manager *CacheManager) get(key string) (entry CacheEntry, ok bool) {
	if manager.Values == nil {
		ok = false
		return
	}

	entry, ok = manager.Values[key]
	if !ok {
		return
	}

	if entry.ExpiredAtMs < int64(time.Now().UnixNano()/1000000) {
		manager.deleteValue(key)
		ok = false
		return
	}

	ok = true

	return
}

// 删除某个key
func (manager *CacheManager) deleteValue(key string) {
	manager.Mutex.Lock()
	defer manager.Mutex.Unlock()

	if manager.Values == nil {
		return
	}

	entry, ok := manager.Values[key]
	if !ok {
		return
	}

	delete(manager.Values, key)

	if entry.Tags != nil && len(entry.Tags) > 0 {
		for _, tagName := range entry.Tags {
			keyMap, ok := manager.Tags[tagName]
			if !ok {
				continue
			}

			delete(keyMap, key)

			if len(keyMap) == 0 {
				delete(manager.Tags, tagName)
			}
		}
	}
}

// 统计标签信息
// 只取前1000个标签
func (manager *CacheManager) statTag(tag string) (count int, keys []string, ok bool) {
	keys = []string{}

	if manager.Tags == nil {
		ok = false
		return
	}

	keyMap, ok := manager.Tags[tag]
	if !ok {
		ok = false
		return
	}

	ok = true

	count = len(keyMap)

	i := 0
	for key := range keyMap {
		if i >= 1000 {
			break
		}

		keys = append(keys, key)

		i ++
	}

	return
}
