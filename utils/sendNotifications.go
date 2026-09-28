package utils

import (
	"context"
	"fmt"

	"firebase.google.com/go/v4/messaging"
	fcm "github.com/appleboy/go-fcm"
)

// SendNotification pushes an FCM message to one device token. Returns an
// error instead of calling log.Fatal(f) on failure — this used to crash the
// entire backend process (taking down every request in flight, not just this
// notification) on something as routine as a missing local credentials file
// or an expired device token. Requires real Firebase project credentials in
// utils/dailzo-firebase-adminsdk-fbsvc-75618272e7.json, which don't exist in
// this repo (env/credential provisioning, out of scope here) — but the
// per-user token this sends to now has somewhere to come from: see
// users.fcm_token (migrations/017_user_fcm_token.sql) and
// PatchAuthFCMToken in internal/server/auth_handlers.go.
func SendNotification(token string, title string, body string) error {
	ctx := context.Background()
	client, err := fcm.NewClient(ctx, fcm.WithCredentialsFile("utils/dailzo-firebase-adminsdk-fbsvc-75618272e7.json"))
	if err != nil {
		return fmt.Errorf("create FCM client: %w", err)
	}

	resp, err := client.Send(
		ctx, &messaging.Message{
			Token: token,
			Notification: &messaging.Notification{
				Title: title,
				Body:  body,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("send FCM notification: %w", err)
	}
	fmt.Printf("Response: %+v\n", resp)
	return nil
}
