package quickget

import (
	"os/exec"
	"strconv"
	"strings"
)

type PVEItem struct {
	VMID       int
	Name       string
	Status     string
	MemMB      int
	BootDiskMB int
	PID        int
}

func QMList() ([]*PVEItem, error) {
	out, err := exec.Command("qm", "list").Output()
	if err != nil {
		return nil, err
	}
	return parseQMList(out)
}

func parseQMList(out []byte) ([]*PVEItem, error) {
	lines := strings.Split(string(out), "\n")
	items := make([]*PVEItem, 0, len(lines))
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		l := len(fields)
		vmid, _ := strconv.Atoi(fields[0])
		name := strings.Join(fields[1:l-4], " ")
		status := fields[l-4]
		memMB, _ := strconv.Atoi(fields[l-3])
		bootDiskGB, _ := strconv.ParseFloat(fields[l-2], 64)
		pid, _ := strconv.Atoi(fields[l-1])
		vm := &PVEItem{
			VMID:       vmid,
			Name:       name,
			Status:     status,
			MemMB:      memMB,
			BootDiskMB: int(bootDiskGB * 1024),
			PID:        pid,
		}
		items = append(items, vm)
	}
	return items, nil
}

func DiskStatus() ([]string, error) {
	out, err := exec.Command("pvesm", "status").Output()
	if err != nil {
		return nil, err
	}
	return parsePVEStatus(out)
}

func parsePVEStatus(out []byte) ([]string, error) {
	lines := strings.Split(string(out), "\n")
	items := make([]string, 0, len(lines))
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}
		items = append(items, fields[0])
	}
	return items, nil
}
