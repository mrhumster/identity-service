// Package metrics holds identity-service business metric counters.
package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	LoginAttempts = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "identity_login_attempts_total",
		Help: "Total number of login attempts by result.",
	}, []string{"result"})

	UsersCreated = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "identity_users_created_total",
		Help: "Total number of users created.",
	})

	TokensRefreshed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "identity_tokens_refreshed_total",
		Help: "Total number of successful access token refreshes.",
	})
)

func init() {
	prometheus.MustRegister(LoginAttempts, UsersCreated, TokensRefreshed)
}