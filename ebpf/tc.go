//go:build linux

package ebpf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// TODO (bartsmykla): check if it's possible to replace execs of "sh -c tc [...]"
//  with some more idiomatic approach (maybe https://github.com/florianl/go-tc ?)

const clsact = "clsact"

// helper struct to parse "tc -json qdisc show [...]` results
type tcQdisc struct {
	Kind string `json:"kind"`
}

// TODO (bartsmykla): create common abstraction with run function from ebpf.go
func runTc(args ...string) (bytes.Buffer, error) {
	cmdWithArgs := append([]string{"tc"}, args...)
	tcCmd := strings.Join(cmdWithArgs, " ")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command("sh", "-c", tcCmd)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout, fmt.Errorf("executing %q failed with error: %q (stderr: %s)",
			tcCmd, err, strings.TrimSpace(stderr.String()))
	}

	return stdout, nil
}

func isQdiscPresent(qdisc, dev string) (bool, error) {
	var qdiscs []tcQdisc

	stdout, err := runTc("-json", "qdisc", "show", "dev", dev)
	if err != nil {
		return false, err
	}

	if err := json.NewDecoder(&stdout).Decode(&qdiscs); err != nil {
		return false, fmt.Errorf("json decoding failed: %v", err)
	}

	for _, tcQdisc := range qdiscs {
		if tcQdisc.Kind == qdisc {
			return true, nil
		}
	}

	return false, nil
}

// TODO (bartsmykla): currently we are assuming there is only one other than
//  loopback interface, and we are attaching eBPF programs only to it.
//  It's probably fine in the context of k8s and init containers,
//  but not for vms/universal
func getNonLoopbackInterface() (*net.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list network interfaces: %v", err)
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 {
			return &iface, nil
		}
	}

	return nil, fmt.Errorf("cannot find other than loopback interface")
}

// AttachTC will attach tc-related eBPF programs
func AttachTC(dev, obj string) error {
	hasClsact, err := isQdiscPresent(clsact, dev)
	if err != nil {
		return fmt.Errorf("checking if %q qdisc is already present failed: %v", clsact, err)
	}

	if !hasClsact {
		if _, err := runTc("qdisc", "add", "dev", dev, clsact); err != nil {
			return fmt.Errorf("adding %s qdisc to %s failed: %q", clsact, dev, err)
		}
	}

	if _, err := runTc("filter", "add", "prio", "66", "dev", dev, "ingress",
		"bpf", "da", "obj", obj, "sec", "classifier_ingress"); err != nil {
		return fmt.Errorf("failed to attach tc(ingress) to %s: %v", dev, err)
	}

	if _, err := runTc("filter", "add", "prio", "66", "dev", dev, "egress",
		"bpf", "da", "obj", obj, "sec", "classifier_egress"); err != nil {
		return fmt.Errorf("failed to attach tc(egress) to %s: %v", dev, err)
	}

	return nil
}

func CleanUpTC(dev string) error {
	hasClsact, err := isQdiscPresent(clsact, dev)
	if err != nil {
		return fmt.Errorf("checking if %q qdisc is already present failed: %v", clsact, err)
	}

	if hasClsact {
		if _, err := runTc("qdisc", "delete", "dev", dev, clsact); err != nil {
			return fmt.Errorf("failed to delete %s qdisc from %s: %v", clsact, dev, err)
		}

		return nil
	}

	if _, err := runTc("filter", "delete", "dev", dev, "egress", "prio", "66"); err != nil {
		return fmt.Errorf("failed to delete egress filter from %s: %v", dev, err)
	}

	if _, err := runTc("filter", "delete", "dev", dev, "ingress", "prio", "66"); err != nil {
		return fmt.Errorf("failed to delete ingress filter from %s: %v", dev, err)
	}

	return nil
}
