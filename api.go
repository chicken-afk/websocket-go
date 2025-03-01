package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
)

type ResponseData struct {
	Status  int      `json:"status"`
	Message string   `json:"message"`
	Data    UserInfo `json:"data"`
}

type UserInfo struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

func GetUserInfoByToken(token string) (ResponseData, error) {
	var serviceApi = os.Getenv("BACKEND_API")
	var xApiKey = os.Getenv("X_API_KEY")

	logrus.Info("Service API:", serviceApi)
	logrus.Info("X-API-KEY:", xApiKey)
	logrus.Info("Token:", token)

	// Membuat request HTTP
	req, err := http.NewRequest("GET", serviceApi+"/profile", nil)
	if err != nil {
		return ResponseData{}, err
	}

	// Menambahkan header Authorization dan lainnya
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", xApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ResponseData{}, err
	}
	defer resp.Body.Close()

	// Cek status code terlebih dahulu
	if resp.StatusCode != http.StatusOK {
		logrus.Errorf("Failed to fetch user info, status: %d", resp.StatusCode)
		body, _ := io.ReadAll(resp.Body) // Baca body untuk log error
		logrus.Infof("Error Response Body: %s", string(body))
		return ResponseData{}, fmt.Errorf("failed to fetch user info with status: %d", resp.StatusCode)
	}

	// Membaca body untuk debugging dan logging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("Failed to read response body: %v", err)
		return ResponseData{}, err
	}

	// Log the response body for debugging
	logrus.Infof("User Info Response: %s", string(body))

	// Dekode JSON ke struct ResponseData
	var response ResponseData
	err = json.Unmarshal(body, &response)
	if err != nil {
		logrus.Errorf("Failed to decode JSON response: %v", err)
		return ResponseData{}, err
	}

	logrus.Info("Response Data:", response)
	return response, nil
}
