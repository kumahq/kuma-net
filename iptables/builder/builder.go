package builder

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kumahq/kuma-net/iptables/config"
	"github.com/kumahq/kuma-net/iptables/table"
)

type IPTables struct {
	raw    *table.RawTable
	nat    *table.NatTable
	mangle *table.MangleTable
}

func newIPTables(
	raw *table.RawTable,
	nat *table.NatTable,
	mangle *table.MangleTable,
) *IPTables {
	return &IPTables{
		raw:    raw,
		nat:    nat,
		mangle: mangle,
	}
}

func (t *IPTables) Build(verbose bool) string {
	var tables []string

	raw := t.raw.Build(verbose)
	if raw != "" {
		tables = append(tables, raw)
	}

	nat := t.nat.Build(verbose)
	if nat != "" {
		tables = append(tables, nat)
	}

	mangle := t.mangle.Build(verbose)
	if mangle != "" {
		tables = append(tables, mangle)
	}

	separator := "\n"
	if verbose {
		separator = "\n\n"
	}

	return strings.Join(tables, separator) + "\n"
}

func BuildIPTables(cfg config.Config, dnsServers []string, ipv6 bool) (string, error) {
	cfg = config.MergeConfigWithDefaults(cfg)

	loopbackIface, err := getLoopback()
	if err != nil {
		return "", fmt.Errorf("cannot obtain loopback interface: %s", err)
	}

	return newIPTables(
		buildRawTable(cfg, dnsServers),
		buildNatTable(cfg, dnsServers, loopbackIface.Name, ipv6),
		buildMangleTable(cfg),
	).Build(cfg.Verbose), nil
}

// runtimeOutput is the file (should be os.Stdout by default) where we can dump generated
// rules for used to see and debug if something goes wrong, which can be overwritten
// in tests to not obfuscate the other, more relevant logs
func saveIPTablesRestoreFile(runtimeOutput io.Writer, f *os.File, content string) error {
	_, _ = fmt.Fprintln(runtimeOutput, "Writing following contents to rules file: ", f.Name())
	_, _ = fmt.Fprintln(runtimeOutput, content)

	writer := bufio.NewWriter(f)
	_, err := writer.WriteString(content)
	if err != nil {
		return fmt.Errorf("unable to write iptables-restore file: %s", err)
	}

	return writer.Flush()
}

func createRulesFile(ipv6 bool) (*os.File, error) {
	iptables := "iptables"
	if ipv6 {
		iptables = "ip6tables"
	}

	filename := fmt.Sprintf("%s-rules-%d.txt", iptables, time.Now().UnixNano())

	f, err := os.CreateTemp("", filename)
	if err != nil {
		return nil, fmt.Errorf("unable to create %s rules file: %s", iptables, err)
	}

	return f, nil
}

func runRestoreCmd(cmdName string, f *os.File) (string, error) {
	cmd := exec.Command(cmdName, "--noflush", f.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("executing command failed: %s (with output: %q)", err, output)
	}

	return string(output), nil
}

func restoreIPTables(cfg config.Config, dnsServers []string, ipv6 bool) (string, error) {
	cfg = config.MergeConfigWithDefaults(cfg)

	rulesFile, err := createRulesFile(cfg.IPv6)
	if err != nil {
		return "", err
	}
	defer rulesFile.Close()
	defer os.Remove(rulesFile.Name())

	rules, err := BuildIPTables(cfg, dnsServers, ipv6)
	if err != nil {
		return "", fmt.Errorf("unable to build iptable rules: %s", err)
	}

	if err := saveIPTablesRestoreFile(cfg.RuntimeOutput, rulesFile, rules); err != nil {
		return "", fmt.Errorf("unable to save iptables restore file: %s", err)
	}

	cmdName := "iptables-restore"
	if ipv6 {
		cmdName = "ip6tables-restore"
	}

	return runRestoreCmd(cmdName, rulesFile)
}

// RestoreIPTables
// TODO (bartsmykla): add validation if ip{,6}tables are available
func RestoreIPTables(cfg config.Config) (string, error) {
	var err error

	dnsIpv4, dnsIpv6 := []string{}, []string{}
	if cfg.ShouldRedirectDNS() && !cfg.ShouldCaptureAllDNS() {
		dnsIpv4, dnsIpv6, err = GetDnsServers(cfg.Redirect.DNS.ResolvConfigPath)
		if err != nil {
			return "", err
		}
	}

	output, err := restoreIPTables(cfg, dnsIpv4, false)
	if err != nil {
		return "", fmt.Errorf("cannot restore ipv4 iptable rules: %s", err)
	}

	if cfg.IPv6 {
		ipv6Output, err := restoreIPTables(cfg, dnsIpv6, true)
		if err != nil {
			return "", fmt.Errorf("cannot restore ipv6 iptable rules: %s", err)
		}

		output += ipv6Output
	}

	return output, nil
}
