package main

import (
	"github.com/joho/godotenv"
  "encoding/binary"
  "hash/fnv"
  "bufio"
  "sync"
  "time"
  "fmt"
  "net"
  "io"
  "os"
)

const SHARD_COUNT = 16
const MAX_KEY_SIZE = 256
const MAX_VALUE_SIZE = 10240
const POOL_SIZE = 3

var shards [SHARD_COUNT]*Shard
var apiKey string

type Shard struct {
  cache sync.Map
  times sync.Map
}

func init() {
  for i := 0; i < SHARD_COUNT; i++ {
    shards[i] = &Shard{}
  }
  startCleanupTask()
}

func main() {
  PORT := ":4000"
  server, err := net.Listen("tcp4", PORT)
  if err != nil {
    fmt.Println(err);
  }
  defer server.Close()

	err = godotenv.Load()
	if err != nil {
    fmt.Println("Error loading .env file")
    return
	}

	apiKey = os.Getenv("API_KEY")
  fmt.Println("listening on PORT=" + PORT)

  for {
    conn, err := server.Accept()
    if err != nil {
      fmt.Println(err)
      continue
    }
    handleConnection(conn) // only have 1 connection at a time
  }
}


func handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Printf("Serving %s\n", conn.RemoteAddr().String())

  jobs := make(chan string, 100) // either GET DEL SET RAL
  results := make(chan string, 100)

	reader := bufio.NewReader(conn)
  for i := 0; i < POOL_SIZE; i++ {
    go worker(jobs, results, conn, reader)
  }

	for {
		token := make([]byte, len(apiKey))
		_, err := reader.Read(token)
		if err != nil {
			fmt.Println("User has disconnected")
      return
		}

    if string(token) != apiKey {
      conn.Write([]byte("-ERR 1001 access denied\n"))
      return
    }

		command := make([]byte, 3)
		_, err = reader.Read(command)
		if err != nil {
			fmt.Println("User has disconnected")
      return
		}

    jobs <- string(command)
    response := <- results
    conn.Write([]byte(response))
	}
}

func worker(jobs <-chan string, results chan<- string, conn net.Conn, reader *bufio.Reader) {
  for command := range jobs {
    switch command {
    case "GET":
      results <- handleGet(reader)
    case "SET":
      results <- handlePost(reader)
    case "DEL":
      results <- handleDel(reader)
    case "RAL":
      results <- handleRemoveAll()
    default:
      results <- "-ERR unknown command\n"
    }
  }
}

func handleRemoveAll() string {
  for _, shard := range shards {
    shard.cache.Range(func(key, _ interface{}) bool {
      shard.cache.Delete(key)
      return true
    })
    shard.times.Range(func(key, _ interface{}) bool {
      shard.times.Delete(key)
      return true
    })
  }
  return "OK"
}

func handleGet(reader *bufio.Reader) string {
	lengthBuf := make([]byte, 4) // length of the key
	_, err := reader.Read(lengthBuf)
	if err != nil {
		return "-ERR 1003 failed to read key length"
	}

  keyLength := int(binary.BigEndian.Uint32(lengthBuf))
  if keyLength >= MAX_KEY_SIZE || keyLength <= 0 {
    return "-ERR 1001 Invalid key length"
  }

  key := make([]byte, keyLength)
  _, err = reader.Read(key)
  if err != nil {
    return "-ERR 1003 failed to read key length"
  }

  skey := string(key)

  shard := getShard(skey)
  ttl, hasTTL := shard.times.Load(skey)

  if hasTTL && ttl.(int64) <= time.Now().Unix() {
    shard.cache.Delete(skey)
    shard.times.Delete(skey)
    return "-ERR 1004 Key not found (expired)"
  }

  value, exists := shard.cache.Load(skey)
  if !exists {
    return "-ERR 1004 Key not found"
  }

  return value.(string)
}

func handlePost(reader *bufio.Reader) string {
  lengthBuf := make([]byte, 4)
  _, err := reader.Read(lengthBuf)
  if err != nil {
    if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
      return "-ERR timeout waiting for key length"
    }
    return "-ERR failed to read key length"
  }

  keyLength := int(binary.BigEndian.Uint32(lengthBuf))
  if keyLength >= MAX_KEY_SIZE || keyLength <= 0 {
    return "-ERR 1001 Invalid key length"
  }

  key := make([]byte, keyLength)
  _, err = reader.Read(key)
  if err != nil {
    return "-ERR failed to read key"
  }

  _, err = reader.Read(lengthBuf)
  if err != nil {
    return "-ERR failed to read value length"
  }

  valueLength := int(binary.BigEndian.Uint32(lengthBuf))
  if valueLength >= MAX_VALUE_SIZE || valueLength <= 0 {
    return "-ERR 1001 Invalid value length"
  }

  value := make([]byte, valueLength)

  _, err = reader.Read(value)
  if err != nil {
    return "-ERR failed to read value"
  }

  // Optional Step 5: Read TTL (Time-to-Live) as a 4-byte integer
  _, err = reader.Read(lengthBuf)
  ttl := 0
  if err == nil {
    ttl = int(binary.BigEndian.Uint32(lengthBuf))
  } else if err != io.EOF {
    return "-ERR failed to read TTL"
  }

  skey := string(key)
  svalue := string(value)
  shard := getShard(skey)

  shard.cache.Store(skey, svalue)
  if ttl > 0 {
    shard.times.Store(skey, time.Now().Unix()+int64(ttl))
  } else {
    shard.times.Delete(skey)
  }

  return "OK"
}


func handleDel(reader *bufio.Reader) string {
	lengthBuf := make([]byte, 4) // length of the key
	_, err := reader.Read(lengthBuf)
	if err != nil {
		return "-ERR failed to read key length"
	}

  keyLength := int(binary.BigEndian.Uint32(lengthBuf))
  if keyLength >= MAX_KEY_SIZE || keyLength <= 0 {
    return "-ERR 1001 Invalid key length"
  }

  key := make([]byte, keyLength)
  _, err = reader.Read(key)
  if err != nil {
    return "-ERR failed to read key"
  }

  skey := string(key)
  shard := getShard(skey)

  shard.cache.Delete(skey)
  shard.times.Delete(skey)

	return "OK"
}


func getShard(key string) *Shard {
  hasher := fnv.New32()
  hasher.Write([]byte(key))
  shardIndex := hasher.Sum32() % SHARD_COUNT
  return shards[shardIndex]
}


func startCleanupTask() {
  go func() {
    for {
      time.Sleep(60 * time.Second)
      now := time.Now().Unix()

      for _, shard := range shards {
        shard.times.Range(func(key, value interface{}) bool {
          if value.(int64) <= now {
            shard.cache.Delete(key)
            shard.times.Delete(key)
          }
          return true
        })
      }
    }
  }()
}

