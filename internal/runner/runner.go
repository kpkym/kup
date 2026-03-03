package runner

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/kpkym/kup/internal/config"
)

// RunRestic executes restic with the given args against a single repo.
func RunRestic(global config.GlobalConfig, repo string, args []string) error {
	if _, err := exec.LookPath("restic"); err != nil {
		return fmt.Errorf("restic not found in PATH; install from https://restic.net")
	}

	cmd := exec.Command("restic", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = resticEnv(global, repo)

	return runWithSignalForward(cmd)
}

// RunRclone executes rclone with the given args.
func RunRclone(global config.GlobalConfig, args []string) error {
	if _, err := exec.LookPath("rclone"); err != nil {
		return fmt.Errorf("rclone not found in PATH; install from https://rclone.org")
	}

	cmd := exec.Command("rclone", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = rcloneEnv(global)

	return runWithSignalForward(cmd)
}

// RunResticForEachRepo runs restic against each repo sequentially.
// Returns an error summarizing any failures.
func RunResticForEachRepo(global config.GlobalConfig, repos []string, args []string) error {
	var failed []string

	for _, repo := range repos {
		fmt.Fprintf(os.Stderr, "=== %s ===\n", repo)

		if err := RunRestic(global, repo, args); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR [%s]: %v\n", repo, err)
			failed = append(failed, repo)
		}

		fmt.Fprintln(os.Stderr)
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed for %d repo(s): %v", len(failed), failed)
	}
	return nil
}

func resticEnv(global config.GlobalConfig, repo string) []string {
	env := os.Environ()
	env = append(env,
		"RESTIC_REPOSITORY="+repo,
		"RCLONE_CONFIG="+global.RcloneConfig,
	)
	env = append(env, "RESTIC_PASSWORD="+global.ResticPassword)
	if global.ResticPackSize > 0 {
		env = append(env, "RESTIC_PACK_SIZE="+strconv.Itoa(global.ResticPackSize))
	}
	if global.RcloneNoCheckCertificate {
		env = append(env, "RCLONE_NO_CHECK_CERTIFICATE=true")
	}
	return env
}

func rcloneEnv(global config.GlobalConfig) []string {
	env := os.Environ()
	env = append(env, "RCLONE_CONFIG="+global.RcloneConfig)
	if global.RcloneNoCheckCertificate {
		env = append(env, "RCLONE_NO_CHECK_CERTIFICATE=true")
	}
	return env
}

func runWithSignalForward(cmd *exec.Cmd) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting process: %w", err)
	}

	go func() {
		sig := <-sigCh
		if cmd.Process != nil {
			cmd.Process.Signal(sig)
		}
	}()

	return cmd.Wait()
}
