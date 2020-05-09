package slack

import "testing"

func TestConnectSlack(t *testing.T) {
	ConnectSlack("")
	postMessage("pyspa", "zdoge", "日本語のテスト")
}
