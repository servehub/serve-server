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
}
