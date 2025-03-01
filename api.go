package main

import (
	"encoding/json"
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

	req, err := http.NewRequest("GET", serviceApi+"/profile", nil)
	if err != nil {
		return ResponseData{}, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", xApiKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	logrus.Info("Response:", resp)
	logrus.Info("Response body:", resp.Body)
	logrus.Info("Response status:", resp.Status)
	if err != nil {
		return ResponseData{}, err
	}

	defer resp.Body.Close()
	var Response ResponseData
	err = json.NewDecoder(resp.Body).Decode(&Response)
	if err != nil {
		return ResponseData{}, err
	}
	logrus.Info("Response Data:", Response)

	if resp.StatusCode != http.StatusOK {
		logrus.Errorf("Failed to fetch user info, status: %d", resp.StatusCode)
		return ResponseData{}, err
	}

	// Cetak hasilnya (bisa diubah sesuai kebutuhan)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("Failed to read response body: %v", err)
		return ResponseData{}, err
	}

	// Log the response body for debugging
	logrus.Infof("User Info Response: %s", string(body))

	return Response, nil
}
