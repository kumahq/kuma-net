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
	raw *table.RawTable
	nat *table.NatTable
}

func newIPTables(raw *table.RawTable, nat *table.NatTable) *IPTables {
	return &IPTables{
		raw: raw,
		nat: nat,
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

	separator := "\n"
	if verbose {
		separator = "\n\n"
	}

	return strings.Join(tables, separator) + "\n"
}

func BuildIPTables(config *config.Config) (string, error) {
	loopbackIface, err := getLoopback()
	if err != nil {
		return "", fmt.Errorf("cannot obtain loopback interface: %s", err)
	}

	return newIPTables(
		buildRawTable(config),
		buildNatTable(config, loopbackIface.Name),
	).Build(config.Verbose), nil
}

// infoOutput is the file (should be os.Stdout by default) where we can dump generated
// rules for used to see and debug if something goes wrong, which can be overwritten
// in tests to not obfuscate the other, more relevant logs
func saveIPTablesRestoreFile(infoOutput io.Writer, f *os.File, content string) error {
	defer f.Close()

	fmt.Fprintln(infoOutput, "Writing following contents to rules file: ", f.Name())
	fmt.Fprintln(infoOutput, content)

	writer := bufio.NewWriter(f)
	_, err := writer.WriteString(content)
	if err != nil {
		return fmt.Errorf("unable to write iptables-restore file: %s", err)
	}

	return writer.Flush()
}

func RestoreIPTables(config *config.Config) (string, error) {
	filename := fmt.Sprintf("iptables-rules-%d.txt", time.Now().UnixNano())
	rulesFile, err := os.CreateTemp("", filename)
	if err != nil {
		return "", fmt.Errorf("unable to create iptables-restore file: %s", err)
	}
	defer os.Remove(rulesFile.Name())

	rules, err := BuildIPTables(config)
	if err != nil {
		return "", fmt.Errorf("unable to build iptable rules: %s", err)
	}

	if err := saveIPTablesRestoreFile(config.Output, rulesFile, rules); err != nil {
		return "", fmt.Errorf("unable to save iptables restore file: %s", err)
	}

	cmd := exec.Command("iptables-restore", "--noflush", rulesFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("executing command failed: %s (with output: %q)", err, output)
	}

	return string(output), nil
}
