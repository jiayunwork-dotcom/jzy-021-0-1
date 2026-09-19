package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	dataDir := getenv("CPM_DATA_DIR", "./data")
	addr := getenv("CPM_ADDR", ":8080")
	store, err := NewStore(dataDir)
	if err != nil {
		log.Fatalf("打开作业库失败: %v", err)
	}
	log.Printf("CPM 计划服务监听 %s，作业库目录 %s", addr, dataDir)
	log.Fatal(http.ListenAndServe(addr, newServer(store)))
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
