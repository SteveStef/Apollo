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
const MAX_CONNECTIONS = 100
const MAX_KEY_SIZE = 256
const MAX_VALUE_SIZE = 10240 // 10KB
const MAX_QUEUE_SIZE = 50

var waitingQueue = make(chan net.Conn, MAX_QUEUE_SIZE)
var currentConnections = 0
var apiKey string
var globalMu sync.Mutex

type Shard struct {
  cache map[string]string
  times map[string]int64
  mu sync.RWMutex
}

var shards [SHARD_COUNT]*Shard
func init() {
  for i:=0;i<SHARD_COUNT;i++ {
    shards[i] = &Shard {cache: make(map[string]string), times: make(map[string]int64),}
  }
  startCleanupTask()
}

func main() {
  fmt.Println("Hello World");
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

  go processQueue()
  for {
    conn, err := server.Accept()
    if err != nil {
      fmt.Println(err)
      continue
    }

		globalMu.Lock()
    if currentConnections >= MAX_CONNECTIONS {
      if len(waitingQueue) < MAX_QUEUE_SIZE {
        fmt.Println("Queueing connection", conn.RemoteAddr())
        waitingQueue <- conn // Add connection to queue
      } else {
        fmt.Println("Connection queue full. Rejecting", conn.RemoteAddr())
        conn.Close() // Queue is full, reject the connection
      }
    } else { // not at capacity
      currentConnections++
      globalMu.Unlock()

      go func() {
        handleConnection(conn)
        globalMu.Lock()
        currentConnections--
        globalMu.Unlock()
      }()
    }
  }

}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Printf("Serving %s\n", conn.RemoteAddr().String())

	reader := bufio.NewReader(conn)

	for {
		token := make([]byte, len(apiKey))
		_, err := reader.Read(token)
		if err != nil {
			fmt.Println("User has disconnected")
      return
		}

    if string(token) != apiKey {
      conn.Write([]byte("-ERR 1001 access denied"))
      return
    }

		command := make([]byte, 3)
		_, err = reader.Read(command)
		if err != nil {
			fmt.Println("User has disconnected")
      return
		}
    
    cmdStr := string(command)
    fmt.Println("command: " + cmdStr)

		switch cmdStr {
		case "GET":
			result := handleGet(reader)
			conn.Write([]byte(result))
		case "SET":
			result := handlePost(reader)
			conn.Write([]byte(result))
		case "DEL":
			result := handleDel(reader)
			conn.Write([]byte(result))
    case "RAL":
      result := handleRemoveAll()
      conn.Write([]byte(result))
		default:
			conn.Write([]byte("-ERR unknown command\n"))
		}
    printShardsWithTTL()
	}
}

func handleRemoveAll() string {
  for _, shard := range shards {
    shard.mu.Lock()
    shard.cache = make(map[string]string)
    shard.times = make(map[string]int64)
    shard.mu.Unlock()
  }
  return "OK"
}

func printShardsWithTTL() {
  for i, shard := range shards {
    shard.mu.RLock() // Read lock to safely access shard data
    
    if len(shard.cache) != 0 {
      fmt.Printf("Shard %d: %s", i, " ")
    }
    for key, value := range shard.cache {
      ttl, hasTTL := shard.times[key]
      if hasTTL {
        fmt.Printf("  Key: %s, Value: %s, TTL: %d (expires in %d seconds)\n", key, value, ttl, ttl-time.Now().Unix())
      } else {
        fmt.Printf("  Key: %s, Value: %s, TTL: None\n", key, value)
      }
    }
    
    shard.mu.RUnlock()
  }
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
  shard.mu.RLock()
  ttl, hasTTL := shard.times[skey]

  if hasTTL && ttl <= time.Now().Unix() {
    delete(shard.cache, skey)
    delete(shard.times, skey)
    shard.mu.RUnlock()
    return "-ERR 1004 Key not found (expired)"
  }

  value, exists := shard.cache[skey]
  shard.mu.RUnlock()

  if !exists{
    return "-ERR 1004 Key not found"
  }

  return value
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

  shard.mu.Lock()
  shard.cache[skey] = svalue

  if ttl > 0 {
    shard.times[skey] = time.Now().Unix() + int64(ttl)
  }

  shard.mu.Unlock()

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

	shard.mu.Lock()
	delete(shard.cache, skey)
	delete(shard.times, skey)
	shard.mu.Unlock()

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
      time.Sleep(1 * time.Minute)  // Run cleanup every minute
      for _, shard := range shards {
        expiredKeys := []string{}
        shard.mu.RLock()
        now := time.Now().Unix()
        for key, expiration := range shard.times {
          if expiration <= now {
            expiredKeys = append(expiredKeys, key)
          }
        }
        shard.mu.RUnlock()
        if len(expiredKeys) > 0 {
          shard.mu.Lock()
          for _, key := range expiredKeys {
            delete(shard.cache, key)
            delete(shard.times, key)
          }
          shard.mu.Unlock()
        }
      }
    }
  }()
}

func processQueue() {
  for conn := range waitingQueue {
    globalMu.Lock()
    if currentConnections < MAX_CONNECTIONS {
      currentConnections++
      globalMu.Unlock()

      go func() {
        handleConnection(conn)
        globalMu.Lock()
        currentConnections--
        globalMu.Unlock()
      }()
    } else {
      globalMu.Unlock()
      time.Sleep(100 * time.Millisecond)
    }
  }
}
/*
  Why this is better than REDIS
  1. Multithreaded
  2. Needs Mutex locks
  3. Has sharding within a single instance
  4. TTL
  5. Simple Protocol
  6. API_KEY only access
  7. 100 concurrent request limit
  8. Secure cleanup process
  9. input sanitization
  10. Has a queue of people when at max capacity
  11. The only thing left to do is make this a TLS server
*/
