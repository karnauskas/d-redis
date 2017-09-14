package server

import (
	"strings"
	"strconv"
	"fmt"
	"sync"
)

type T struct {
	fn fn
	w bool
}

type fn func(s *Server,conn Conn, cmd Command) error

var commandMap = make(map[string]T)

func registerCmd(cmd string,f fn,needwait bool)  {
	commandMap[cmd] = T{f,needwait}
}

func DoCmd(s *Server ,conn Conn, cmd Command) (error,bool) {
	c := strings.ToLower(string(cmd.Args[0]))
	f,found := commandMap[c]
	if !found{
		conn.WriteError("ERR unknown command '" + string(cmd.Args[0]) + "'")
		return nil,false
	}
	return f.fn(s ,conn,cmd),f.w
}

func ping(s *Server,conn Conn, cmd Command) error  {
	conn.WriteString("PONG")
	return nil
}

func sselect(s *Server,conn Conn, cmd Command) error  {
	conn.WriteString("OK")
	return nil
}

//string opt
func set(s *Server,conn Conn, cmd Command) error {
	if len(cmd.Args) != 3 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	//s.db.Set(string(cmd.Args[1]),cmd.Args[2])
	_Storage.Propose(&kv{Method:"set",Args:cmd.Args[1:]})
	conn.WriteString("OK")
	return nil
}

func mset(s *Server,conn Conn, cmd Command) error {
	if len(cmd.Args) < 3 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	err := s.db.Mset(cmd.Args[1:]...)
	if err != nil {
		conn.WriteError(err.Error())
		return nil
	}
	conn.WriteString("OK")
	return nil
}

func del(s *Server,conn Conn, cmd Command) error {
	if len(cmd.Args) < 2 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		conn.Context().(*sync.WaitGroup).Done()
		return nil
	}
	k := fmt.Sprintf("%s",conn.RemoteAddr())
	Conns.Add(k,conn)
	//defer  Conns.Del(k)
	_Storage.Propose(&kv{Method:"del",Args:cmd.Args[1:],Conn:k})

	return nil
}


func incr(s *Server,conn Conn, cmd Command) error {
	if len(cmd.Args) != 2 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	num,err := s.db.Incr(string(cmd.Args[1]))
	if err != nil {
		conn.WriteError(err.Error())
		return nil
	}
	conn.WriteInt(num)
	return nil
}


func get(s *Server,conn Conn, cmd Command) error {
	if len(cmd.Args) != 2 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	v,err := s.db.Get(string(cmd.Args[1]))
	if err != nil {
		conn.WriteError(err.Error())
		return nil
	}
	if v == nil {
		conn.WriteNull()
	}else {
		conn.WriteBulk(v)
	}

	return nil
}

//list opt
func lpush(s *Server,conn Conn, cmd Command)  error {
	if len(cmd.Args) < 3 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	v,err := s.db.Lpush(cmd.Args[1:]...)
	if err != nil {
		conn.WriteError(err.Error())
		return nil
	}
	conn.WriteInt(v)
	return nil
}

func rpush(s *Server,conn Conn, cmd Command)  error {
	if len(cmd.Args) < 3 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	v,err := s.db.Rpush(cmd.Args[1:]...)
	if err != nil {
		conn.WriteError(err.Error())
		return nil
	}
	conn.WriteInt(v)
	return nil
}

func lpop(s *Server,conn Conn, cmd Command)  error {
	if len(cmd.Args) != 2 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	v,_ := s.db.Lpop(string(cmd.Args[1]))
	if v == nil {
		conn.WriteNull()
	}else {
		conn.WriteBulk(v)
	}

	return nil
}

func rpop(s *Server,conn Conn, cmd Command)  error {
	if len(cmd.Args) != 2 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	v,_ := s.db.Rpop(string(cmd.Args[1]))
	if v == nil {
		conn.WriteNull()
	}else {
		conn.WriteBulk(v)
	}
	return nil
}

func lrange(s *Server,conn Conn, cmd Command)  error {
	if len(cmd.Args) != 4 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	start,err1 := strconv.Atoi(string(cmd.Args[2]))
	end,err2 := strconv.Atoi(string(cmd.Args[3]))
	if err1 != nil || err2!= nil{
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	v,_ := s.db.Lrange(string(cmd.Args[1]),start,end)
	if v == nil {
		conn.WriteNull()
	}else {
		conn.WriteArray(len(*v))
		for _,val := range *v {
			conn.WriteBulk(val)
		}
	}
	return nil
}

//set opt
func sadd(s *Server,conn Conn, cmd Command)  error {
	if len(cmd.Args) < 3 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	num,err := s.db.Sadd(string(cmd.Args[1]),cmd.Args[2:]...)
	if err != nil {
		conn.WriteError(err.Error())
	}else {
		conn.WriteInt(num)
	}
	return nil
}

func spop(s *Server,conn Conn, cmd Command)  error {
	if len(cmd.Args) != 2 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	v,_ := s.db.Spop(string(cmd.Args[1]))
	if v == nil {
		conn.WriteNull()
	}else {
		conn.WriteBulk(v)
	}
	return nil
}

func smembers(s *Server,conn Conn, cmd Command)  error {
	if len(cmd.Args) != 2 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	v,_ := s.db.Smembers(string(cmd.Args[1]))
	if v == nil {
		conn.WriteNull()
	}else {
		conn.WriteArray(len(v))
		for _,val := range v {
			conn.WriteBulk(val)
		}
	}
	return nil
}

//hash opt
func hset(s *Server,conn Conn, cmd Command)  error {
	if len(cmd.Args) != 4 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	num,err := s.db.Hset(string(cmd.Args[1]),string(cmd.Args[2]),cmd.Args[3])
	if err != nil {
		conn.WriteError(err.Error())
	}else {
		conn.WriteInt(num)
	}
	return nil
}
func hget(s *Server,conn Conn, cmd Command)  error {
	if len(cmd.Args) != 3 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	v,err := s.db.Hget(string(cmd.Args[1]),string(cmd.Args[2]))
	if err != nil {
		conn.WriteError(err.Error())
	}else {
		conn.WriteBulk(v)
	}
	return nil
}
func hgetall(s *Server,conn Conn, cmd Command)  error {
	if len(cmd.Args) != 2 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	v,err := s.db.Hgetall(string(cmd.Args[1]))
	if err != nil {
		conn.WriteError(err.Error())
	}else {
		conn.WriteArray(len(v) * 2)
		for key,val := range v {
			conn.WriteBulkString(key)
			conn.WriteBulk(val)
		}
	}
	return nil
}

//sort set
func zadd(s *Server,conn Conn, cmd Command)  error {
	if len(cmd.Args) != 4 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	score ,err:= strconv.Atoi(string(cmd.Args[2]))
	if err !=nil {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	num,err := s.db.Zadd(string(cmd.Args[1]),score,string(cmd.Args[3]))
	if err != nil {
		conn.WriteError(err.Error())
	}else {
		conn.WriteInt(num)
	}
	return nil
}
func zrange(s *Server,conn Conn, cmd Command)  error {
	if len(cmd.Args) < 4 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	start,err1 := strconv.Atoi(string(cmd.Args[2]))
	end,err2 := strconv.Atoi(string(cmd.Args[3]))
	if err1 != nil || err2!= nil {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return nil
	}
	v,err := s.db.Zrange(string(cmd.Args[1]),start,end,cmd.Args[4:]...)
	if err != nil {
		conn.WriteError(err.Error())
		return nil
	}
	if v == nil {
		conn.WriteNull()
	}else {
		conn.WriteArray(len(*v))
		for _,val := range *v {
			conn.WriteBulk(val)
		}
	}
	return nil
}

func init()  {
	registerCmd("ping",ping,false)
	registerCmd("select",sselect,false)
	registerCmd("set",set,false)
	registerCmd("get",get,false)
	registerCmd("del",del,true)
	registerCmd("incr",incr,true)
	registerCmd("lpush",lpush,true)
	registerCmd("rpush",rpush,true)
	registerCmd("lpop",lpop,true)
	registerCmd("rpop",rpop,true)
	registerCmd("lrange",lrange,false)
	registerCmd("sadd",sadd,true)
	registerCmd("spop",spop,true)
	registerCmd("smembers",smembers,false)
	registerCmd("mset",mset,false)
	registerCmd("zrange",zrange,false)
	registerCmd("zadd",zadd,true)
	registerCmd("hset",hset,true)
	registerCmd("hget",hget,false)
	registerCmd("hgetall",hgetall,false)
}