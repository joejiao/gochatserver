package chat

import (
    "encoding/json"
    "log"
    "io/ioutil"
    "sync"
    "time"
)

type Filter struct {
    BlackList
}

// blacklist.json
// 数据为json, 格式: uid:roomId, 数据类型: uid是string，roomId是int, 如果全局禁言，则roomId为0
// {"1":1, "2":1, "3":0, "4":3}

type BlackList struct {
    sync.RWMutex
    Enable      bool
    FileName    string
    Data        map[string]interface{}
}

func NewFilter(opts *Options) *Filter {
    filter := &Filter{
        BlackList: BlackList{
            Enable:     true,
            FileName:   opts.FilterDir + "/blacklist.json",
            Data:       make(map[string]interface{}),
        },
    }
    return filter
}

func (self *Filter) StartAndServe() {
    self.BlackList.loadData()
    // 定时重新载入数据
    go func() {
        ticker := time.NewTicker(time.Second * 120)
        defer ticker.Stop()

        for _ = range ticker.C {
            self.BlackList.loadData()
        }
    }()
}

func (self *BlackList) loadData() {
    name := self.FileName
    if IsFileExist(name) == false {
        self.Enable = false
        log.Printf("Json File %s not found, BlackList disable\n", name)
        return
    }

    file, err := ioutil.ReadFile(name)
    if err != nil {
        self.Enable = false
        log.Printf("ReadFile %s error, BlackList disable: %s\n", name, err)
        return
    }
    //fmt.Printf("%s\n", string(file))

    var f interface{}
    err = json.Unmarshal(file, &f)
    if err != nil {
        self.Enable = false
        log.Printf("json.Unmarshal %s error, BlackList disable: %s\n", name, err)
        return
    }

    self.Lock()
    self.Enable = true
    self.Data = f.(map[string]interface{})
    self.Unlock()
    return
}

func (self *BlackList) IsBlocked(uid string, roomName string) bool {
    if self.Enable == false {
        return false
    }

    self.RLock()
    rid, ok := self.Data[uid]
    self.RUnlock()
    if !ok {
        return false
    }

    ridStr, ok1 := rid.(string)
    if !ok1 {
        log.Printf("type assertion error: %+v\n", rid)
        return false
    }

    if rid == 0 || ridStr == roomName {
        return true
    }

    return false
}
