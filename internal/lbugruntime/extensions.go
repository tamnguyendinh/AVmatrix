package lbugruntime

import "fmt"

type ExecRunner interface {
	Exec(query string) error
}

type ExtensionState struct {
	FTSLoaded    bool
	VectorLoaded bool
}

func (s *ExtensionState) EnsureFTS(runner ExecRunner) error {
	if s.FTSLoaded {
		return nil
	}
	if runner == nil {
		return fmt.Errorf("extension runner is nil")
	}
	if err := runner.Exec("LOAD EXTENSION fts"); err == nil || IsAlreadyLoadedOrInstalledError(err) {
		s.FTSLoaded = true
		return nil
	}
	if err := runner.Exec("INSTALL fts"); err != nil && !IsAlreadyLoadedOrInstalledError(err) {
		return err
	}
	if err := runner.Exec("LOAD EXTENSION fts"); err != nil && !IsAlreadyLoadedOrInstalledError(err) {
		return err
	}
	s.FTSLoaded = true
	return nil
}

func (s *ExtensionState) EnsureVector(runner ExecRunner) error {
	if s.VectorLoaded {
		return nil
	}
	if runner == nil {
		return fmt.Errorf("extension runner is nil")
	}
	if err := runner.Exec("INSTALL VECTOR"); err != nil && !IsAlreadyLoadedOrInstalledError(err) {
		return err
	}
	if err := runner.Exec("LOAD EXTENSION VECTOR"); err != nil && !IsAlreadyLoadedOrInstalledError(err) {
		return err
	}
	s.VectorLoaded = true
	return nil
}
