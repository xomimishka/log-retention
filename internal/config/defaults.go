package config

const (
	DefaultGroupBy      = "dir"
	DefaultCompress     = "zip"
	DefaultLevel        = 9
	DefaultArchiveName  = "{group}-{date}.zip"
	DefaultAfterArchive = "delete"
)

func (c *Config) ApplyDefaults() {
	for i := range c.Policies {
		applyPolicyDefaults(&c.Policies[i])
	}
}

func applyPolicyDefaults(p *Policy) {
	if p.Group.By == "" {
		p.Group.By = DefaultGroupBy
	}
	if p.AfterArchive == "" {
		p.AfterArchive = DefaultAfterArchive
	}
	if p.Archive.Compress == "" {
		p.Archive.Compress = DefaultCompress
	}
	if p.Archive.Level == nil {
		p.Archive.Level = intPtr(DefaultLevel)
	}
	if p.Archive.Name == "" {
		p.Archive.Name = DefaultArchiveName
	}
}

func intPtr(v int) *int { return &v }
