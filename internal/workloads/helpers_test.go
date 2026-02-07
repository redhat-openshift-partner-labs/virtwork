package workloads_test

import (
	"strings"

	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

// parseYAML strips the #cloud-config header and unmarshals the YAML into a map.
func parseYAML(cloudConfig string) map[string]interface{} {
	body := strings.TrimPrefix(cloudConfig, "#cloud-config\n")
	var parsed map[string]interface{}
	ExpectWithOffset(1, yaml.Unmarshal([]byte(body), &parsed)).To(Succeed())
	return parsed
}
