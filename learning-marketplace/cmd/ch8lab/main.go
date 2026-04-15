package main

import (
	"fmt"

	"learning-marketplace/internal/distsys"
)

func main() {
	clockSkewDemo()
	leasePauseDemo()
	delayedRequestDemo()
}

func clockSkewDemo() {
	scenario := distsys.ClockSkewScenario()

	fmt.Println("== Clock skew can break last-write-wins ==")
	fmt.Printf("causal order: %s then %s\n", scenario.Earlier.Value, scenario.Later.Value)
	fmt.Printf("timestamps : %s@%d, %s@%d\n", scenario.Earlier.Value, scenario.Earlier.TimestampMS, scenario.Later.Value, scenario.Later.TimestampMS)
	fmt.Printf("lww winner : %s (later write %s is lost)\n", scenario.Winner.Value, scenario.LostUpdateFor)
	fmt.Println("lesson     : wall-clock timestamps are not a safe ordering mechanism in distributed systems")
	fmt.Println()
}

func leasePauseDemo() {
	scenario := distsys.ProcessPauseScenario()
	printLeaseScenario(scenario)
}

func delayedRequestDemo() {
	scenario := distsys.DelayedRequestScenario()
	printLeaseScenario(scenario)
}

func printLeaseScenario(scenario distsys.LeaseScenario) {
	fmt.Printf("== %s ==\n", scenario.Name)
	fmt.Printf("accepted write: actor=%s token=%d accepted=%t value=%q\n", scenario.AcceptedWrite.Actor, scenario.AcceptedWrite.Token, scenario.AcceptedWrite.Accepted, scenario.AcceptedWrite.Value)
	fmt.Printf("rejected write: actor=%s token=%d accepted=%t value=%q\n", scenario.RejectedWrite.Actor, scenario.RejectedWrite.Token, scenario.RejectedWrite.Accepted, scenario.RejectedWrite.Value)
	fmt.Printf("final state   : token=%d value=%q\n", scenario.FinalFenceToken, scenario.FinalValue)
	fmt.Printf("lesson        : %s\n", scenario.WhyRejected)
	fmt.Println()
}
