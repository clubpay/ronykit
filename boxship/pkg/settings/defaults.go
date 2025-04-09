package settings

import (
	"time"
)

func PrepareDefaults(s Settings) Settings {
	s.SetDefault(WorkDir, "./_hdd")
	s.SetDefault(Setup, "./setup")
	s.SetDefault(LogAll, false)
	s.SetDefault(CACertFile, "./setup/rootca/ca.crt")
	s.SetDefault(CAKeyFile, "./setup/rootca/ca.key")
	s.SetDefault(ShallowClone, false)
	s.SetDefault(BuildContainerTimeout, time.Minute*15)
	s.SetDefault(BuildNetworkTimeout, time.Second*10)

	return s
}
