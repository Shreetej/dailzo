package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"firebase.google.com/go/v4/messaging"
	fcm "github.com/appleboy/go-fcm"
	// "github.com/appleboy/go-fcm/v4"
)

func SendNotification(token string, title string, body string) {
	ctx := context.Background()
	client, err := fcm.NewClient(ctx, fcm.WithCredentialsFile("utils/dailzo-firebase-adminsdk-fbsvc-75618272e7.json"))
	if err != nil {
		log.Fatalf("Error creating FCM client: %v", err)
	}

	resp, err := client.Send(
		ctx, &messaging.Message{
			Token: token,
			Notification: &messaging.Notification{
				Title: title,
				Body:  body,
			},
			Data: map[string]string{
				"foo": "bar",
			},
		},
	)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Response: %+v\n", resp)
}

func SendFCMNotification(token string, title string, body string) {
	url := "https://fcm.googleapis.com/fcm/send"
	// serverKey := "MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDHJRP7oXQPS64X\nQpdNrd/9/6iKGM/G3SGPJoOiy2nqznZgEQCNramu/hbxbP6TmYrqLTjvUUxjQzVk\nkRk5jEA3WhKuM0zJO83V/4M+pw+Sgo4/HlxyrQncEWPEhMFGW85ds0QBogPs1yWP\nS6rs4VaJUsyGUdPBesAoeEKzHXy9GqUEjB98C6M0auR3BV6yM1EIqaDSX+TUdud8\nLWFLnTNhGNrFTJNFmR6kfjX6a4BnldHUm6OBjmT1Lip5TLJBxkEsuVsAMvovR1sd\noQ7IRdmX/9+V1AzSLJRu3gqThXOZguYJgOatOrmd15QkYcdM+2WtUVrzuQTNAc2L\nkScBhqO/AgMBAAECggEAOYTTFb3XWMeiG+PG8c/Dv4hFNMXf1W9vTvpv0Ohxyjkh\n5bNjdohvVzTaiZbSnO0fO24eYLkZmB4EYOyr3XxN2+7vMFNu3TyhwiqPzNvR2p5M\n2Zw6VYD3NRHswAkcDedcXqH1hYc3HQQzPAU99DdNmFJK8ozV3a9Hqyi5EyT0L2Xt\nJcBvUpfyKF95greXqveb86udzW0wA3CiWLBBVIQEUgOw3h1Q/8OjvJ3YJ8impTQ2\nt8+L4alq3RIdC2rOYPrHpC8FVCsRVbioujih1igk/xndWCZtSIkCmieu5HTlntVb\nNNoTsWzZ1GihskLRPJ29Lks3dGK7U1bVjtE9q0mJiQKBgQDtVZ5OchmAJJePMuio\nTQtoyGBasgBjpRL+lMyvY8vFuQyx0wzqNHbj0HCgWz3zriEfYbYVxbGb+7QmbFit\n4kwAjrzep6e7ozY1Y5IiXg288lbbOYAbPnT/UeETCMkWT6IY2MRJteuG+/95qKj9\nubAJxlx2fgHDYosXeryGXlR9ZQKBgQDWzpFZttlx8b01gjNR3K3hF9BmzObV5+fv\npeYY/YGY2jVsh4pCLG4zvPkOylKkDkEvMicVKZ745dOEfxmCTMV6sM2TqLlCrMTF\nH/jS+ozI9LYP1694QHHLrmWQCgdUAEeVx2EWXb98Z0BAh0O6gObZhzoPBqON/TMJ\nkao/4MhMUwKBgQDMkIeqxd4E/YVFAHRY3E+BOXUTt2luedItbMQgSLxS6HVwsKDp\nHd977SWmkf6MEwKpseboTUYRVJqqo6ir9+nacS2KHKgOq1cGHZTP2pGs0pTa0G0D\nDop5p3GAnon0mR72m6BUiGFCL+K8UguW5n49bqQz3dhXOJD64+erSZM5oQKBgFVG\nQfuMlDwgzI1Od9MauUhvrMLyuvzWCIRhprvq/6TPk3/XOvLUMpeFgJX3ieEo64Wx\n8kP7dum2S0cBMf5BPfBb+fCRfaJTdfYPoDcZUgSA6TnW1Qj3BHXocNdCs/AMAF8c\nfJVleBwJ3T8As8l5XKukfE70wr8eckFtO1oKgVmTAoGAHCJ3J70hsH0yhAfWbCjn\nSN4mti2w7hzMvhxLqb9oiAZ8xZtpGzYuFTkxnqlY3G0B5DTRxBZ7zFtM/je7qNXk\nZT2IFXwKJdrY/sTl/Xhdx9xm88fDx9Ccc6UErX0bsTeA0XVWk2x4b6MU7kSxps/8\nnGghXQozP/4ay5Fb1yyS1e4="
	serverKey := "AIzaSyCYV2_TgHZZkOFAkDcasP08KrijbB7cCk4"

	payload := map[string]any{
		"to": token,
		"notification": map[string]string{
			"title": title,
			"body":  body,
		},
		"data": map[string]any{
			"key": "value",
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshalling payload:", err)
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	req.Header.Set("Authorization", serverKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Response Status:", resp.Status)
}
