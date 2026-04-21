package tfc

import "time"

type Workspace struct {
	ID         string             `json:"id"`
	Attributes WorkspaceAttrs     `json:"attributes"`
	Relationships map[string]any  `json:"relationships"`
}

type WorkspaceAttrs struct {
	Name        string     `json:"name"`
	Locked      bool       `json:"locked"`
	LockedBy    string     `json:"locked-by"`
	CreatedAt   time.Time  `json:"created-at"`
	UpdatedAt   time.Time  `json:"updated-at"`
}

type StateVersion struct {
	ID         string            `json:"id"`
	Attributes StateVersionAttrs `json:"attributes"`
}

type StateVersionAttrs struct {
	Serial           int64     `json:"serial"`
	CreatedAt        time.Time `json:"created-at"`
	Status           string    `json:"status"`
	DownloadURL      string    `json:"hosted-state-download-url"`
	UploadURL        string    `json:"hosted-state-upload-url"`
	TerraformVersion string    `json:"terraform-version"`
	Lineage          string    `json:"lineage"`
	Finalized        bool      `json:"finalized"`
	CreatedBy        struct {
		Username string `json:"username"`
	} `json:"created-by"`
}

type AccountDetails struct {
	Data struct {
		ID         string `json:"id"`
		Attributes struct {
			Username        string `json:"username"`
			Email           string `json:"email"`
			TwoFactor       struct {
				Enabled bool `json:"enabled"`
			} `json:"two-factor"`
		} `json:"attributes"`
	} `json:"data"`
}
