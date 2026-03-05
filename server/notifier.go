package main

// Notifier sends Live Activity push notifications via APNs.
type Notifier interface {
	StartActivity(token string, taskTitle string, startedAtMs int64) error
	UpdateActivity(token string, taskTitle string, startedAtMs int64) error
	EndActivity(token string) error
}
