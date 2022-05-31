package models

type ManifestChanged struct {
	Manifest string `json:"manifest"`
	Repo     string `json:"repo"`
	Branch   string `json:"branch"`
	Purge    bool   `json:"purge,string"`
}

type CodeUpdated struct {
	Repo       string `json:"repo"`
	Branch     string `json:"branch"`
	Commit     string `json:"commit"`
	PrevCommit string `json:"prevCommit"`
	Purge      bool   `json:"purge,string"`
}

type FeatureBranchRemoved struct {
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
}

type ServiceDeployData struct {
	AppName string `json:"app-name"`
	GitRepo string `json:"git-repo"`
	Branch  string `json:"branch"`
}
