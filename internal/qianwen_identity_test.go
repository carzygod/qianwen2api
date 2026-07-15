package internal

import "testing"

func TestParseQianwenIdentity(t *testing.T) {
	tests := []struct {
		name      string
		page      string
		wantID    string
		wantFound bool
	}{
		{
			name:      "logged in",
			page:      `<script>window._USER_ = { avatarUrl: "x", userId: "1222401702212036", aliyunUid: "1222401702212036" };</script>`,
			wantID:    "1222401702212036",
			wantFound: true,
		},
		{
			name:      "logged out",
			page:      `<script>window._USER_ = { avatarUrl: "x", userId: "", aliyunUid: "" };</script>`,
			wantFound: true,
		},
		{
			name: "format changed",
			page: `<html><body>no identity bootstrap</body></html>`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotID, gotFound := parseQianwenIdentity(test.page)
			if gotID != test.wantID || gotFound != test.wantFound {
				t.Fatalf("parseQianwenIdentity() = (%q, %v), want (%q, %v)", gotID, gotFound, test.wantID, test.wantFound)
			}
		})
	}
}
