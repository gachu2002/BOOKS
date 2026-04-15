package distsys

type LWWWrite struct {
	Actor       string
	Value       string
	TimestampMS int64
}

type LWWScenario struct {
	Earlier       LWWWrite
	Later         LWWWrite
	Winner        LWWWrite
	LostUpdateFor string
}

// ClockSkewScenario shows how wall-clock timestamps can violate causal order.
func ClockSkewScenario() LWWScenario {
	earlier := LWWWrite{Actor: "client-a", Value: "x=1", TimestampMS: 42004}
	later := LWWWrite{Actor: "client-b", Value: "x=2", TimestampMS: 42003}

	return LWWScenario{
		Earlier:       earlier,
		Later:         later,
		Winner:        earlier,
		LostUpdateFor: later.Value,
	}
}

type ProtectedResource struct {
	LastAppliedToken int64
	Value            string
}

func (r *ProtectedResource) Apply(token int64, value string) bool {
	if token <= r.LastAppliedToken {
		return false
	}

	r.LastAppliedToken = token
	r.Value = value
	return true
}

type LeaseWrite struct {
	Actor    string
	Token    int64
	Value    string
	Accepted bool
}

type LeaseScenario struct {
	Name            string
	AcceptedWrite   LeaseWrite
	RejectedWrite   LeaseWrite
	FinalValue      string
	FinalFenceToken int64
	WhyRejected     string
}

// ProcessPauseScenario shows a paused leaseholder becoming a zombie.
func ProcessPauseScenario() LeaseScenario {
	resource := &ProtectedResource{}
	accepted := LeaseWrite{Actor: "client-2", Token: 34, Value: "write from replacement leader"}
	accepted.Accepted = resource.Apply(accepted.Token, accepted.Value)

	rejected := LeaseWrite{Actor: "client-1", Token: 33, Value: "write from paused zombie"}
	rejected.Accepted = resource.Apply(rejected.Token, rejected.Value)

	return LeaseScenario{
		Name:            "process-pause zombie leaseholder",
		AcceptedWrite:   accepted,
		RejectedWrite:   rejected,
		FinalValue:      resource.Value,
		FinalFenceToken: resource.LastAppliedToken,
		WhyRejected:     "the paused node resumed after its lease expired, so fencing keeps the stale token from corrupting shared state",
	}
}

// DelayedRequestScenario shows an old request arriving after a new leaseholder took over.
func DelayedRequestScenario() LeaseScenario {
	resource := &ProtectedResource{}
	accepted := LeaseWrite{Actor: "client-2", Token: 41, Value: "write from current leaseholder"}
	accepted.Accepted = resource.Apply(accepted.Token, accepted.Value)

	rejected := LeaseWrite{Actor: "client-1", Token: 40, Value: "late write delayed in network"}
	rejected.Accepted = resource.Apply(rejected.Token, rejected.Value)

	return LeaseScenario{
		Name:            "delayed request from old leaseholder",
		AcceptedWrite:   accepted,
		RejectedWrite:   rejected,
		FinalValue:      resource.Value,
		FinalFenceToken: resource.LastAppliedToken,
		WhyRejected:     "the delayed request was sent before the lease expired but arrived after takeover, so fencing rejects it on arrival order rather than send time",
	}
}
