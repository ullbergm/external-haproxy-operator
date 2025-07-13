package monitoring

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	ruleName                     = "external-haproxy-operator-rules"
	alertRuleGroup               = "external-haproxy-operator.rules"
	haProxyClientErrorCountAlert = "HAProxyClientErrorCountTotal"
	runbookURLBasePath           = "https://github.com/ullbergm/external-haproxy-operator/tree/master/docs/monitoring/runbooks/"
)

// NewPrometheusRule creates new PrometheusRule(CR) for the operator to have alerts and recording rules
func NewPrometheusRule(namespace string) *monitoringv1.PrometheusRule {
	return &monitoringv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monitoringv1.SchemeGroupVersion.String(),
			Kind:       "PrometheusRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleName,
			Namespace: namespace,
		},
		Spec: *NewPrometheusRuleSpec(),
	}
}

// NewPrometheusRuleSpec creates PrometheusRuleSpec for alerts and recording rules
func NewPrometheusRuleSpec() *monitoringv1.PrometheusRuleSpec {
	return &monitoringv1.PrometheusRuleSpec{
		Groups: []monitoringv1.RuleGroup{{
			Name: alertRuleGroup,
			Rules: []monitoringv1.Rule{
				createhaProxyClientErrorCountAlertRule(),
			},
		}},
	}
}

// createhaProxyClientErrorCountAlertRule creates HAProxyClientErrorCountTotal alert rule that triggers when the HAProxy client error count exceeds a threshold for the last 5 minutes.
func createhaProxyClientErrorCountAlertRule() monitoringv1.Rule {
	return monitoringv1.Rule{
		Alert: haProxyClientErrorCountAlert,
		Expr:  intstr.FromString("increase(haproxy_client_errors_count_total[5m]) > 0"),
		Annotations: map[string]string{
			"description": "HAProxy client has encountered errors in the last 5 minutes.",
		},
		Labels: map[string]string{
			"severity":    "warning",
			"runbook_url": runbookURLBasePath + "HAProxyClientErrorCount.md",
		},
	}
}
