package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	// читаємо тіло запиту
	_, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// декодуємо як зображення
	_, _, err = image.DecodeConfig(r.Body) //❌ не спрацює, бо .Body вже прочитане
}

func main() {
	http.HandleFunc("/findBounds", uploadHandler)
	fmt.Println("Server started at :2224")
	log.Fatal(http.ListenAndServe(":2224", nil))
}
