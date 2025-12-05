package utils

import (
	"log"
	"os"
	"spm/pkg/config"
	"spm/pkg/logger"
	"spm/pkg/utils/constants"
)

func setup() {
	config.SetConfig(GlobalConfigFile)

	_, err := os.Stat(constants.SpmHome)
	if err != nil {
		log.Printf("Try to create %s directory\n", constants.SpmHome)
		if err := os.MkdirAll(constants.SpmHome, 0755); err != nil {
			log.Fatalf("create directory %q error: %v", constants.SpmHome, err)
		}
	}

	_ = CheckPerm(constants.SpmHome)

	if config.ForegroundFlag {
		config.GetConfig().Log.FileEnabled = false
	}

	logger.InitLogger()
}

func InitEnv() {
	setup()
}
