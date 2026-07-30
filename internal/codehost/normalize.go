package codehost

import (
	"fmt"
	"strings"

	"github.com/hero-engine/hero/contracts/codehostbroker"
)

type githubUser struct {
	ID    int64  `json:"id"`
	Node  string `json:"node_id"`
	Login string `json:"login"`
	Name  string `json:"name"`
}

type githubRepository struct {
	ID       int64      `json:"id"`
	Node     string     `json:"node_id"`
	Name     string     `json:"name"`
	FullName string     `json:"full_name"`
	Owner    githubUser `json:"owner"`
}

type githubPullRef struct {
	Ref  string           `json:"ref"`
	SHA  string           `json:"sha"`
	Repo githubRepository `json:"repo"`
}

type githubPullRequest struct {
	ID             int64         `json:"id"`
	Node           string        `json:"node_id"`
	Number         int64         `json:"number"`
	Title          string        `json:"title"`
	Body           string        `json:"body"`
	HTMLURL        string        `json:"html_url"`
	State          string        `json:"state"`
	Draft          bool          `json:"draft"`
	Merged         bool          `json:"merged"`
	MergeCommitSHA string        `json:"merge_commit_sha"`
	User           githubUser    `json:"user"`
	Base           githubPullRef `json:"base"`
	Head           githubPullRef `json:"head"`
	CreatedAt      string        `json:"created_at"`
	UpdatedAt      string        `json:"updated_at"`
	MergedAt       string        `json:"merged_at"`
}

type githubCommit struct {
	SHA     string     `json:"sha"`
	HTMLURL string     `json:"html_url"`
	Author  githubUser `json:"author"`
	Commit  struct {
		Message string `json:"message"`
		Author  struct {
			Name string `json:"name"`
			Date string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

type githubDiffFile struct {
	Filename  string  `json:"filename"`
	Status    string  `json:"status"`
	Additions int     `json:"additions"`
	Deletions int     `json:"deletions"`
	Patch     *string `json:"patch"`
}

type githubCheckRuns struct {
	TotalCount int              `json:"total_count"`
	CheckRuns  []githubCheckRun `json:"check_runs"`
}

type githubCheckRun struct {
	ID         int64  `json:"id"`
	Node       string `json:"node_id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	HTMLURL    string `json:"html_url"`
}

type githubReview struct {
	ID          int64      `json:"id"`
	Node        string     `json:"node_id"`
	User        githubUser `json:"user"`
	Body        string     `json:"body"`
	State       string     `json:"state"`
	CommitID    string     `json:"commit_id"`
	SubmittedAt string     `json:"submitted_at"`
}

type githubComment struct {
	ID        int64      `json:"id"`
	Node      string     `json:"node_id"`
	User      githubUser `json:"user"`
	Body      string     `json:"body"`
	HTMLURL   string     `json:"html_url"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
}

type githubGraphQLResponse struct {
	Data struct {
		Repository struct {
			PullRequest githubGraphQLPullRequest `json:"pullRequest"`
		} `json:"repository"`
		RateLimit githubGraphQLRateLimit `json:"rateLimit"`
	} `json:"data"`
	Errors []githubGraphQLError `json:"errors"`
}

type githubGraphQLError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Path    []any  `json:"path"`
}

type githubGraphQLRateLimit struct {
	Limit     int64  `json:"limit"`
	Remaining int64  `json:"remaining"`
	ResetAt   string `json:"resetAt"`
}

type githubGraphQLPullRequest struct {
	ID               string `json:"id"`
	HeadRefOID       string `json:"headRefOid"`
	BaseRefOID       string `json:"baseRefOid"`
	IsDraft          bool   `json:"isDraft"`
	Mergeable        string `json:"mergeable"`
	MergeStateStatus string `json:"mergeStateStatus"`
	ReviewDecision   string `json:"reviewDecision"`
	ViewerCanMerge   *bool  `json:"viewerCanMerge"`
	MergeQueueEntry  *struct {
		ID string `json:"id"`
	} `json:"mergeQueueEntry"`
	MergeQueue *struct {
		ID string `json:"id"`
	} `json:"mergeQueue"`
	BaseRef struct {
		BranchProtectionRule *struct {
			RequiresApprovingReviews     bool `json:"requiresApprovingReviews"`
			RequiredApprovingReviewCount int  `json:"requiredApprovingReviewCount"`
			RequiresStatusChecks         bool `json:"requiresStatusChecks"`
		} `json:"branchProtectionRule"`
	} `json:"baseRef"`
	Commits struct {
		Nodes []struct {
			Commit struct {
				StatusCheckRollup *struct {
					State string `json:"state"`
				} `json:"statusCheckRollup"`
			} `json:"commit"`
		} `json:"nodes"`
	} `json:"commits"`
}

func (p githubGraphQLPullRequest) CheckState() string {
	if len(p.Commits.Nodes) == 0 || p.Commits.Nodes[0].Commit.StatusCheckRollup == nil {
		return ""
	}
	return p.Commits.Nodes[0].Commit.StatusCheckRollup.State
}

func normalizePullRequest(connectionID string, item githubPullRequest) codehostbroker.PullRequest {
	state := strings.ToLower(item.State)
	if item.Merged {
		state = "merged"
	}
	return codehostbroker.PullRequest{
		Identity: codehostbroker.PullRequestIdentity{
			ConnectionID: connectionID,
			Repository:   normalizeRepository(item.Base.Repo),
			ProviderID:   providerID(item.Node, item.ID),
			Number:       item.Number,
		},
		Title:     boundedText(item.Title, codehostbroker.MaxTextBytes),
		Body:      boundedText(item.Body, codehostbroker.MaxBodyBytes),
		URL:       boundedText(item.HTMLURL, 2048),
		State:     boundedText(state, 128),
		Draft:     item.Draft,
		Author:    normalizeActor(item.User),
		Base:      codehostbroker.RefIdentity{Repository: normalizeRepository(item.Base.Repo), Name: boundedText(item.Base.Ref, 1024), SHA: boundedText(item.Base.SHA, 128)},
		Head:      codehostbroker.RefIdentity{Repository: normalizeRepository(item.Head.Repo), Name: boundedText(item.Head.Ref, 1024), SHA: boundedText(item.Head.SHA, 128)},
		CreatedAt: boundedText(item.CreatedAt, 64),
		UpdatedAt: boundedText(item.UpdatedAt, 64),
		MergedAt:  boundedText(item.MergedAt, 64),
	}
}

func normalizeRepository(repository githubRepository) codehostbroker.RepositoryIdentity {
	owner := repository.Owner.Login
	name := repository.Name
	fullName := repository.FullName
	if fullName != "" {
		if before, after, ok := strings.Cut(fullName, "/"); ok {
			owner, name = before, after
		}
	}
	return codehostbroker.RepositoryIdentity{
		Host:       repositoryHostFromURL(repository),
		ProviderID: providerID(repository.Node, repository.ID),
		Owner:      owner,
		Name:       name,
		FullName:   owner + "/" + name,
	}
}

// GitHub's REST repository payload does not carry the web host. github.com is
// the correct host for public API responses; Enterprise payloads are adjusted
// by the adapter before contract validation.
func repositoryHostFromURL(githubRepository) string { return "github.com" }

func normalizeCommit(item githubCommit) codehostbroker.Commit {
	author := normalizeActor(item.Author)
	if author.Login == "" {
		author.Login = item.Commit.Author.Name
		author.Display = item.Commit.Author.Name
	}
	return codehostbroker.Commit{
		SHA:        boundedText(item.SHA, 128),
		Message:    boundedText(item.Commit.Message, codehostbroker.MaxTextBytes),
		Author:     author,
		AuthoredAt: boundedText(item.Commit.Author.Date, 64),
		URL:        boundedText(item.HTMLURL, 2048),
	}
}

func normalizeCheck(item githubCheckRun) codehostbroker.Check {
	availability := codehostbroker.AvailabilityAvailable
	if item.Status == "" {
		availability = codehostbroker.AvailabilityUnknown
	}
	return codehostbroker.Check{
		ProviderID:   providerID(item.Node, item.ID),
		Name:         boundedText(item.Name, codehostbroker.MaxTextBytes),
		Status:       boundedText(item.Status, 128),
		Conclusion:   boundedText(item.Conclusion, 128),
		URL:          boundedText(item.HTMLURL, 2048),
		Availability: availability,
	}
}

func normalizeReview(item githubReview) codehostbroker.Review {
	return codehostbroker.Review{
		ProviderID:  providerID(item.Node, item.ID),
		Author:      normalizeActor(item.User),
		State:       boundedText(strings.ToLower(item.State), 128),
		Body:        boundedText(stripHeroMarkers(item.Body), codehostbroker.MaxBodyBytes),
		HeadSHA:     boundedText(item.CommitID, 128),
		SubmittedAt: boundedText(item.SubmittedAt, 64),
	}
}

func normalizeComment(item githubComment) codehostbroker.Comment {
	return codehostbroker.Comment{
		ProviderID: providerID(item.Node, item.ID),
		Author:     normalizeActor(item.User),
		Body:       boundedText(stripHeroMarkers(item.Body), codehostbroker.MaxBodyBytes),
		URL:        boundedText(item.HTMLURL, 2048),
		CreatedAt:  boundedText(item.CreatedAt, 64),
		UpdatedAt:  boundedText(item.UpdatedAt, 64),
	}
}

func normalizeActor(user githubUser) codehostbroker.Actor {
	return codehostbroker.Actor{
		ProviderID: providerID(user.Node, user.ID),
		Login:      boundedText(user.Login, 255),
		Display:    boundedText(user.Name, codehostbroker.MaxTextBytes),
	}
}

func providerID(node string, numeric int64) string {
	if node != "" {
		return boundedText(node, 512)
	}
	if numeric != 0 {
		return fmt.Sprintf("%d", numeric)
	}
	return ""
}
