package models

type ManifestChanged struct {
	Manifest string `json:"manifest"`
	Repo     string `json:"repo"`
	Branch   string `json:"branch"`
	Purge    bool   `json:"purge,string"`
}
