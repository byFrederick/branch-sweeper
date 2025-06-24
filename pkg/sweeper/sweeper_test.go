package sweeper

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/goombaio/namegenerator"
)

var defaultBaseBranch string = "main"

func TestSweeperWithNonExistingPath(t *testing.T) {
	path := randomName()

	options := SweeperOptions{
		Path:       path,
		StaleDays:  30,
		BaseBranch: defaultBaseBranch,
	}

	repoBranches, err := Sweeper(options)

	if len(repoBranches) != 0 && err == nil {
		t.Error("Expected empty result for non-existing path")
	}
}

func TestSweeperWithEmptyPath(t *testing.T) {
	options := SweeperOptions{
		Path:       t.TempDir(),
		StaleDays:  30,
		BaseBranch: defaultBaseBranch,
	}

	repoBranches, err := Sweeper(options)

	if len(repoBranches) != 0 && err == nil {
		t.Error("Expected empty result for empty path")
	}
}

func TestSweeperWithValidRepo(t *testing.T) {
	repo, path, hash := createTestRepo(t)
	staledBranch := randomName()
	_ = createTestBranch(t, repo, staledBranch, hash, time.Now().AddDate(0, 0, -30))

	options := SweeperOptions{
		Path:       path,
		StaleDays:  30,
		BaseBranch: "main",
	}

	repoBranches, err := Sweeper(options)

	if err != nil {
		t.Errorf("Sweeper returned error: %v", err)
	}

	if !slices.Contains(repoBranches[0], staledBranch) {
		t.Errorf("Expected branch %s", staledBranch)
	}
}

func TestBaseBranchWithValidBranch(t *testing.T) {
	repo, path, _ := createTestRepo(t)
	repoName := filepath.Base(path)

	branches, _ := repo.Branches()

	baseBranch, err := baseBranch(repoName, branches, defaultBaseBranch)

	if err != nil {
		t.Errorf("baseBranch returned error: %v", err)
	}

	if baseBranch.Name().Short() != defaultBaseBranch {
		t.Errorf("Expected base branch: %s", defaultBaseBranch)
	}
}

func TestBaseBranchWithInvalidBranch(t *testing.T) {
	repo, path, _ := createTestRepo(t)
	repoName := filepath.Base(path)

	branches, _ := repo.Branches()

	_, err := baseBranch(repoName, branches, "test")

	if err == nil {
		t.Errorf("Expected empty result")
	}
}

func TestIsStaleWithStaleBranch(t *testing.T) {
	repo, path, hash := createTestRepo(t)
	staleBranch := randomName()
	branch := createTestBranch(t, repo, staleBranch, hash, time.Now().AddDate(0, 0, -31))
	repoName := filepath.Base(path)

	staled, err := isStale(repoName, repo, branch, 30)

	if err != nil {
		t.Errorf("isStale returned error: %v", err)
	}

	if staled == false {
		t.Errorf("Expected isStale to be true")
	}
}

func TestIsStaleWithFreshBranch(t *testing.T) {
	repo, path, hash := createTestRepo(t)
	staleBranch := randomName()
	branch := createTestBranch(t, repo, staleBranch, hash, time.Now())
	repoName := filepath.Base(path)

	staled, err := isStale(repoName, repo, branch, 30)

	if err != nil {
		t.Errorf("isStale returned error: %v", err)
	}

	if staled == true {
		t.Errorf("Expected isStale to be false")
	}

}

func TestIsMergedWithMergedBranch(t *testing.T) {
	repo, path, hash := createTestRepo(t)
	branch := createTestBranch(t, repo, randomName(), hash, time.Now())
	mergedBranchWithBase(t, repo, branch)

	repoName := filepath.Base(path)

	baseBranch, err := repo.Reference(plumbing.NewBranchReferenceName(defaultBaseBranch), true)
	if err != nil {
		t.Fatalf("Error getting base branch reference: %v", err)
	}

	merged, err := isMerged(repoName, repo, baseBranch, branch)

	if err != nil {
		t.Errorf("isMerged returned error: %v", err)
	}

	if merged == false {
		t.Errorf("Expected isMerged to be true")
	}
}

func TestIsMergedWithUnmergedBranch(t *testing.T) {
	repo, path, hash := createTestRepo(t)
	branch := createTestBranch(t, repo, randomName(), hash, time.Now())
	repoName := filepath.Base(path)

	baseBranch, err := repo.Reference(plumbing.NewBranchReferenceName(defaultBaseBranch), true)
	if err != nil {
		t.Fatalf("Error getting base branch reference: %v", err)
	}

	merged, err := isMerged(repoName, repo, baseBranch, branch)

	if err != nil {
		t.Errorf("isMerged returned error: %v", err)
	}

	if merged == true {
		t.Errorf("Expected isMerged to be false")
	}
}

func TestDeleteBranch(t *testing.T) {
	repo, path, hash := createTestRepo(t)
	branchToBeDeleted := randomName()
	branch := createTestBranch(t, repo, branchToBeDeleted, hash, time.Now())
	repoName := filepath.Base(path)

	err := deleteBranch(repoName, repo, branch)
	if err != nil {
		t.Errorf("deleteBranch returned error: %v", err)
	}

	_, err = repo.Reference(plumbing.NewBranchReferenceName(branchToBeDeleted), true)
	if err != plumbing.ErrReferenceNotFound {
		t.Errorf("Expected branch %s to be removed: %v", branchToBeDeleted, err)
	}

}

func createTestRepo(t *testing.T) (*git.Repository, string, plumbing.Hash) {
	path := t.TempDir()
	defaultBranch := plumbing.NewBranchReferenceName(defaultBaseBranch)

	initOptions := git.PlainInitOptions{
		InitOptions: git.InitOptions{
			DefaultBranch: defaultBranch,
		},
		Bare: false}

	repo, err := git.PlainInitWithOptions(path, &initOptions)

	if err != nil {
		t.Errorf("Error creating test repo: %v", err)
	}

	worktree, _ := repo.Worktree()

	author := object.Signature{
		Name:  randomName(),
		Email: randomName() + "@test.com",
		When:  time.Now(),
	}

	commitOptions := git.CommitOptions{
		AllowEmptyCommits: true,
		Author:            &author,
	}

	hash, err := worktree.Commit(randomName(), &commitOptions)

	if err != nil {
		t.Errorf("Error creating initial commit: %v", err)
	}

	return repo, path, hash
}

func createTestBranch(t *testing.T, repo *git.Repository, branchName string, hash plumbing.Hash, date time.Time) *plumbing.Reference {
	branchRefName := plumbing.NewBranchReferenceName(branchName)

	worktree, err := repo.Worktree()

	if err != nil {
		t.Error("Error getting worktree")
	}

	checkoutOptions := git.CheckoutOptions{
		Hash:   hash,
		Branch: branchRefName,
		Create: true,
	}

	err = worktree.Checkout(&checkoutOptions)

	if err != nil {
		t.Error("Error checking out branch")
	}

	author := object.Signature{
		Name:  randomName(),
		Email: randomName() + "@test.com",
		When:  date,
	}

	commitOptions := git.CommitOptions{
		AllowEmptyCommits: true,
		Author:            &author,
	}

	_, err = worktree.Commit(randomName(), &commitOptions)

	if err != nil {
		t.Errorf("Error creating first commit of branch %s: %v", branchName, err)
	}

	branch, err := repo.Reference(branchRefName, true)

	if err != nil {
		t.Error("Error getting created branch reference")
	}

	return branch
}

func mergedBranchWithBase(t *testing.T, repo *git.Repository, branchToMerge *plumbing.Reference) {
	worktree, err := repo.Worktree()

	if err != nil {
		t.Error("Error getting worktree")
	}

	checkoutOptions := git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/heads/" + defaultBaseBranch),
	}

	err = worktree.Checkout(&checkoutOptions)

	if err != nil {
		t.Error("Error checking out base branch")
	}

	err = repo.Merge(*branchToMerge, git.MergeOptions{})

	if err != nil {
		t.Errorf("Error merging branch %s to base branch", branchToMerge.Name().Short())
	}

}

func randomName() string {
	seed := time.Now().UTC().UnixNano()
	nameGenerator := namegenerator.NewNameGenerator(seed)

	return nameGenerator.Generate()
}
