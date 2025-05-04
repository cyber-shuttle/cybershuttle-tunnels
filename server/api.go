package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

type ReservePortRequest struct {
	AgentID string `json:"agent_id"`
}

type ReservePortResponse struct {
	Port    int    `json:"port"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

var portMap = make(map[int]int64)
var portRange = [2]int{10000, 15000}

func ReservePortHandler(w http.ResponseWriter, r *http.Request) {
	var request ReservePortRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if port, err := FindFreePort(portRange[0], portRange[1]); err != nil {
		http.Error(w, "No free ports available", http.StatusInternalServerError)
		return
	} else {
		response := ReservePortResponse{
			Port:    port,
			Success: true,
			Message: "Port reserved successfully",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}

}

func FindFreePort(rangeStart int, rangeEnd int) (int, error) {
	for port := rangeStart; port <= rangeEnd; port++ {

		// check if the por is already in portMap
		if _, exists := portMap[port]; exists {
			// check if the port is expired
			if time.Now().UnixMilli()-portMap[port] < 60000 {
				// port is already reserved
				fmt.Println("Port", port, "is already reserved")

				continue // Skip this port if it's already reserved
			} else {
				// port is expired
				fmt.Println("Port", port, "is expired")
				delete(portMap, port) // Remove the expired port from the map
			}
		}

		if ok := isPortAvailable(port); ok {
			// store current timemills inf portMap[port]
			portMap[port] = time.Now().UnixMilli()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free ports available in the range")
}

func isPortAvailable(port int) bool {
	// Try to listen on the port (localhost:port)
	address := fmt.Sprintf("localhost:%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		// Port is not available
		fmt.Println("Port", port, "is not available:", err)
		return false
	}
	// Close the listener after checking availability
	defer listener.Close()

	// Port is available
	fmt.Println("Port", port, "is available")
	return true
}

func StartAPIServer(port int) {
	http.HandleFunc("/reserve_port", ReservePortHandler)

	fmt.Printf("API server started on port %d\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatalf("Failed to start API server: %v", err)
	}
}
