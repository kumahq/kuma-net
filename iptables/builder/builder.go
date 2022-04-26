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

func BuildIPTables(cfg config.Config) (string, error) {
	cfg = config.MergeConfigWithDefaults(cfg)

	loopbackIface, err := getLoopback()
	if err != nil {
		return "", fmt.Errorf("cannot obtain loopback interface: %s", err)
	}

	return newIPTables(
		buildRawTable(cfg),
		buildNatTable(cfg, loopbackIface.Name),
		buildMangleTable(cfg),
	).Build(cfg.Verbose), nil
}

// runtimeOutput is the file (should be os.Stdout by default) where we can dump generated
// rules for used to see and debug if something goes wrong, which can be overwritten
// in tests to not obfuscate the other, more relevant logs
func saveIPTablesRestoreFile(runtimeOutput io.Writer, f *os.File, content string) error {
	defer f.Close()

	_, _ = fmt.Fprintln(runtimeOutput, "Writing following contents to rules file: ", f.Name())
	_, _ = fmt.Fprintln(runtimeOutput, content)

	writer := bufio.NewWriter(f)
	_, err := writer.WriteString(content)
	if err != nil {
		return fmt.Errorf("unable to write iptables-restore file: %s", err)
	}

	return writer.Flush()
}

func RestoreIPTables(cfg config.Config) (string, error) {
	cfg = config.MergeConfigWithDefaults(cfg)

	filename := fmt.Sprintf("iptables-rules-%d.txt", time.Now().UnixNano())
	rulesFile, err := os.CreateTemp("", filename)
	if err != nil {
		return "", fmt.Errorf("unable to create iptables-restore file: %s", err)
	}
	defer os.Remove(rulesFile.Name())

	rules, err := BuildIPTables(cfg)
	if err != nil {
		return "", fmt.Errorf("unable to build iptable rules: %s", err)
	}

	if err := saveIPTablesRestoreFile(cfg.RuntimeOutput, rulesFile, rules); err != nil {
		return "", fmt.Errorf("unable to save iptables restore file: %s", err)
	}

	cmd := exec.Command("iptables-restore", "--noflush", rulesFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("executing command failed: %s (with output: %q)", err, output)
	}

	return string(output), nil
}
