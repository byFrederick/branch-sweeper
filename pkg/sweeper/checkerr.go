package sweeper

import "log"

func checkErrGetBranches(repo string, err error) {
	if err != nil {
		log.Fatalf("%s failed to get branches: %v", repo, err)
	}
}

func checkErrGetCommit(repo string, branch string, err error) {
	if err != nil {
		log.Fatalf("%s failed to get branch %s commits: %v", repo, branch, err)
	}
}
