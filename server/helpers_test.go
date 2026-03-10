package main

type mockNotifier struct {
	startCalls      int
	updateCalls     int
	endCalls        int
	silentPushCalls int
	alertPushCalls  int

	lastSilentToken string
	lastAlertToken  string
	lastAlertTitle  string
	lastAlertBody   string
}

func (m *mockNotifier) StartActivity(token, title string, startedAt int64) error {
	m.startCalls++
	return nil
}
func (m *mockNotifier) UpdateActivity(token, title string, startedAt int64) error {
	m.updateCalls++
	return nil
}
func (m *mockNotifier) EndActivity(token string) error {
	m.endCalls++
	return nil
}
func (m *mockNotifier) SendSilentPush(deviceToken string, taskTitle string, startedAtMs int64) error {
	m.silentPushCalls++
	m.lastSilentToken = deviceToken
	return nil
}
func (m *mockNotifier) SendAlertPush(deviceToken string, title string, body string) error {
	m.alertPushCalls++
	m.lastAlertToken = deviceToken
	m.lastAlertTitle = title
	m.lastAlertBody = body
	return nil
}
