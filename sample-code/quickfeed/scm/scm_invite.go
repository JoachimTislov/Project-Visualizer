package scm

// InvitationOptions contains information on which organization and user to accept invitations for.
type InvitationOptions struct {
	Login        string // GitHub username.
	Owner        string // Name of the organization.
	RefreshToken string // Refresh token for the user.
}

func (opt InvitationOptions) valid() bool {
	return opt.Login != "" && opt.Owner != "" && opt.RefreshToken != ""
}
