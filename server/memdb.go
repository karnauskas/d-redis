package server

import (
	"sync"
	"github.com/widaT/newredis/structure"
	"fmt"
	"strings"
	"errors"
	"strconv"
	"github.com/vmihailenco/msgpack"
)

type (
	HashValue   map[string][]byte
	HashFloat     map[string]float64
	HashHash    map[string]HashValue
	HashHashFloat map[string]HashFloat
	HashBrStack map[string]*structure.List
	HashSkipList map[string]*structure.SkipList
	HashSet     map[string]*structure.Set
	HashList    map[string][][]byte
)

type Memdb struct {
	Values  HashValue
	Hvalues HashHash
	dlList HashBrStack
	HSet HashSet
	HList HashList
	HSortSet HashHashFloat
	skiplist HashSkipList
	rwmu sync.RWMutex
	recovebool bool
}

func NewMemdb() *Memdb {
	db := &Memdb{
		Values:   make(HashValue),
		dlList:  make(HashBrStack),
		HSet    :  make(HashSet),
		HSortSet    :  make(HashHashFloat),
		HList    :  make(HashList),
		Hvalues :make(HashHash),
		skiplist : make(HashSkipList),
	}
	return db
}

type Opt struct {
	Method string
	Key  string
	Args   [][]byte
}

func (o *Opt)String() string  {
	return  o.Method + o.Key
}

func (m *Memdb) getSnapshot()  ([]byte, error) {
	b,err := msgpack.Marshal(m)
	if err != nil {
		return nil,err
	}
	List  := m.dlList
	m.HList = make(HashList)
	for key,v:= range List {
		m.HList[key] = v.Values()
	}
	m.HList = nil
	return b,nil
}

func (m *Memdb)  recoverFromSnapshot(snapshot []byte) error {
	var db Memdb
	if err := msgpack.Unmarshal(snapshot,&db); err != nil {
		return err
	}
	db.dlList = make(HashBrStack)
	for key,v := range db.HList {
		if _,found := db.dlList[key];!found {
			db.dlList[key] = structure.NewList()
		}
		db.dlList[key].Add(v...)
	}
	db.HList = nil

	db.skiplist = make(HashSkipList)
	//重新初始化skiplist
	for key,val := range db.HSortSet {
		intmap := structure.NewSkipList()
		for k,v := range val {
			intmap.Set(v,k)
		}
		db.skiplist[key] = intmap
	}
	*m = db
	return nil
}

//list operation
func (m *Memdb) Rpush(uk string ,values ...[]byte)  {
	m.rwmu.Lock()
	defer m.rwmu.Unlock()
	key := string(values[0])
	if _, exists := m.dlList[key]; !exists {
		m.dlList[key] =structure.NewList()
	}
	n := m.dlList[key].Rpush(values[1:]...)
	if conn := Conns.Get(uk) ;conn != nil{
		conn.WriteInt(n)
		if  wait :=conn.Context() ;wait!= nil {
			wait.(* sync.WaitGroup).Done()
		}
		Conns.Del(uk)
	}
}

func (m *Memdb) Lrange(key string, start, stop int) (*[][]byte, error) {
	m.rwmu.RLock()
	defer m.rwmu.RUnlock()

	if _, exists := m.dlList[key]; !exists {
		m.dlList[key] = structure.NewList()
	}

	if start < 0 {
		if start = m.dlList[key].Size() + start; start < 0 {
			start = 0
		}
	}

	var ret [][]byte
	if stop < 0 {
		stop =  m.dlList[key].Size() + stop
		if stop <0 {
			return nil,nil
		}
	}
	var iter = m.dlList[key].Seek(start)
	if iter != nil {
		ret = append(ret, iter.Value())
	}
	for iter.Next(){
		if iter.Key() <= stop {
			ret = append(ret, iter.Value())
		}else {
			break
		}
	}
	iter.Close()
	return &ret, nil
}

func (m *Memdb) Lindex(key string, index int) ([]byte, error) {
	m.rwmu.RLock()
	defer m.rwmu.RUnlock()
	if _, exists := m.dlList[key]; !exists {
		m.dlList[key] =  structure.NewList()
	}
	ret,_ := m.dlList[key].Get(index)
	return ret, nil
}

func (m *Memdb) Lpush(uk string, values ...[]byte) {
	m.rwmu.Lock()
	defer m.rwmu.Unlock()
	key := string(values[0])
	if _, exists := m.dlList[key]; !exists {
		m.dlList[key] = structure.NewList()
	}
	num := m.dlList[key].Lpush(values[1:]...)
	if conn := Conns.Get(uk) ;conn != nil{
		conn.WriteInt(num)
		if  wait :=conn.Context() ;wait!= nil {
			wait.(* sync.WaitGroup).Done()
		}
		Conns.Del(uk)
	}
}


func (m *Memdb)Lpop(uk string,key string) {
	m.rwmu.Lock()
	defer m.rwmu.Unlock()
	conn := Conns.Get(uk)
	if _,found := m.dlList[key];!found{
		if conn != nil{
			conn.WriteNull()
			if  wait :=conn.Context() ;wait!= nil {
				wait.(* sync.WaitGroup).Done()
			}
			Conns.Del(uk)
		}
		return
	}
	ret := m.dlList[key].Lpop()
	if conn != nil{
		if ret != nil {
			conn.WriteBulk(ret)
		}else {
			conn.WriteNull()
		}
		if  wait :=conn.Context() ;wait!= nil {
			wait.(* sync.WaitGroup).Done()
		}
		Conns.Del(uk)
	}
}

func (m *Memdb)Rpop(uk,key string) {
	m.rwmu.Lock()
	defer m.rwmu.Unlock()
	conn := Conns.Get(uk)
	if _,found := m.dlList[key];!found{
		if conn != nil{
			conn.WriteNull()
			if  wait :=conn.Context() ;wait!= nil {
				wait.(* sync.WaitGroup).Done()
			}
			Conns.Del(uk)
		}
		return
	}
	ret := m.dlList[key].Rpop()
	if conn != nil{
		if ret != nil {
			conn.WriteBulk(ret)
		}else {
			conn.WriteNull()
		}
		if  wait :=conn.Context() ;wait!= nil {
			wait.(* sync.WaitGroup).Done()
		}
		Conns.Del(uk)
	}
}

//set operation
func (m *Memdb) Sadd (uk,key string, values ...[]byte){
	m.rwmu.Lock()
	defer m.rwmu.Unlock()
	if _, exists := m.HSet[key]; !exists {
		m.HSet[key] = structure.NewSset(key)
	}
	count := 0
	for _,value :=range values {
		count =count + m.HSet[key].Add(string(value))
	}
	if conn := Conns.Get(uk); conn != nil{
		conn.WriteInt(count)
		if  wait :=conn.Context() ;wait!= nil {
			wait.(* sync.WaitGroup).Done()
		}
		Conns.Del(uk)
	}
}


func (m *Memdb) Scard (key string)( int,error) {
	m.rwmu.RLock()
	defer m.rwmu.RUnlock()

	if _, exists := m.HSet[key]; !exists {
		return 0,nil
	}
	return m.HSet[key].Len(),nil
}

func (m *Memdb)Spop(key string)( []byte,error)  {
	m.rwmu.Lock()
	defer m.rwmu.Unlock()
	if _, exists := m.HSet[key]; !exists {
		return nil,nil
	}
	if m.HSet[key].Len() == 0 {
		return nil,nil
	}
	v := m.HSet[key].RandomKey()
	return []byte(v),nil
}

func (m * Memdb)Sspop(uk,key string,k []byte)  {
	m.rwmu.Lock()
	defer m.rwmu.Unlock()
	if _, exists := m.HSet[key]; !exists {
		return
	}
	m.HSet[key].Del(string(k))
	if conn := Conns.Get(uk); conn != nil{
		conn.WriteBulk(k)
		if  wait :=conn.Context() ;wait!= nil {
			wait.(* sync.WaitGroup).Done()
		}
		Conns.Del(uk)
	}
}

func (m *Memdb) Smembers (key string)  ([][]byte,error) {
	m.rwmu.RLock()
	defer m.rwmu.RUnlock()
	if _, exists := m.HSet[key]; !exists {
		return nil,nil
	}
	return *m.HSet[key].Members(),nil
}

//hash set
func (m *Memdb) Hget(key, subkey string) ([]byte, error) {
	if m.Hvalues == nil {
		return nil, nil
	}
	m.rwmu.RLock()
	defer m.rwmu.RUnlock()
	if v, exists := m.Hvalues[key]; exists {
		if v, exists := v[subkey]; exists {
			return v, nil
		}
	}
	return nil, nil
}

func (m *Memdb) Hset(uk ,key, subkey string, value []byte) {
	ret := 0
	m.rwmu.Lock()
	defer m.rwmu.Unlock()
	if _, exists := m.Hvalues[key]; !exists {
		m.Hvalues[key] = make(HashValue)
		ret = 1
	}
	if _, exists := m.Hvalues[key][subkey]; !exists {
		ret = 1
	}
	m.Hvalues[key][subkey] = value
	if conn := Conns.Get(uk) ;conn != nil{
		conn.WriteInt(ret)
		if  wait :=conn.Context() ;wait!= nil {
			wait.(* sync.WaitGroup).Done()
		}
		Conns.Del(uk)
	}
}

func (m *Memdb) Hgetall(key string) (HashValue, error) {
	if  m.Hvalues == nil {
		return nil, nil
	}
	m.rwmu.RLock()
	defer m.rwmu.RUnlock()
	return m.Hvalues[key], nil
}

func (m *Memdb) Get(key string) ([]byte, error) {
	m.rwmu.RLock()
	defer m.rwmu.RUnlock()
	return m.Values[key], nil
}

func (m *Memdb) Set(key string, value []byte) error {
	m.rwmu.Lock()
	defer m.rwmu.Unlock()
	m.Values[key] = value
	return nil
}


func (m *Memdb) Mset(values ...[]byte) error {
	m.rwmu.Lock()
	defer m.rwmu.Unlock()
	if len(values) % 2 != 0 {
		return errors.New("wrong number of arguments for MSET")
	}

	var bytes [][]byte
	kvmap := make(map[string][]byte)
	for i,v:= range values {
		bytes = append(bytes,[]byte(v))
		if i % 2 == 0 {
			kvmap[string(v)] = values[i+1]
		}
	}
	for k,v:= range kvmap {
		m.Values[k] = v
	}
	return nil
}


func (m *Memdb) Incr (uk,key string)  {
	m.rwmu.Lock()
	defer m.rwmu.Unlock()
	v,found := m.Values[key]
	num  := 0
	conn := Conns.Get(uk)
	var err error
	if found {
		num ,err = strconv.Atoi(string(v))
		if err != nil {
			if conn != nil{
				conn.WriteError("value is not an integer or out of range")
				if  wait :=conn.Context() ;wait!= nil {
					wait.(* sync.WaitGroup).Done()
				}
				Conns.Del(uk)
			}
			return
		}
	}
	m.Values[key] = []byte(fmt.Sprintf("%d",num+1))
	if conn != nil{
		conn.WriteInt(num+1)
		if  wait :=conn.Context() ;wait!= nil {
			wait.(* sync.WaitGroup).Done()
		}
		Conns.Del(uk)
	}
}

func (m *Memdb) Del(uk string,keys ...[]byte) {
	m.rwmu.Lock()
	defer m.rwmu.Unlock()
	count := 0
	for _, k := range keys {
		key := string(k)
		if _, exists := m.Values[key]; exists {
			delete(m.Values, key)
			count++
		}
		if _, exists := m.Hvalues[key]; exists {
			delete(m.Hvalues, key)
			count++
		}
		if _, exists := m.HSet[key]; exists {
			delete(m.HSet, key)
			count++
		}

		if _, exists := m.HList[key]; exists {
			delete(m.HList, key)
			count++
		}
		if _, exists := m.skiplist[key]; exists {
			delete(m.skiplist, key)
			//count++
		}
		if _, exists := m.HSortSet[key]; exists {
			delete(m.HSortSet, key)
			count++
		}
	}
	if conn := Conns.Get(uk) ;conn != nil{
		conn.WriteInt(count)
		if  wait :=conn.Context() ;wait!= nil {
			wait.(* sync.WaitGroup).Done()
		}
		Conns.Del(uk)
	}
}

//sort set
func (m *Memdb) Zadd (uk,key string,score float64,val string){
	m.rwmu.Lock()
	defer m.rwmu.Unlock()
	if _, exists := m.HSortSet[key]; !exists {
		m.HSortSet[key] = make(HashFloat)
	}
	if _, exists := m.skiplist[key]; !exists {
		m.skiplist[key] = structure.NewSkipList()
	}
	count := 0
	_ ,found :=m.HSortSet[key][val]
	if !found {
		count = 1
	}
	m.HSortSet[key][val] = score
	m.skiplist[key].Set(score,val)

	if conn := Conns.Get(uk) ;conn != nil{
		conn.WriteInt(count)
		if  wait :=conn.Context() ;wait!= nil {
			wait.(* sync.WaitGroup).Done()
		}
		Conns.Del(uk)
	}
}

//@todo 这个实现算法有点问题
func (m *Memdb) Zrange(key string, start, stop float64,args ...[]byte) (*[][]byte, error) {
	withscores := false
	if len (args) > 0  {
		if strings.ToLower(string(args[0])) != "withscores"{
			return nil,errors.New("ERR syntax error")
		}else{
			withscores = true
		}
	}
	m.rwmu.RLock()
	defer m.rwmu.RUnlock()

	if _, exists := m.skiplist[key]; !exists {
		return nil,nil
	}
	iter := m.skiplist[key].Range(start,stop)
	var ret [][]byte
	for iter.Next() {
		if withscores {
			ret = append(ret,  []byte(strconv.FormatFloat(iter.Key(), 'g', -1, 64)))
		}
		ret = append(ret,[]byte(iter.Value()))
	}
	iter.Close()
	return &ret, nil
}


