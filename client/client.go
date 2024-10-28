package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

const serverAddress = "localhost:4000"
const numKeys = 100
const API_KEY = "penguins"

func main() {
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()


	basicTests(conn)
}

// Perform basic SET, GET, DEL operations for single-threaded testing
func basicTests(conn net.Conn) {
	fmt.Println("Running basic tests...")

	// SET a key
	setResponse := set(conn, "foo", "bar", 0)
	if setResponse != "OK" {
		fmt.Println("TEST FAILED: SET failed, response:", setResponse)
		return
	}
	fmt.Println("SET passed")

	// GET the same key
	getResponse := GET(conn, "foo")
	if getResponse != "bar" {
		fmt.Println("TEST FAILED: GET failed, expected 'bar', got:", getResponse)
		return
	}
	fmt.Println("GET passed")

	// DEL the key
	delResponse := DEL(conn, "foo")
	if delResponse != "OK" {
		fmt.Println("TEST FAILED: DEL failed, response:", delResponse)
		return
	}
	fmt.Println("DEL passed")

	// GET the key again (should be missing)
	getResponse = GET(conn, "foo")
	if getResponse != "-ERR " {
		fmt.Println("TEST FAILED: GET missing key failed, expected '!OK', got:", getResponse)
		return
	}
	fmt.Println("GET missing key test passed")
}

// Stress test: Create multiple goroutines to perform concurrent GET/SET operations
func stressTest() {
	fmt.Println("Running stress test with 10 concurrent clients...")

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			conn, err := net.Dial("tcp", serverAddress)
			if err != nil {
				fmt.Println("Error connecting to server:", err)
				return
			}
			defer conn.Close()

			for j := 0; j < numKeys; j++ {
				key := fmt.Sprintf("key%d_client%d", j, clientID)
				value := fmt.Sprintf("value%d", j)

				// SET the key
				setResponse := set(conn, key, value, 0)
				if setResponse != "OK" {
					fmt.Printf("TEST FAILED: Client %d SET key %s failed, response: %s\n", clientID, key, setResponse)
					return
				}

				// GET the key and verify the value
				getResponse := GET(conn, key)
				if getResponse != value {
					fmt.Printf("TEST FAILED: Client %d GET key %s failed, expected %s, got %s\n", clientID, key, value, getResponse)
					return
				}
				fmt.Printf("Client %d GET key %s passed\n", clientID, key)
			}
		}(i)
	}
	wg.Wait()
	fmt.Println("Stress test complete.")
}

func ttlExpiryTest() bool {
	fmt.Println("Running TTL expiry test...")
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return false
	}
	defer conn.Close()

	// SET a key with 5 seconds TTL
	setResponse := set(conn, "temp_key", "temp_value", 5)
	if setResponse != "OK" {
		fmt.Println("TEST FAILED: SET failed, response:", setResponse)
		return false
	}
	fmt.Println("SET passed")

	// Verify the key exists initially
	getResponse := GET(conn, "temp_key")
	if getResponse != "temp_value" {
		fmt.Println("TEST FAILED: GET before expiry failed, expected 'temp_value', got:", getResponse)
		return false
	}
	fmt.Println("GET before expiry passed")

	// Wait for 6 seconds for the key to expire
	time.Sleep(6 * time.Second)

	// Check if the key has expired
	getResponse = GET(conn, "temp_key")
	if getResponse != "!OK" {
		fmt.Println("TEST FAILED: GET after expiry failed, expected '!OK', got:", getResponse)
		return false
	}
	fmt.Println("GET after expiry passed")

	return true // Test passed
}

// Test deleting a non-existent key
func deleteNonExistentKeyTest() bool {
	fmt.Println("Running delete non-existent key test...")
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return false
	}
	defer conn.Close()

	// Attempt to delete a key that doesn't exist
	delResponse := DEL(conn, "non_existent_key")
	if delResponse != "OK" { // Adjust this based on your expected response for DEL
		fmt.Println("TEST FAILED: DEL non-existent key failed, response:", delResponse)
		return false
	}
	fmt.Println("DEL non-existent key test passed")
	return true // Test passed
}

// Helper function to send a SET command to the server
func set(conn net.Conn, key string, value string, ttl uint32) string {
  sendCommand(conn, API_KEY) 
	sendCommand(conn, "SET")

	sendLength(conn, uint32(len(key)))  // Send key length
	conn.Write([]byte(key))             // Send the key itself

	sendLength(conn, uint32(len(value))) // Send value length
	conn.Write([]byte(value))            // Send the value itself

	sendLength(conn, ttl) // Send TTL

	return readResponse(conn)
}

// Helper function to send a GET command to the server
func GET(conn net.Conn, key string) string {
  sendCommand(conn, API_KEY) 
	sendCommand(conn, "GET")
	sendLength(conn, uint32(len(key)))  // Send key length
	conn.Write([]byte(key))             // Send the key itself
	return readResponse(conn)
}

// Helper function to send a DEL command to the server
func DEL(conn net.Conn, key string) string {
  sendCommand(conn, API_KEY) 
	sendCommand(conn, "DEL")
	sendLength(conn, uint32(len(key)))
	conn.Write([]byte(key))
	return readResponse(conn)
}

// Helper function to send the command to the server
func sendCommand(conn net.Conn, command string) {
	conn.Write([]byte(command))
}

// Helper function to send the length of a key or value
func sendLength(conn net.Conn, length uint32) {
	lengthBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBuf, length)
	conn.Write(lengthBuf)
}

// Helper function to read the server's response
func readResponse(conn net.Conn) string {
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading from server:", err)
		return "-ERR"
	}
	return string(buf[:n])
}

