package golden

import (
	"os"
)

func UpdateGoldenFiles() bool {
	return os.Getenv("UPDATE_GOLDEN_FILES") == "true"
}

const RerunMsg = "Rerun the test with UPDATE_GOLDEN_FILES=true flag. Example: " +
	"UPDATE_GOLDEN_FILES=true ginkgo"
