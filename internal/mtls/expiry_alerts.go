package mtls

import (
	"fmt"

	"github.com/PavelNeyman/netductor/internal/notify"
)

// AlertExpiringCerts fires once/day-ish when plane or client certs expire within days.
func AlertExpiringCerts(withinDays int) {
	if withinDays < 1 {
		withinDays = 30
	}
	for _, pc := range ListPlaneCerts() {
		if pc.DaysLeft < 0 {
			notify.AlertOnce("mtls:expired:"+pc.Name, fmt.Sprintf("🔴 mTLS %s expired", pc.Name))
		} else if pc.DaysLeft <= withinDays {
			notify.AlertOnce(
				fmt.Sprintf("mtls:expiring:%s:%d", pc.Name, pc.DaysLeft/7),
				fmt.Sprintf("⚠️ mTLS %s expires in %d days (%s)", pc.Name, pc.DaysLeft, pc.NotAfter.Format("2006-01-02")),
			)
		}
	}
	clients, _ := ListClientCerts()
	for _, c := range clients {
		if c.DaysLeft < 0 {
			notify.AlertOnce("mtls:client-expired:"+c.NodeID, fmt.Sprintf("🔴 mTLS client %s expired", c.NodeID))
		} else if c.DaysLeft <= withinDays {
			notify.AlertOnce(
				fmt.Sprintf("mtls:client-expiring:%s:%d", c.NodeID, c.DaysLeft/7),
				fmt.Sprintf("⚠️ mTLS client %s expires in %d days", c.NodeID, c.DaysLeft),
			)
		}
	}
}
