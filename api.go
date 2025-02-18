package main

import (
	"encoding/json"
	"io"
	"net/http"

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

var serviceApi = "http://localhost:8080/api/v1"
var xApiKey = "1234567890"

func GetUserInfoByToken(token string) (ResponseData, error) {
	req, err := http.NewRequest("GET", serviceApi+"/profile", nil)
	if err != nil {
		return ResponseData{}, err
	}

	req.Header.Set("Authorization", token)
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
