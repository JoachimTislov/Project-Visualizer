package qf

func (pr *PullRequest) SetApproved() {
	pr.Stage = PullRequest_APPROVED
}

func (pr *PullRequest) SetReview() {
	pr.Stage = PullRequest_REVIEW
}

func (pr *PullRequest) SetDraft() {
	pr.Stage = PullRequest_DRAFT
}

// IsApproved returns true if a pull request is at the approved stage.
func (pr *PullRequest) IsApproved() bool {
	return pr.Stage == PullRequest_APPROVED
}

// HasReviewers returns true if a pull request is in the approved or review stage.
// This implies that it should have had reviewers assigned to it.
func (pr *PullRequest) HasReviewers() bool {
	return pr.Stage == PullRequest_APPROVED || pr.Stage == PullRequest_REVIEW
}

// HasFeedbackComment returns true if the pull request has a comment associated with it.
func (pr *PullRequest) HasFeedbackComment() bool {
	return pr.ScmCommentID > 0
}

// Checks if a pull request is valid for creation.
func (pr *PullRequest) Valid() bool {
	return pr.ScmRepositoryID > 0 && pr.TaskID > 0 &&
		pr.IssueID > 0 && pr.SourceBranch != "" && pr.Number > 0 &&
		pr.UserID > 0
}
