package cli

import (
	"fmt"
	"k8s.io/utils/exec"
	"os"
	"path/filepath"
)

func (r *Restic) DoCITAStateBackup(blockHeight int64, nodeRoot string, configPath string, backupPath string, crypto string, consensus string) error {
	execer := exec.New()
	err := execer.Command("cloud-op", "state-backup", fmt.Sprintf("%d", blockHeight),
		"--node-root", nodeRoot,
		"--config-path", fmt.Sprintf("%s/config.toml", configPath),
		"--backup-path", backupPath,
		"--crypto", crypto,
		"--consensus", consensus).Run()
	if err != nil {
		return err
	}
	return nil
}

func (r *Restic) DoCITAStateRecover(blockHeight int64, nodeRoot string, configPath string, backupPath string,
	crypto string, consensus string, deleteConsensusData bool) error {
	execer := exec.New()
	args := []string{"state-recover", fmt.Sprintf("%d", blockHeight),
		"--node-root", nodeRoot,
		"--config-path", fmt.Sprintf("%s/config.toml", configPath),
		"--backup-path", backupPath,
		"--crypto", crypto,
		"--consensus", consensus,
	}
	if deleteConsensusData {
		args = append(args, "--is-clear")
	}
	err := execer.Command("cloud-op", args...).Run()
	if err != nil {
		return err
	}
	return filepath.Walk(nodeRoot, func(name string, info os.FileInfo, err error) error {
		if err == nil {
			err = os.Chown(name, 1000, 1000)
		}
		return err
	})
}
