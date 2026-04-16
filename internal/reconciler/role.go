package reconciler

import (
	"context"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/common"

	"github.com/HertzFlow/chain-reconciler/internal/chain"
	"github.com/HertzFlow/chain-reconciler/internal/config"
	"github.com/HertzFlow/chain-reconciler/internal/metrics"
)

// Roles snapshots RoleStore membership counts (and addresses) for the
// well-known roles. Detection of grants/revokes is done downstream by
// Prometheus via changes(reconciler_role_members_count[...]).
type Roles struct {
	C         *chain.Client
	RoleNames []string
}

func (Roles) Name() string { return "roles" }

func (r Roles) Run(_ context.Context, vc config.VersionConfig) (Report, error) {
	rep := Report{}
	rs := common.HexToAddress(vc.RoleStore)

	for _, name := range r.RoleNames {
		role := chain.RoleHash(name)
		count, err := r.C.GetRoleMemberCount(rs, role)
		if err != nil {
			rep.Findings = append(rep.Findings, fmt.Sprintf("role %s: count err: %v", name, err))
			continue
		}
		metrics.RoleMembersCount.WithLabelValues(vc.Name, name).Set(float64(count))

		// fetch members for human-readable digest; skip enumerate when count is huge
		if count == 0 {
			rep.Lines = append(rep.Lines, fmt.Sprintf("%s: 0 members", name))
			continue
		}
		end := count
		if end > 50 {
			end = 50
		}
		members, err := r.C.GetRoleMembers(rs, role, 0, end)
		if err != nil {
			rep.Findings = append(rep.Findings, fmt.Sprintf("role %s: getRoleMembers err: %v", name, err))
			continue
		}
		// stable order for diffability across runs
		sort.Slice(members, func(i, j int) bool {
			return members[i].Hex() < members[j].Hex()
		})
		rep.Lines = append(rep.Lines, fmt.Sprintf("%s: %d members", name, count))

		// flag privileged roles with too many holders as a finding
		if isPrivilegedRole(name) && count > 5 {
			rep.Findings = append(rep.Findings, fmt.Sprintf("%s has %d members — privileged role, expected ≤5", name, count))
		}
	}
	metrics.ReconcilerLastRunTimestamp.WithLabelValues("roles_" + vc.Name).SetToCurrentTime()
	return rep, nil
}

func isPrivilegedRole(name string) bool {
	switch name {
	case "CONFIG_KEEPER", "TIMELOCK_ADMIN", "TIMELOCK_MULTISIG", "GOV_TOKEN_CONTROLLER":
		return true
	}
	return false
}
