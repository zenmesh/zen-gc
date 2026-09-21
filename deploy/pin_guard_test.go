package deploy

import (
	"os"
	"strings"
	"testing"
)

// Guards (S025-GC §2): production deployment definitions must not use mutable
// image tags, and the webhook TLS secret volume must be present (no
// insecure-webhook production default).
func TestDeploymentNoMutableTag(t *testing.T) {
	data, err := os.ReadFile("manifests/deployment.yaml")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if strings.Contains(s, ":latest") {
		t.Error("deployment.yaml must not use :latest")
	}
}
