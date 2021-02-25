package platformoperator

import (
	"fmt"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/flanksource/karina/pkg/constants"
	"github.com/flanksource/karina/pkg/platform"
)

const Namespace = constants.PlatformSystem

func Install(platform *platform.Platform) error {
	if platform.PlatformOperator.IsDisabled() {
		return platform.DeleteSpecs(Namespace, "platform-operator.yaml")
	}

	labels := map[string]string{
		"control-plane":            "controller-manager",
		"quack.pusher.com/enabled": "true",
	}
	if err := platform.CreateOrUpdateNamespace(constants.PlatformSystem, labels, nil); err != nil {
		return err
	}

	var secrets = make(map[string][]byte)

	secrets["AWS_ACCESS_KEY_ID"] = []byte(platform.S3.AccessKey)
	secrets["AWS_SECRET_ACCESS_KEY"] = []byte(platform.S3.SecretKey)

	if platform.Ldap != nil {
		secrets["LDAP_USERNAME"] = []byte(platform.Ldap.Username)
		secrets["LDAP_PASSWORD"] = []byte(platform.Ldap.Password)
	}

	if err := platform.CreateOrUpdateSecret("secrets", constants.PlatformSystem, secrets); err != nil {
		return err
	}

	if platform.PlatformOperator.WhitelistedPodAnnotations == nil {
		platform.PlatformOperator.WhitelistedPodAnnotations = []string{"com.flanksource.infra.logs/enabled", "co.elastic.logs/enabled"}
	}

	if platform.PlatformOperator.Args == nil {
		platform.PlatformOperator.Args = make(map[string]string)
	}
	args := platform.PlatformOperator.Args
	args["annotations"] = strings.Join(platform.PlatformOperator.WhitelistedPodAnnotations, ",")
	args["oauth2-proxy-service-name"] = "oauth2-proxy"
	args["oauth2-proxy-service-namespace"] = "ingress-nginx"
	args["domain"] = platform.Domain
	args["enable-cluster-resource-quota"] = fmt.Sprintf("%v", platform.PlatformOperator.EnableClusterResourceQuota)

	v, _ := semver.Parse(strings.TrimLeft(platform.PlatformOperator.Version, "v"))
	expectedRange, _ := semver.ParseRange(">= 0.5.0")
	if expectedRange(v) {
		if platform.PlatformOperator.DefaultRegistry != "" {
			platform.PlatformOperator.RegistryWhitelist = append(platform.PlatformOperator.RegistryWhitelist, platform.PlatformOperator.DefaultRegistry)
		}
		args["registry-whitelist"] = strings.Join(platform.PlatformOperator.RegistryWhitelist, ",")
		args["default-image-pull-secret"] = platform.PlatformOperator.DefaultImagePullSecret
		args["default-registry-prefix"] = platform.PlatformOperator.DefaultRegistry
	}
	platform.PlatformOperator.Args = args
	return platform.ApplySpecs("", "platform-operator.yaml")
}
